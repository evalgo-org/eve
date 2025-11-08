package tracing

import (
	"net/http"
)

// PropagateHeaders adds correlation headers to an HTTP request
// Use this when making service-to-service calls to maintain trace context
func PropagateHeaders(req *http.Request, correlationID, operationID string) {
	if correlationID != "" {
		req.Header.Set("X-Correlation-ID", correlationID)
	}

	if operationID != "" {
		// Current operation becomes parent for downstream calls
		req.Header.Set("X-Parent-Operation-ID", operationID)
	}
}

// PropagateFromContext extracts correlation IDs from Echo context and adds to HTTP request
func PropagateFromContext(c interface{}, req *http.Request) {
	correlationID := GetCorrelationID(c)
	operationID := GetOperationID(c)

	PropagateHeaders(req, correlationID, operationID)
}
