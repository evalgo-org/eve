// Package api provides HTTP handlers and routing for the EVE evaluation service.
// It includes authentication, message publishing, and process management endpoints.
package api

import (
	"net/http"
	"time"

	eve "eve.evalgo.org/common"
	"eve.evalgo.org/db"
	"eve.evalgo.org/queue"
	"eve.evalgo.org/security"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

// Handlers contains the service dependencies required for API operations.
// It provides access to RabbitMQ for message queuing, CouchDB for data persistence,
// and JWT service for authentication.
type Handlers struct {
	RabbitMQ *queue.RabbitMQService // RabbitMQ service for message publishing
	CouchDB  *db.CouchDBService     // CouchDB service for document storage
	JWT      *security.JWTService   // JWT service for token generation and validation
}

// SetupRoutes configures all API routes for the EVE service.
// It sets up both public and protected endpoints under the /v1/api base path.
//
// Public routes:
//   - POST /auth/token - Generate authentication token
//
// Protected routes (require JWT authentication):
//   - POST /v1/api/publish - Publish flow process messages
//   - GET /v1/api/processes/:id - Get specific process by ID
//   - GET /v1/api/processes - Get processes, optionally filtered by state
//
// Parameters:
//   - e: Echo instance to register routes with
//   - h: Handlers struct containing service dependencies
//   - c: FlowConfig containing API configuration including signing key
func SetupRoutes(e *echo.Echo, h *Handlers, c *eve.FlowConfig) {
	// Public routes - no authentication required
	auth := e.Group("/auth")
	auth.POST("/token", h.GenerateToken)

	// Protected routes - require JWT authentication
	protected := e.Group("/v1/api")
	protected.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey:  []byte(c.ApiKey),
		TokenLookup: "header:Authorization:Bearer ",
	}))

	// Message publishing endpoint
	protected.POST("/publish", h.PublishMessage)

	// Process management endpoints
	protected.GET("/processes/:id", h.GetProcess)
	protected.GET("/processes", h.GetProcessesByState)
}

// TokenRequest represents the request payload for token generation.
// It requires a user ID to associate with the generated JWT token.
type TokenRequest struct {
	UserID string `json:"user_id" validate:"required"` // User identifier for token association
}

// TokenResponse represents the response payload containing the generated JWT token.
type TokenResponse struct {
	Token string `json:"token"` // JWT token for API authentication
}

// GenerateToken handles JWT token generation for user authentication.
// It validates the user ID and generates a token with 24-hour expiration.
//
// Endpoint: POST /auth/token
//
// Request body:
//
//	{
//	  "user_id": "string" // Required: User identifier
//	}
//
// Response:
//
//	Success (200): {"token": "jwt_token_string"}
//	Bad Request (400): {"error": "error_message"}
//	Internal Error (500): {"error": "error_message"}
func (h *Handlers) GenerateToken(c echo.Context) error {
	var req TokenRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_id is required"})
	}

	token, err := h.JWT.GenerateToken(req.UserID, 24*time.Hour)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"})
	}

	return c.JSON(http.StatusOK, TokenResponse{Token: token})
}

// PublishMessage handles publishing of flow process messages to RabbitMQ.
// It validates the message format and required fields before publishing.
// If no timestamp is provided, it sets the current time.
//
// Endpoint: POST /v1/api/publish
// Authentication: Required (JWT Bearer token)
//
// Request body:
//
//	{
//	  "process_id": "string",    // Required: Process identifier
//	  "state": "string",         // Required: Process state
//	  "timestamp": "datetime",   // Optional: Message timestamp (auto-set if empty)
//	  // ... other FlowProcessMessage fields
//	}
//
// Response:
//
//	Success (200): {"status": "Message published successfully"}
//	Bad Request (400): {"error": "error_message"}
//	Internal Error (500): {"error": "error_message"}
func (h *Handlers) PublishMessage(c echo.Context) error {
	var message eve.FlowProcessMessage
	if err := c.Bind(&message); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid message format"})
	}

	// Set timestamp if not provided
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	// Validate required fields
	if message.ProcessID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "process_id is required"})
	}
	if message.State == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "state is required"})
	}

	err := h.RabbitMQ.PublishMessage(message)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to publish message"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "Message published successfully"})
}

// GetProcess retrieves a specific process document by its ID from CouchDB.
// Returns the complete process document if found.
//
// Endpoint: GET /v1/api/processes/:id
// Authentication: Required (JWT Bearer token)
//
// Path Parameters:
//   - id: Process identifier
//
// Response:
//
//	Success (200): Process document JSON
//	Bad Request (400): {"error": "Process ID is required"}
//	Not Found (404): {"error": "Process not found"}
//	Internal Error (500): {"error": "Failed to retrieve process"}
func (h *Handlers) GetProcess(c echo.Context) error {
	processID := c.Param("id")
	if processID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Process ID is required"})
	}

	doc, err := h.CouchDB.GetDocument(processID)
	if err != nil {
		if err.Error() == "document not found" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Process not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve process"})
	}

	return c.JSON(http.StatusOK, doc)
}

// GetProcessesByState retrieves processes from CouchDB, optionally filtered by state.
// If no state parameter is provided, returns all processes.
// Validates state parameter against allowed values if provided.
//
// Endpoint: GET /v1/api/processes
// Authentication: Required (JWT Bearer token)
//
// Query Parameters:
//   - state (optional): Filter processes by state
//     Valid values: "started", "running", "successful", "failed"
//
// Response:
//
//	Success (200): {
//	  "processes": [...],    // Array of process documents
//	  "count": number        // Total count of returned processes
//	}
//	Bad Request (400): {"error": "Invalid state value"}
//	Internal Error (500): {"error": "Failed to retrieve processes"}
func (h *Handlers) GetProcessesByState(c echo.Context) error {
	state := c.QueryParam("state")
	if state == "" {
		// If no state specified, return all processes
		docs, err := h.CouchDB.GetAllDocuments()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve processes"})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"processes": docs,
			"count":     len(docs),
		})
	}

	// Validate state parameter against allowed values
	validStates := []string{string(eve.StateStarted), string(eve.StateRunning), string(eve.StateSuccessful), string(eve.StateFailed)}
	isValid := false
	for _, validState := range validStates {
		if state == validState {
			isValid = true
			break
		}
	}

	if !isValid {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid state value"})
	}

	docs, err := h.CouchDB.GetDocumentsByState(eve.FlowProcessState(state))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve processes"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"processes": docs,
		"count":     len(docs),
	})
}
