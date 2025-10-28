package network

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	eve "eve.evalgo.org/common"
)

// ZitiProxy represents the main proxy server
type ZitiProxy struct {
	config        *ProxyConfig
	router        *Router
	loadBalancers map[string]*LoadBalancer
	server        *http.Server
	mu            sync.RWMutex
}

// NewZitiProxy creates a new Ziti proxy server from configuration
func NewZitiProxy(configPath string) (*ZitiProxy, error) {
	// Load configuration
	config, err := LoadProxyConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create router
	router := NewRouter(config)

	// Initialize load balancers for each route
	loadBalancers := make(map[string]*LoadBalancer)
	for i := range config.Routes {
		route := &config.Routes[i]
		lb, err := NewLoadBalancer(route)
		if err != nil {
			return nil, fmt.Errorf("failed to create load balancer for route %s: %w", route.Path, err)
		}
		loadBalancers[route.Path] = lb
	}

	proxy := &ZitiProxy{
		config:        config,
		router:        router,
		loadBalancers: loadBalancers,
	}

	return proxy, nil
}

// Start starts the proxy server
func (zp *ZitiProxy) Start() error {
	// Build middleware chain
	var middlewares []Middleware

	// Add recovery middleware first (catches panics from all other middleware)
	middlewares = append(middlewares, RecoveryMiddleware())

	// Add logging middleware
	if zp.config.Logging != nil && zp.config.Logging.Enabled {
		middlewares = append(middlewares, LoggingMiddleware(zp.config.Logging))
	}

	// Add CORS middleware
	if zp.config.CORS != nil && zp.config.CORS.Enabled {
		middlewares = append(middlewares, CORSMiddleware(zp.config.CORS))
	}

	// Add global auth middleware if configured
	if zp.config.Auth != nil && zp.config.Auth.Type != "none" {
		middlewares = append(middlewares, AuthMiddleware(zp.config.Auth))
	}

	// Create main handler
	handler := ChainMiddleware(http.HandlerFunc(zp.handleRequest), middlewares...)

	// Create server
	addr := fmt.Sprintf("%s:%d", zp.config.Server.Host, zp.config.Server.Port)
	zp.server = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  zp.config.Server.ReadTimeout.Duration,
		WriteTimeout: zp.config.Server.WriteTimeout.Duration,
		IdleTimeout:  zp.config.Server.IdleTimeout.Duration,
	}

	eve.Logger.Info(fmt.Sprintf("Starting Ziti Proxy on %s", addr))

	// Start server
	if err := zp.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Stop gracefully stops the proxy server
func (zp *ZitiProxy) Stop(ctx context.Context) error {
	eve.Logger.Info("Stopping Ziti Proxy...")

	// Stop all load balancers and health checkers
	zp.mu.Lock()
	for _, lb := range zp.loadBalancers {
		lb.Stop()
	}
	zp.mu.Unlock()

	// Shutdown HTTP server
	if zp.server != nil {
		return zp.server.Shutdown(ctx)
	}

	return nil
}

// handleRequest handles incoming HTTP requests
func (zp *ZitiProxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Match route
	match := zp.router.Match(r)
	if match == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Apply route-specific auth if configured
	if match.Route.Auth != nil && match.Route.Auth.Type != "none" {
		authMiddleware := AuthMiddleware(match.Route.Auth)
		handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			zp.proxyRequest(w, r, match)
		}))
		handler.ServeHTTP(w, r)
		return
	}

	// Proxy the request
	zp.proxyRequest(w, r, match)
}

// proxyRequest proxies the request to a backend service
func (zp *ZitiProxy) proxyRequest(w http.ResponseWriter, r *http.Request, match *RouteMatch) {
	// Get load balancer for this route
	zp.mu.RLock()
	lb, exists := zp.loadBalancers[match.Route.Path]
	zp.mu.RUnlock()

	if !exists || lb == nil {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	// Select backend
	backend := lb.SelectBackend()
	if backend == nil {
		http.Error(w, "No Healthy Backends Available", http.StatusServiceUnavailable)
		return
	}

	// Track connection
	lb.IncrementConnections(backend)
	defer lb.DecrementConnections(backend)

	// Rewrite path
	originalPath := r.URL.Path
	r.URL.Path = RewritePath(originalPath, match.Route, match.Params)

	// Set up retry logic
	var lastErr error
	maxRetries := backend.Config.MaxRetries
	if match.Route.Retry != nil {
		maxRetries = match.Route.Retry.MaxAttempts
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt) * time.Second
			if match.Route.Retry != nil {
				backoff = match.Route.Retry.InitialInterval.Duration * time.Duration(attempt)
			}
			time.Sleep(backoff)

			eve.Logger.Info(fmt.Sprintf("Retrying request (attempt %d/%d)", attempt, maxRetries))
		}

		// Create proxied request with proper URL
		// Include port if specified and not default (80)
		host := backend.Config.ZitiService
		if backend.Config.Port > 0 && backend.Config.Port != 80 {
			host = fmt.Sprintf("%s:%d", backend.Config.ZitiService, backend.Config.Port)
		}

		targetURL := fmt.Sprintf("http://%s%s", host, r.URL.Path)
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}

		proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
		if err != nil {
			lastErr = err
			continue
		}

		// Copy headers
		for key, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}

		// Rewrite host header if configured
		if match.Route.RewriteHost {
			proxyReq.Host = backend.Config.ZitiService
			proxyReq.Header.Set("Host", backend.Config.ZitiService)
		}

		// Add X-Forwarded headers
		proxyReq.Header.Set("X-Forwarded-For", r.RemoteAddr)
		proxyReq.Header.Set("X-Forwarded-Proto", "http")
		proxyReq.Header.Set("X-Forwarded-Host", r.Host)

		// Get client (lazy initialization on first use)
		client, err := backend.GetClient()
		if err != nil {
			lastErr = err
			lb.RecordFailure(backend)
			continue
		}

		// Execute request
		resp, err := client.Do(proxyReq)
		if err != nil {
			lastErr = err
			lb.RecordFailure(backend)
			continue
		}

		// Check if response should be retried
		if match.Route.Retry != nil && zp.shouldRetry(resp.StatusCode, match.Route.Retry) {
			resp.Body.Close()
			lastErr = fmt.Errorf("retryable status code: %d", resp.StatusCode)
			lb.RecordFailure(backend)
			continue
		}

		// Success - copy response
		lb.RecordSuccess(backend)

		// Copy response headers
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// Write status code
		w.WriteHeader(resp.StatusCode)

		// Copy response body
		io.Copy(w, resp.Body)
		resp.Body.Close()

		return
	}

	// All retries failed
	eve.Logger.Error(fmt.Sprintf("All retries failed for %s: %v", r.URL.Path, lastErr))
	http.Error(w, "Bad Gateway", http.StatusBadGateway)
}

// shouldRetry determines if a request should be retried based on status code
func (zp *ZitiProxy) shouldRetry(statusCode int, retry *RetryConfig) bool {
	if retry == nil || len(retry.RetryableStatus) == 0 {
		// Default retryable status codes
		return statusCode >= 500 && statusCode < 600
	}

	for _, code := range retry.RetryableStatus {
		if statusCode == code {
			return true
		}
	}

	return false
}

// Reload reloads the proxy configuration without stopping the server
func (zp *ZitiProxy) Reload(configPath string) error {
	eve.Logger.Info("Reloading configuration...")

	// Load new configuration
	newConfig, err := LoadProxyConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load new config: %w", err)
	}

	// Create new router
	newRouter := NewRouter(newConfig)

	// Create new load balancers
	newLoadBalancers := make(map[string]*LoadBalancer)
	for i := range newConfig.Routes {
		route := &newConfig.Routes[i]
		lb, err := NewLoadBalancer(route)
		if err != nil {
			return fmt.Errorf("failed to create load balancer for route %s: %w", route.Path, err)
		}
		newLoadBalancers[route.Path] = lb
	}

	// Stop old load balancers
	zp.mu.Lock()
	for _, lb := range zp.loadBalancers {
		lb.Stop()
	}

	// Swap to new configuration
	zp.config = newConfig
	zp.router = newRouter
	zp.loadBalancers = newLoadBalancers
	zp.mu.Unlock()

	eve.Logger.Info("Configuration reloaded successfully")
	return nil
}

// GetStatus returns the current status of the proxy
func (zp *ZitiProxy) GetStatus() map[string]interface{} {
	zp.mu.RLock()
	defer zp.mu.RUnlock()

	routes := make([]map[string]interface{}, 0, len(zp.config.Routes))
	for _, route := range zp.config.Routes {
		lb := zp.loadBalancers[route.Path]
		healthyCount := 0
		if lb != nil {
			healthyCount = lb.GetHealthyBackendCount()
		}

		routes = append(routes, map[string]interface{}{
			"path":            route.Path,
			"backends_total":  len(route.Backends),
			"backends_healthy": healthyCount,
			"load_balancing":  route.LoadBalancing,
		})
	}

	return map[string]interface{}{
		"status": "running",
		"routes": routes,
	}
}
