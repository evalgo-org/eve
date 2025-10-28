package network

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Backend represents a backend service with its state
type Backend struct {
	Config      *BackendConfig
	Client      *http.Client
	Transport   *http.Transport
	Healthy     atomic.Bool
	Connections atomic.Int64
	LastCheck   time.Time
	FailCount   int
	SuccessCount int
	mu          sync.RWMutex
}

// LoadBalancer manages backend selection and health
type LoadBalancer struct {
	backends       []*Backend
	strategy       LoadBalancingStrategy
	roundRobinIdx  atomic.Uint64
	healthChecker  *HealthChecker
	mu             sync.RWMutex
}

// NewLoadBalancer creates a new load balancer for a route
func NewLoadBalancer(route *RouteConfig) (*LoadBalancer, error) {
	lb := &LoadBalancer{
		backends: make([]*Backend, 0, len(route.Backends)),
		strategy: route.LoadBalancing,
	}

	// Initialize backends
	for i := range route.Backends {
		backendConfig := &route.Backends[i]

		// Create Ziti transport for this backend
		transport, err := ZitiSetup(backendConfig.IdentityFile, backendConfig.ZitiService)
		if err != nil {
			return nil, err
		}

		// Create HTTP client with Ziti transport
		client := &http.Client{
			Transport: transport,
			Timeout:   backendConfig.Timeout.Duration,
		}

		backend := &Backend{
			Config:    backendConfig,
			Client:    client,
			Transport: transport,
		}
		backend.Healthy.Store(true) // Assume healthy initially
		backend.Connections.Store(0)

		lb.backends = append(lb.backends, backend)
	}

	// Start health checker if enabled
	if route.HealthCheck != nil && route.HealthCheck.Enabled {
		lb.healthChecker = NewHealthChecker(lb.backends, route.HealthCheck)
		lb.healthChecker.Start()
	}

	return lb, nil
}

// SelectBackend selects a backend based on the load balancing strategy
func (lb *LoadBalancer) SelectBackend() *Backend {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Filter healthy backends
	healthy := make([]*Backend, 0, len(lb.backends))
	for _, backend := range lb.backends {
		if backend.Healthy.Load() {
			healthy = append(healthy, backend)
		}
	}

	// If no healthy backends, try all backends (circuit breaker pattern)
	if len(healthy) == 0 {
		healthy = lb.backends
	}

	if len(healthy) == 0 {
		return nil
	}

	switch lb.strategy {
	case WeightedRoundRobin:
		return lb.selectWeightedRoundRobin(healthy)
	case LeastConnections:
		return lb.selectLeastConnections(healthy)
	default: // RoundRobin
		return lb.selectRoundRobin(healthy)
	}
}

// selectRoundRobin implements round-robin selection
func (lb *LoadBalancer) selectRoundRobin(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}
	idx := lb.roundRobinIdx.Add(1) - 1
	return backends[idx%uint64(len(backends))]
}

// selectWeightedRoundRobin implements weighted round-robin selection
func (lb *LoadBalancer) selectWeightedRoundRobin(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}

	// Calculate total weight
	totalWeight := 0
	for _, backend := range backends {
		totalWeight += backend.Config.Weight
	}

	if totalWeight == 0 {
		return lb.selectRoundRobin(backends)
	}

	// Use round-robin counter modulo total weight
	idx := lb.roundRobinIdx.Add(1) - 1
	position := int(idx % uint64(totalWeight))

	// Find backend at this weighted position
	currentWeight := 0
	for _, backend := range backends {
		currentWeight += backend.Config.Weight
		if position < currentWeight {
			return backend
		}
	}

	return backends[0]
}

// selectLeastConnections implements least connections selection
func (lb *LoadBalancer) selectLeastConnections(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}

	var selected *Backend
	minConnections := int64(-1)

	for _, backend := range backends {
		connections := backend.Connections.Load()
		if minConnections == -1 || connections < minConnections {
			minConnections = connections
			selected = backend
		}
	}

	return selected
}

// RecordSuccess records a successful request to a backend
func (lb *LoadBalancer) RecordSuccess(backend *Backend) {
	backend.mu.Lock()
	defer backend.mu.Unlock()

	backend.SuccessCount++
	backend.FailCount = 0 // Reset failure count on success
}

// RecordFailure records a failed request to a backend
func (lb *LoadBalancer) RecordFailure(backend *Backend) {
	backend.mu.Lock()
	defer backend.mu.Unlock()

	backend.FailCount++
	backend.SuccessCount = 0

	// Mark unhealthy after consecutive failures
	if backend.FailCount >= 3 {
		backend.Healthy.Store(false)
	}
}

// IncrementConnections increments active connection count
func (lb *LoadBalancer) IncrementConnections(backend *Backend) {
	backend.Connections.Add(1)
}

// DecrementConnections decrements active connection count
func (lb *LoadBalancer) DecrementConnections(backend *Backend) {
	backend.Connections.Add(-1)
}

// Stop stops the load balancer and health checker
func (lb *LoadBalancer) Stop() {
	if lb.healthChecker != nil {
		lb.healthChecker.Stop()
	}
}

// GetHealthyBackendCount returns the number of healthy backends
func (lb *LoadBalancer) GetHealthyBackendCount() int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	count := 0
	for _, backend := range lb.backends {
		if backend.Healthy.Load() {
			count++
		}
	}
	return count
}

// HealthChecker performs periodic health checks on backends
type HealthChecker struct {
	backends []*Backend
	config   *HealthCheckConfig
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(backends []*Backend, config *HealthCheckConfig) *HealthChecker {
	return &HealthChecker{
		backends: backends,
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Start starts the health checking routine
func (hc *HealthChecker) Start() {
	hc.wg.Add(1)
	go hc.run()
}

// Stop stops the health checking routine
func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
	hc.wg.Wait()
}

// run performs periodic health checks
func (hc *HealthChecker) run() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.config.Interval.Duration)
	defer ticker.Stop()

	// Perform initial health check
	hc.checkAllBackends()

	for {
		select {
		case <-ticker.C:
			hc.checkAllBackends()
		case <-hc.stopChan:
			return
		}
	}
}

// checkAllBackends checks health of all backends
func (hc *HealthChecker) checkAllBackends() {
	for _, backend := range hc.backends {
		go hc.checkBackend(backend)
	}
}

// checkBackend performs a health check on a single backend
func (hc *HealthChecker) checkBackend(backend *Backend) {
	backend.mu.Lock()
	defer backend.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), hc.config.Timeout.Duration)
	defer cancel()

	// Create health check request
	url := "http://" + backend.Config.ZitiService + hc.config.Path
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		hc.markUnhealthy(backend)
		return
	}

	// Perform health check
	resp, err := backend.Client.Do(req)
	if err != nil {
		hc.markUnhealthy(backend)
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode == hc.config.ExpectedStatus {
		hc.markHealthy(backend)
	} else {
		hc.markUnhealthy(backend)
	}

	backend.LastCheck = time.Now()
}

// markHealthy marks a backend as healthy
func (hc *HealthChecker) markHealthy(backend *Backend) {
	backend.SuccessCount++
	backend.FailCount = 0

	if backend.SuccessCount >= hc.config.SuccessCount {
		backend.Healthy.Store(true)
	}
}

// markUnhealthy marks a backend as unhealthy
func (hc *HealthChecker) markUnhealthy(backend *Backend) {
	backend.FailCount++
	backend.SuccessCount = 0

	if backend.FailCount >= hc.config.FailureCount {
		backend.Healthy.Store(false)
	}
}
