// Package semantic provides Schema.org-based action types and error handling utilities
package semantic

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// SetErrorOnAction sets error information on a semantic action
// This is a generic helper that works with actions that have string EndTime fields
func SetErrorOnAction(action interface{}, message string) {
	now := time.Now().Format(time.RFC3339)

	switch v := action.(type) {
	// BaseX actions (EndTime: string)
	case *TransformAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = now
	case *QueryAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = now
	case *BaseXUploadAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = now
	case *CreateDatabaseAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = now
	case *DeleteDatabaseAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = now

	// S3 actions (EndTime: string)
	case *S3UploadAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = now
	case *S3DownloadAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = now
	case *S3DeleteAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = now
	case *S3ListAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = now

	// SPARQL actions (no EndTime field)
	case *SearchAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}

	// Infisical actions (EndTime: string)
	case *InfisicalRetrieveAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = now

	// Container actions (EndTime: string)
	case *ActivateAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = now
	case *DeactivateAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = now
	case *DownloadAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = now
	case *BuildAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = now

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
// GraphDB actions use *time.Time instead of string
func SetErrorOnTimeAction(action interface{}, message string) {
	nowTime := time.Now()

	switch v := action.(type) {
	// GraphDB actions (EndTime: *time.Time)
	case *TransferAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = &nowTime
	case *CreateAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = &nowTime
	case *DeleteAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = &nowTime
	case *UpdateAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = &nowTime
	case *UploadAction:
		v.ActionStatus = "FailedActionStatus"
		v.Error = &PropertyValue{Type: "PropertyValue", Name: "error", Value: message}
		v.EndTime = &nowTime
	}
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

	SetErrorOnAction(action, fullMessage)
	return c.JSON(http.StatusInternalServerError, action)
}

// ReturnActionErrorWithStatus is like ReturnActionError but allows custom HTTP status code
func ReturnActionErrorWithStatus(c echo.Context, action interface{}, statusCode int, message string, err error) error {
	fullMessage := message
	if err != nil {
		fullMessage = fmt.Sprintf("%s: %v", message, err)
	}

	SetErrorOnAction(action, fullMessage)
	return c.JSON(statusCode, action)
}

// SetSuccessOnAction sets success status on a semantic action with string EndTime
func SetSuccessOnAction(action interface{}) {
	now := time.Now().Format(time.RFC3339)

	switch v := action.(type) {
	// BaseX actions
	case *TransformAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = now
	case *QueryAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = now
	case *BaseXUploadAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = now
	case *CreateDatabaseAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = now
	case *DeleteDatabaseAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = now

	// S3 actions
	case *S3UploadAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = now
	case *S3DownloadAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = now
	case *S3DeleteAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = now
	case *S3ListAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = now

	// SPARQL actions (no EndTime)
	case *SearchAction:
		v.ActionStatus = "CompletedActionStatus"

	// Infisical actions
	case *InfisicalRetrieveAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = now

	// Container actions
	case *ActivateAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = now
	case *DeactivateAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = now
	case *DownloadAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = now
	case *BuildAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = now

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
func SetSuccessOnTimeAction(action interface{}) {
	nowTime := time.Now()

	switch v := action.(type) {
	// GraphDB actions
	case *TransferAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = &nowTime
	case *CreateAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = &nowTime
	case *DeleteAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = &nowTime
	case *UpdateAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = &nowTime
	case *UploadAction:
		v.ActionStatus = "CompletedActionStatus"
		v.EndTime = &nowTime
	}
}
