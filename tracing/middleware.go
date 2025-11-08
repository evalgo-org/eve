// Package tracing provides middleware for action-based execution tracing
// across EVE services using hybrid S3 + PostgreSQL storage.
package tracing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Middleware returns an Echo middleware that captures action execution traces
func (t *Tracer) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Only trace semantic action endpoint
			if c.Path() != "/v1/api/semantic/action" {
				return next(c)
			}

			if !t.config.Enabled {
				return next(c)
			}

			// Extract or generate correlation IDs
			correlationID := c.Request().Header.Get("X-Correlation-ID")
			if correlationID == "" {
				correlationID = fmt.Sprintf("wf-%s", uuid.New().String()[:8])
			}

			operationID := c.Request().Header.Get("X-Operation-ID")
			if operationID == "" {
				operationID = fmt.Sprintf("op-%s", uuid.New().String()[:8])
			}

			parentOperationID := c.Request().Header.Get("X-Parent-Operation-ID")

			// Store in context for downstream use
			c.Set("correlation_id", correlationID)
			c.Set("operation_id", operationID)
			c.Set("parent_operation_id", parentOperationID)

			// Capture request body
			reqBody, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return err
			}
			c.Request().Body = io.NopCloser(bytes.NewBuffer(reqBody))

			// Parse action type for metadata extraction
			actionType, objectType := parseActionTypes(reqBody)

			// Parse action-level tracing metadata
			metaTrace, metaStorePayload := parseTracingMeta(reqBody)

			// Check if action explicitly disables tracing
			if !metaTrace {
				return next(c)
			}

			// Check if this action should be traced based on exclusion rules
			if !t.shouldTrace(actionType, objectType) {
				return next(c)
			}

			// Store action-level payload preference
			c.Set("meta_store_payload", metaStorePayload)

			// Create response recorder
			rec := &responseRecorder{
				ResponseWriter: c.Response().Writer,
				body:           &bytes.Buffer{},
			}
			c.Response().Writer = rec

			// Execute handler
			startTime := time.Now()
			handlerErr := next(c)
			duration := time.Since(startTime)

			// Parse response for action status
			actionStatus := parseActionStatus(rec.body.Bytes())
			statusCode := c.Response().Status
			if statusCode == 0 {
				statusCode = 200
			}

			// Extract error if present
			var errorMsg string
			var errorType string
			if handlerErr != nil {
				errorMsg = handlerErr.Error()
				errorType = fmt.Sprintf("%T", handlerErr)
			}

			// Get action-level payload preference
			metaStorePayloadFlag := true
			if val := c.Get("meta_store_payload"); val != nil {
				if flag, ok := val.(bool); ok {
					metaStorePayloadFlag = flag
				}
			}

			// Record trace asynchronously
			go t.recordTrace(traceRecord{
				correlationID:     correlationID,
				operationID:       operationID,
				parentOperationID: parentOperationID,
				actionType:        actionType,
				objectType:        objectType,
				startTime:         startTime,
				duration:          duration,
				actionStatus:      actionStatus,
				statusCode:        statusCode,
				errorMsg:          errorMsg,
				errorType:         errorType,
				requestBody:       reqBody,
				responseBody:      rec.body.Bytes(),
				endpoint:          c.Path(),
				httpMethod:        c.Request().Method,
				clientIP:          c.RealIP(),
				userAgent:         c.Request().UserAgent(),
				metaStorePayload:  metaStorePayloadFlag,
			})

			return handlerErr
		}
	}
}

// responseRecorder captures response data
type responseRecorder struct {
	http.ResponseWriter
	body *bytes.Buffer
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// parseActionTypes extracts @type and object.@type from JSON-LD
func parseActionTypes(body []byte) (actionType, objectType string) {
	var action struct {
		Type   string `json:"@type"`
		Object struct {
			Type string `json:"@type"`
		} `json:"object"`
	}

	if err := json.Unmarshal(body, &action); err != nil {
		return "Unknown", "Unknown"
	}

	if action.Type == "" {
		action.Type = "Unknown"
	}
	if action.Object.Type == "" {
		action.Object.Type = "Unknown"
	}

	return action.Type, action.Object.Type
}

// parseTracingMeta extracts tracing control flags from action metadata
func parseTracingMeta(body []byte) (trace bool, storePayload bool) {
	var action struct {
		Meta struct {
			Trace        *bool `json:"trace"`        // Explicit trace enable/disable
			TracePayload *bool `json:"tracePayload"` // Explicit payload storage control
		} `json:"meta"`
	}

	// Default: allow tracing and use config for payload
	trace = true
	storePayload = true // Will be overridden by config

	if err := json.Unmarshal(body, &action); err != nil {
		return
	}

	// Check explicit trace flag
	if action.Meta.Trace != nil {
		trace = *action.Meta.Trace
	}

	// Check explicit payload flag
	if action.Meta.TracePayload != nil {
		storePayload = *action.Meta.TracePayload
	}

	return
}

// parseActionStatus extracts actionStatus from response
func parseActionStatus(body []byte) string {
	var result struct {
		ActionStatus string `json:"actionStatus"`
	}

	if err := json.Unmarshal(body, &result); err != nil || result.ActionStatus == "" {
		return "CompletedActionStatus" // Default to success
	}

	return result.ActionStatus
}

// traceRecord holds all data for a single trace
type traceRecord struct {
	correlationID     string
	operationID       string
	parentOperationID string
	actionType        string
	objectType        string
	startTime         time.Time
	duration          time.Duration
	actionStatus      string
	statusCode        int
	errorMsg          string
	errorType         string
	requestBody       []byte
	responseBody      []byte
	endpoint          string
	httpMethod        string
	clientIP          string
	userAgent         string
	metaStorePayload  bool // Action-level payload storage preference
}

// recordTrace stores the trace in PostgreSQL + S3
func (t *Tracer) recordTrace(rec traceRecord) {
	ctx := context.Background()

	// Check if payloads should be stored
	// Priority: 1) Credentials (never), 2) Action meta flag, 3) Config
	storePayloads := t.shouldStorePayload(rec.actionType, rec.objectType) && rec.metaStorePayload

	// S3 URLs (only set if payloads are stored)
	var requestURL, responseURL string

	if storePayloads {
		// Upload payloads to S3
		requestURL = fmt.Sprintf("s3://%s/%s/%s/request.json", t.config.S3Bucket, rec.correlationID, rec.operationID)
		responseURL = fmt.Sprintf("s3://%s/%s/%s/response.json", t.config.S3Bucket, rec.correlationID, rec.operationID)

		// Upload request to S3
		if err := t.uploadToS3(ctx, rec.correlationID, rec.operationID, "request.json", rec.requestBody); err != nil {
			t.logError("Failed to upload request to S3", err)
		}

		// Upload response to S3
		if err := t.uploadToS3(ctx, rec.correlationID, rec.operationID, "response.json", rec.responseBody); err != nil {
			t.logError("Failed to upload response to S3", err)
		}
	} else {
		// For credential-related actions, mark URLs as redacted
		requestURL = "[REDACTED - Credential payload not stored]"
		responseURL = "[REDACTED - Credential payload not stored]"
	}

	// Extract metadata based on action type
	metadata := t.extractMetadata(rec.actionType, rec.objectType, rec.requestBody, rec.responseBody)

	// Store metadata in PostgreSQL
	query := `
		INSERT INTO action_executions (
			correlation_id, operation_id, parent_operation_id,
			action_type, object_type,
			service_id, endpoint, http_method,
			started_at, completed_at, duration_ms,
			action_status, error_message, error_type,
			request_url, response_url,
			request_size_bytes, response_size_bytes,
			metadata, client_ip, user_agent
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)
	`

	_, err := t.config.DB.ExecContext(ctx, query,
		rec.correlationID,
		rec.operationID,
		nullString(rec.parentOperationID),
		rec.actionType,
		nullString(rec.objectType),
		t.config.ServiceID,
		rec.endpoint,
		rec.httpMethod,
		rec.startTime,
		rec.startTime.Add(rec.duration),
		rec.duration.Milliseconds(),
		rec.actionStatus,
		nullString(rec.errorMsg),
		nullString(rec.errorType),
		requestURL,
		responseURL,
		int64(len(rec.requestBody)),
		int64(len(rec.responseBody)),
		metadata,
		rec.clientIP,
		rec.userAgent,
	)

	if err != nil {
		t.logError("Failed to insert trace into PostgreSQL", err)
	}
}

// nullString returns nil for empty strings
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// logError logs errors (simple implementation, replace with proper logger)
func (t *Tracer) logError(msg string, err error) {
	fmt.Printf("[tracing] %s: %v\n", msg, err)
}
