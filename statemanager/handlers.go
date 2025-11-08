package statemanager

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// RegisterRoutes adds state endpoints to an Echo group
func (m *Manager) RegisterRoutes(g *echo.Group) {
	g.GET("/state", m.handleListOperations)
	g.GET("/state/:id", m.handleGetOperation)
	g.GET("/state/stats", m.handleGetStats)
}

// handleListOperations returns all tracked operations
func (m *Manager) handleListOperations(c echo.Context) error {
	return c.JSON(http.StatusOK, m.ListOperations())
}

// handleGetOperation returns a specific operation by ID
func (m *Manager) handleGetOperation(c echo.Context) error {
	id := c.Param("id")
	op := m.GetOperation(id)
	if op == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "operation not found",
		})
	}
	return c.JSON(http.StatusOK, op)
}

// handleGetStats returns aggregated statistics
func (m *Manager) handleGetStats(c echo.Context) error {
	return c.JSON(http.StatusOK, m.GetStats())
}
