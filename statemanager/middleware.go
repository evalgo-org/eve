package statemanager

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// ContextKey for storing operation ID in echo context
const OperationIDKey = "operation_id"

// Middleware creates Echo middleware for automatic operation tracking
// Usage: e.Use(stateManager.Middleware("operation-type"))
func (m *Manager) Middleware(operationType string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Generate operation ID
			opID := uuid.New().String()

			// Start tracking with basic metadata
			m.StartOperation(opID, operationType, map[string]interface{}{
				"path":   c.Path(),
				"method": c.Request().Method,
			})

			// Store in context for handlers to use
			c.Set(OperationIDKey, opID)

			// Execute handler
			err := next(c)

			// Complete tracking
			m.CompleteOperation(opID, err)

			return err
		}
	}
}

// GetOperationID retrieves the operation ID from the echo context
// Returns empty string if not found
func GetOperationID(c echo.Context) string {
	if opID, ok := c.Get(OperationIDKey).(string); ok {
		return opID
	}
	return ""
}
