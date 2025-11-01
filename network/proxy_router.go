package network

import (
	"net/http"
	"regexp"
	"strings"
)

// RouteMatch represents a matched route with its configuration
type RouteMatch struct {
	Route  *RouteConfig
	Params map[string]string
}

// Router handles route matching for the proxy
type Router struct {
	staticRoutes  map[string]*RouteConfig // Exact path matches
	patternRoutes []patternRoute          // Pattern-based routes
}

type patternRoute struct {
	regex   *regexp.Regexp
	config  *RouteConfig
	pattern string
	params  []string
}

// NewRouter creates a new router from proxy configuration
func NewRouter(config *ProxyConfig) *Router {
	router := &Router{
		staticRoutes:  make(map[string]*RouteConfig),
		patternRoutes: make([]patternRoute, 0),
	}

	for i := range config.Routes {
		route := &config.Routes[i]
		router.AddRoute(route)
	}

	return router
}

// AddRoute adds a route to the router
func (r *Router) AddRoute(route *RouteConfig) {
	// Check if this is a static route (no wildcards or parameters)
	if !strings.Contains(route.Path, "*") && !strings.Contains(route.Path, ":") {
		r.staticRoutes[route.Path] = route
		return
	}

	// Convert route pattern to regex
	pattern := route.Path
	params := make([]string, 0)

	// Replace :param with named capture groups
	paramRegex := regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := paramRegex.FindAllStringSubmatch(pattern, -1)
	for _, match := range matches {
		paramName := match[1]
		params = append(params, paramName)
		pattern = strings.Replace(pattern, match[0], `(?P<`+paramName+`>[^/]+)`, 1)
	}

	// Replace * with wildcard capture
	pattern = strings.ReplaceAll(pattern, "/*", "/.*")
	pattern = strings.ReplaceAll(pattern, "*", ".*")

	// Ensure pattern matches from start to end exactly (unless it has wildcard)
	if !strings.HasPrefix(pattern, "^") {
		pattern = "^" + pattern
	}
	// Only add $ if pattern doesn't end with wildcard
	if !strings.HasSuffix(pattern, ".*") {
		pattern = pattern + "$"
	}

	regex := regexp.MustCompile(pattern)

	r.patternRoutes = append(r.patternRoutes, patternRoute{
		regex:   regex,
		config:  route,
		pattern: route.Path,
		params:  params,
	})
}

// Match finds a matching route for the given request
func (r *Router) Match(req *http.Request) *RouteMatch {
	path := req.URL.Path

	// Try exact match first (fastest)
	if route, exists := r.staticRoutes[path]; exists {
		if r.methodAllowed(route, req.Method) {
			return &RouteMatch{
				Route:  route,
				Params: make(map[string]string),
			}
		}
	}

	// Try pattern matches
	for _, pr := range r.patternRoutes {
		if matches := pr.regex.FindStringSubmatch(path); matches != nil {
			if r.methodAllowed(pr.config, req.Method) {
				params := make(map[string]string)

				// Extract named parameters
				for i, name := range pr.regex.SubexpNames() {
					if i > 0 && i <= len(matches) && name != "" {
						params[name] = matches[i]
					}
				}

				return &RouteMatch{
					Route:  pr.config,
					Params: params,
				}
			}
		}
	}

	return nil
}

// methodAllowed checks if the HTTP method is allowed for the route
func (r *Router) methodAllowed(route *RouteConfig, method string) bool {
	// If no methods specified, allow all
	if len(route.Methods) == 0 {
		return true
	}

	// Check if method is in allowed list
	for _, allowedMethod := range route.Methods {
		if strings.EqualFold(allowedMethod, method) {
			return true
		}
	}

	return false
}

// RewritePath rewrites the request path based on route configuration
func RewritePath(originalPath string, route *RouteConfig, params map[string]string) string {
	path := originalPath

	// Strip prefix if configured
	if route.StripPrefix {
		// Remove the matched prefix
		prefix := route.Path
		// Handle wildcard patterns
		if strings.Contains(prefix, "*") {
			prefix = strings.Split(prefix, "*")[0]
		}
		// Handle parameter patterns
		if strings.Contains(prefix, ":") {
			prefix = strings.Split(prefix, ":")[0]
		}
		path = strings.TrimPrefix(path, strings.TrimSuffix(prefix, "/"))
	}

	// Add prefix if configured
	if route.AddPrefix != "" {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		path = strings.TrimSuffix(route.AddPrefix, "/") + path
	}

	// Replace parameters in path
	for key, value := range params {
		path = strings.ReplaceAll(path, ":"+key, value)
	}

	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}

// GetAllowedMethods returns all allowed methods for a path
func (r *Router) GetAllowedMethods(path string) []string {
	methods := make(map[string]bool)

	// Check static routes
	if route, exists := r.staticRoutes[path]; exists {
		if len(route.Methods) > 0 {
			for _, method := range route.Methods {
				methods[method] = true
			}
		} else {
			// If no methods specified, all are allowed
			return []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
		}
	}

	// Check pattern routes
	for _, pr := range r.patternRoutes {
		if pr.regex.MatchString(path) {
			if len(pr.config.Methods) > 0 {
				for _, method := range pr.config.Methods {
					methods[method] = true
				}
			} else {
				return []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
			}
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(methods))
	for method := range methods {
		result = append(result, method)
	}

	// Always add OPTIONS for CORS
	if len(result) > 0 {
		result = append(result, "OPTIONS")
	}

	return result
}
