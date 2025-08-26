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

type Handlers struct {
	RabbitMQ *queue.RabbitMQService
	CouchDB  *db.CouchDBService
	JWT      *security.JWTService
}

func SetupRoutes(e *echo.Echo, h *Handlers, c *eve.FlowConfig) {
	// Public routes
	e.POST("/auth/token", h.GenerateToken)

	// Protected routes
	protected := e.Group("/api")
	protected.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey:  []byte(c.ApiKey),
		TokenLookup: "header:Authorization:Bearer ",
	}))

	protected.POST("/publish", h.PublishMessage)
	protected.GET("/processes/:id", h.GetProcess)
	protected.GET("/processes", h.GetProcessesByState)
}

type TokenRequest struct {
	UserID string `json:"user_id"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

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

	// Validate state
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
