// Package semantic provides Schema.org-based action types and error handling utilities
package semantic

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// SetErrorOnAction sets error information on a semantic action
// All legacy action types have been removed - this now only supports SemanticAction and CanonicalSemanticAction
func SetErrorOnAction(action interface{}, message string) {
	switch v := action.(type) {
	// SemanticAction - universal type, uses SemanticError and *time.Time
	case *SemanticAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &SemanticError{
			Type:    "Error",
			Message: message,
		}
		nowTime := time.Now()
		v.EndTime = &nowTime

	// Canonical action - uses SemanticThing for Error, *time.Time for EndTime
	case *CanonicalSemanticAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &SemanticThing{
			Type:        "Thing",
			Name:        "error",
			Description: message,
		}
		nowTime := time.Now()
		v.EndTime = &nowTime
	}
}

// SetErrorOnTimeAction sets error on actions that use *time.Time for EndTime
// DEPRECATED: Legacy action types removed. Use SetErrorOnAction instead.
// This function now delegates to SetErrorOnAction for SemanticAction
func SetErrorOnTimeAction(action interface{}, message string) {
	SetErrorOnAction(action, message)
}

// ReturnActionError is a helper that sets error on action and returns HTTP 500 response
// This standardizes error handling across all semantic action handlers
//
// Example usage:
//
//	action, err := semantic.ParseBaseXAction(body)
//	if err != nil {
//	    return semantic.ReturnActionError(c, action, "Failed to parse action", err)
//	}
//
//	result, err := executeAction(action)
//	if err != nil {
//	    return semantic.ReturnActionError(c, action, "Execution failed", err)
//	}
func ReturnActionError(c echo.Context, action interface{}, message string, err error) error {
	fullMessage := message
	if err != nil {
		fullMessage = fmt.Sprintf("%s: %v", message, err)
	}

	// Log error via logrus for when-v3 log forwarding
	logFields := logrus.Fields{
		"status":      "FailedActionStatus",
		"status_code": http.StatusInternalServerError,
	}

	// Add request context
	if c != nil {
		logFields["request_path"] = c.Request().URL.Path
		logFields["request_method"] = c.Request().Method
		logFields["remote_addr"] = c.RealIP()
	}

	// Extract action details for logging
	switch v := action.(type) {
	case *SemanticAction:
		logFields["action_type"] = v.Type
		logFields["action_id"] = v.Identifier
		logFields["action_name"] = v.Name
		logFields["action_description"] = v.Description

		// Extract workflow context from Properties (additionalProperty) if available
		if v.Properties != nil {
			if wfID, ok := v.Properties["workflowId"].(string); ok {
				logFields["workflow_id"] = wfID
			}
			if actionID, ok := v.Properties["actionId"].(string); ok {
				logFields["workflow_action_id"] = actionID
			}
			// Log other relevant parameters
			for key, val := range v.Properties {
				if key != "workflowId" && key != "actionId" && key != "url" {
					logFields["param_"+key] = val
				}
			}
		}

		// Extract target parameters
		if v.Target != nil {
			if targetMap, ok := v.Target.(map[string]interface{}); ok {
				if addProp, ok := targetMap["additionalProperty"].(map[string]interface{}); ok {
					for key, val := range addProp {
						if key != "url" { // Skip URL as it's often redundant
							logFields["target_"+key] = val
						}
					}
				}
			}
		}

	case *CanonicalSemanticAction:
		logFields["action_type"] = v.Type
		logFields["action_id"] = v.ID
		logFields["action_name"] = v.Name
	}

	// Add error as string field (not using WithError which doesn't serialize well)
	if err != nil {
		logFields["error"] = err.Error()
	}

	logrus.WithFields(logFields).Error(fullMessage)

	SetErrorOnAction(action, fullMessage)
	return c.JSON(http.StatusInternalServerError, action)
}

// ReturnActionErrorWithStatus is like ReturnActionError but allows custom HTTP status code
func ReturnActionErrorWithStatus(c echo.Context, action interface{}, statusCode int, message string, err error) error {
	fullMessage := message
	if err != nil {
		fullMessage = fmt.Sprintf("%s: %v", message, err)
	}

	// Log error via logrus for when-v3 log forwarding
	logFields := logrus.Fields{
		"status":      "FailedActionStatus",
		"status_code": statusCode,
	}

	// Add request context
	if c != nil {
		logFields["request_path"] = c.Request().URL.Path
		logFields["request_method"] = c.Request().Method
		logFields["remote_addr"] = c.RealIP()
	}

	// Extract action details for logging
	switch v := action.(type) {
	case *SemanticAction:
		logFields["action_type"] = v.Type
		logFields["action_id"] = v.Identifier
		logFields["action_name"] = v.Name
		logFields["action_description"] = v.Description

		// Extract workflow context from Properties (additionalProperty) if available
		if v.Properties != nil {
			if wfID, ok := v.Properties["workflowId"].(string); ok {
				logFields["workflow_id"] = wfID
			}
			if actionID, ok := v.Properties["actionId"].(string); ok {
				logFields["workflow_action_id"] = actionID
			}
			// Log other relevant parameters
			for key, val := range v.Properties {
				if key != "workflowId" && key != "actionId" && key != "url" {
					logFields["param_"+key] = val
				}
			}
		}

		// Extract target parameters
		if v.Target != nil {
			if targetMap, ok := v.Target.(map[string]interface{}); ok {
				if addProp, ok := targetMap["additionalProperty"].(map[string]interface{}); ok {
					for key, val := range addProp {
						if key != "url" { // Skip URL as it's often redundant
							logFields["target_"+key] = val
						}
					}
				}
			}
		}

	case *CanonicalSemanticAction:
		logFields["action_type"] = v.Type
		logFields["action_id"] = v.ID
		logFields["action_name"] = v.Name
	}

	// Add error as string field (not using WithError which doesn't serialize well)
	if err != nil {
		logFields["error"] = err.Error()
	}

	logrus.WithFields(logFields).Error(fullMessage)

	SetErrorOnAction(action, fullMessage)
	return c.JSON(statusCode, action)
}

// SetSuccessOnAction sets success status on a semantic action
// All legacy action types have been removed - this now only supports SemanticAction and CanonicalSemanticAction
func SetSuccessOnAction(action interface{}) {
	switch v := action.(type) {
	// SemanticAction - universal type
	case *SemanticAction:
		v.ActionStatus = "CompletedActionStatus"
		nowTime := time.Now()
		v.EndTime = &nowTime

	// Canonical action
	case *CanonicalSemanticAction:
		v.ActionStatus = "CompletedActionStatus"
		nowTime := time.Now()
		v.EndTime = &nowTime
	}
}

// SetSuccessOnTimeAction sets success on actions that use *time.Time for EndTime
// DEPRECATED: Legacy action types removed. Use SetSuccessOnAction instead.
// This function now delegates to SetSuccessOnAction for SemanticAction
func SetSuccessOnTimeAction(action interface{}) {
	SetSuccessOnAction(action)
}
