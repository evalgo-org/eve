// Package tracing - HTTP handler for Prometheus metrics
package tracing

import (
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler returns an Echo handler for /metrics endpoint
func MetricsHandler() echo.HandlerFunc {
	h := promhttp.Handler()

	return func(c echo.Context) error {
		h.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

// RegisterMetricsEndpoint registers the /metrics endpoint on an Echo server
func RegisterMetricsEndpoint(e *echo.Echo, path string) {
	if path == "" {
		path = "/metrics"
	}

	e.GET(path, MetricsHandler())
}
