// Package tracing - Service dependency mapping from traces
package tracing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// DependencyGraph represents service-to-service dependencies
type DependencyGraph struct {
	dependencies map[string]map[string]*DependencyEdge // from -> to -> edge
	services     map[string]*ServiceNode
	lastUpdated  time.Time
}

// ServiceNode represents a service in the dependency graph
type ServiceNode struct {
	ServiceID    string    `json:"service_id"`
	ActionTypes  []string  `json:"action_types"`   // Actions this service performs
	ObjectTypes  []string  `json:"object_types"`   // Objects this service handles
	RequestCount int64     `json:"request_count"`  // Total requests
	ErrorCount   int64     `json:"error_count"`    // Total errors
	AvgLatencyMs float64   `json:"avg_latency_ms"` // Average latency
	LastSeen     time.Time `json:"last_seen"`
}

// DependencyEdge represents a dependency between two services
type DependencyEdge struct {
	FromService  string    `json:"from_service"`
	ToService    string    `json:"to_service"`
	CallCount    int64     `json:"call_count"`     // Number of calls
	ErrorCount   int64     `json:"error_count"`    // Errors in calls
	AvgLatencyMs float64   `json:"avg_latency_ms"` // Average call latency
	ActionTypes  []string  `json:"action_types"`   // Actions used in calls
	LastSeen     time.Time `json:"last_seen"`
}

// CircularDependency represents a detected circular dependency
type CircularDependency struct {
	Cycle    []string  `json:"cycle"` // Service IDs in cycle
	Detected time.Time `json:"detected"`
}

// DependencyMapper builds service dependency graph from traces
type DependencyMapper struct {
	db    *sql.DB
	graph *DependencyGraph
	mu    sync.RWMutex
}

// NewDependencyMapper creates a new dependency mapper
func NewDependencyMapper(db *sql.DB) *DependencyMapper {
	return &DependencyMapper{
		db: db,
		graph: &DependencyGraph{
			dependencies: make(map[string]map[string]*DependencyEdge),
			services:     make(map[string]*ServiceNode),
			lastUpdated:  time.Now(),
		},
	}
}

// BuildGraphFromTraces builds dependency graph from recent traces
func (dm *DependencyMapper) BuildGraphFromTraces(ctx context.Context, hours int) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Query traces grouped by correlation_id
	query := `
		WITH workflow_traces AS (
			SELECT
				correlation_id,
				service_id,
				action_type,
				object_type,
				action_status,
				duration_ms,
				started_at,
				ROW_NUMBER() OVER (PARTITION BY correlation_id ORDER BY started_at) as step_num
			FROM action_executions
			WHERE started_at > NOW() - INTERVAL '1 hour' * $1
			ORDER BY correlation_id, started_at
		)
		SELECT
			w1.correlation_id,
			w1.service_id as from_service,
			w2.service_id as to_service,
			w1.action_type,
			w1.object_type,
			CASE WHEN w1.action_status = 'failed' THEN 1 ELSE 0 END as is_error,
			w1.duration_ms,
			COUNT(*) OVER (PARTITION BY w1.service_id, w2.service_id) as call_count
		FROM workflow_traces w1
		LEFT JOIN workflow_traces w2
			ON w1.correlation_id = w2.correlation_id
			AND w2.step_num = w1.step_num + 1
		WHERE w2.service_id IS NOT NULL
			AND w1.service_id != w2.service_id
	`

	rows, err := dm.db.QueryContext(ctx, query, hours)
	if err != nil {
		return fmt.Errorf("query traces: %w", err)
	}
	defer rows.Close()

	// Reset graph
	dm.graph = &DependencyGraph{
		dependencies: make(map[string]map[string]*DependencyEdge),
		services:     make(map[string]*ServiceNode),
		lastUpdated:  time.Now(),
	}

	// Process rows
	for rows.Next() {
		var (
			correlationID string
			fromService   string
			toService     string
			actionType    string
			objectType    string
			isError       int
			durationMs    float64
			callCount     int64
		)

		if err := rows.Scan(&correlationID, &fromService, &toService, &actionType, &objectType, &isError, &durationMs, &callCount); err != nil {
			return fmt.Errorf("scan row: %w", err)
		}

		// Add/update services
		dm.addOrUpdateService(fromService, actionType, objectType, isError, durationMs)
		dm.addOrUpdateService(toService, actionType, objectType, isError, durationMs)

		// Add/update dependency edge
		dm.addOrUpdateDependency(fromService, toService, actionType, isError, durationMs)
	}

	return rows.Err()
}

// addOrUpdateService adds or updates a service node
func (dm *DependencyMapper) addOrUpdateService(serviceID, actionType, objectType string, isError int, durationMs float64) {
	service, exists := dm.graph.services[serviceID]
	if !exists {
		service = &ServiceNode{
			ServiceID:    serviceID,
			ActionTypes:  []string{},
			ObjectTypes:  []string{},
			RequestCount: 0,
			ErrorCount:   0,
			AvgLatencyMs: 0,
			LastSeen:     time.Now(),
		}
		dm.graph.services[serviceID] = service
	}

	// Update service stats
	service.RequestCount++
	if isError == 1 {
		service.ErrorCount++
	}

	// Update average latency (incremental average)
	service.AvgLatencyMs = ((service.AvgLatencyMs * float64(service.RequestCount-1)) + durationMs) / float64(service.RequestCount)
	service.LastSeen = time.Now()

	// Add action type if new
	if !contains(service.ActionTypes, actionType) {
		service.ActionTypes = append(service.ActionTypes, actionType)
	}

	// Add object type if new
	if !contains(service.ObjectTypes, objectType) {
		service.ObjectTypes = append(service.ObjectTypes, objectType)
	}
}

// addOrUpdateDependency adds or updates a dependency edge
func (dm *DependencyMapper) addOrUpdateDependency(fromService, toService, actionType string, isError int, durationMs float64) {
	// Get or create from service dependencies
	if dm.graph.dependencies[fromService] == nil {
		dm.graph.dependencies[fromService] = make(map[string]*DependencyEdge)
	}

	edge, exists := dm.graph.dependencies[fromService][toService]
	if !exists {
		edge = &DependencyEdge{
			FromService:  fromService,
			ToService:    toService,
			CallCount:    0,
			ErrorCount:   0,
			AvgLatencyMs: 0,
			ActionTypes:  []string{},
			LastSeen:     time.Now(),
		}
		dm.graph.dependencies[fromService][toService] = edge
	}

	// Update edge stats
	edge.CallCount++
	if isError == 1 {
		edge.ErrorCount++
	}

	// Update average latency
	edge.AvgLatencyMs = ((edge.AvgLatencyMs * float64(edge.CallCount-1)) + durationMs) / float64(edge.CallCount)
	edge.LastSeen = time.Now()

	// Add action type if new
	if !contains(edge.ActionTypes, actionType) {
		edge.ActionTypes = append(edge.ActionTypes, actionType)
	}
}

// GetGraph returns a copy of the dependency graph
func (dm *DependencyMapper) GetGraph() *DependencyGraph {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	// Deep copy
	graphCopy := &DependencyGraph{
		dependencies: make(map[string]map[string]*DependencyEdge),
		services:     make(map[string]*ServiceNode),
		lastUpdated:  dm.graph.lastUpdated,
	}

	for serviceID, service := range dm.graph.services {
		serviceCopy := *service
		graphCopy.services[serviceID] = &serviceCopy
	}

	for from, deps := range dm.graph.dependencies {
		graphCopy.dependencies[from] = make(map[string]*DependencyEdge)
		for to, edge := range deps {
			edgeCopy := *edge
			graphCopy.dependencies[from][to] = &edgeCopy
		}
	}

	return graphCopy
}

// GetService returns service information
func (dm *DependencyMapper) GetService(serviceID string) (*ServiceNode, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	service, exists := dm.graph.services[serviceID]
	if !exists {
		return nil, false
	}

	// Return copy
	serviceCopy := *service
	return &serviceCopy, true
}

// GetDependencies returns all dependencies for a service
func (dm *DependencyMapper) GetDependencies(serviceID string) []*DependencyEdge {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	deps, exists := dm.graph.dependencies[serviceID]
	if !exists {
		return []*DependencyEdge{}
	}

	edges := make([]*DependencyEdge, 0, len(deps))
	for _, edge := range deps {
		edgeCopy := *edge
		edges = append(edges, &edgeCopy)
	}

	return edges
}

// GetDependents returns all services that depend on this service
func (dm *DependencyMapper) GetDependents(serviceID string) []*DependencyEdge {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	dependents := []*DependencyEdge{}

	for _, deps := range dm.graph.dependencies {
		if edge, exists := deps[serviceID]; exists {
			edgeCopy := *edge
			dependents = append(dependents, &edgeCopy)
		}
	}

	return dependents
}

// DetectCircularDependencies finds circular dependencies in the graph
func (dm *DependencyMapper) DetectCircularDependencies() []CircularDependency {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	cycles := []CircularDependency{}
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	// DFS to detect cycles
	var dfs func(service string, path []string) bool
	dfs = func(service string, path []string) bool {
		visited[service] = true
		recStack[service] = true
		path = append(path, service)

		// Check all dependencies
		if deps, exists := dm.graph.dependencies[service]; exists {
			for toService := range deps {
				if !visited[toService] {
					if dfs(toService, path) {
						return true
					}
				} else if recStack[toService] {
					// Found cycle - extract it from path
					cycleStart := -1
					for i, s := range path {
						if s == toService {
							cycleStart = i
							break
						}
					}
					if cycleStart >= 0 {
						cycle := make([]string, len(path[cycleStart:]))
						copy(cycle, path[cycleStart:])
						cycles = append(cycles, CircularDependency{
							Cycle:    cycle,
							Detected: time.Now(),
						})
					}
					return true
				}
			}
		}

		recStack[service] = false
		return false
	}

	// Run DFS from each unvisited service
	for serviceID := range dm.graph.services {
		if !visited[serviceID] {
			dfs(serviceID, []string{})
		}
	}

	return cycles
}

// ExportToDOT exports dependency graph to Graphviz DOT format
func (dm *DependencyMapper) ExportToDOT() string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	dot := "digraph ServiceDependencies {\n"
	dot += "  rankdir=LR;\n"
	dot += "  node [shape=box, style=rounded];\n\n"

	// Add nodes
	for serviceID, service := range dm.graph.services {
		errorRate := 0.0
		if service.RequestCount > 0 {
			errorRate = float64(service.ErrorCount) / float64(service.RequestCount) * 100
		}

		// Color based on error rate
		color := "lightblue"
		if errorRate > 10 {
			color = "red"
		} else if errorRate > 5 {
			color = "orange"
		}

		dot += fmt.Sprintf("  \"%s\" [label=\"%s\\n%.0f req/s\\n%.2f%% errors\\n%.0fms avg\", fillcolor=%s, style=filled];\n",
			serviceID, serviceID, float64(service.RequestCount)/3600, errorRate, service.AvgLatencyMs, color)
	}

	dot += "\n"

	// Add edges
	for from, deps := range dm.graph.dependencies {
		for to, edge := range deps {
			label := fmt.Sprintf("%d calls\\n%.0fms", edge.CallCount, edge.AvgLatencyMs)

			// Edge color based on error rate
			edgeColor := "black"
			if edge.CallCount > 0 {
				errorRate := float64(edge.ErrorCount) / float64(edge.CallCount) * 100
				if errorRate > 10 {
					edgeColor = "red"
				} else if errorRate > 5 {
					edgeColor = "orange"
				}
			}

			dot += fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"%s\", color=%s];\n",
				from, to, label, edgeColor)
		}
	}

	dot += "}\n"
	return dot
}

// ExportToJSON exports dependency graph to JSON
func (dm *DependencyMapper) ExportToJSON() (string, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	data := map[string]interface{}{
		"services":     dm.graph.services,
		"dependencies": dm.graph.dependencies,
		"last_updated": dm.graph.lastUpdated,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

// GetMetrics returns dependency graph metrics
func (dm *DependencyMapper) GetMetrics() map[string]interface{} {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	totalEdges := 0
	for _, deps := range dm.graph.dependencies {
		totalEdges += len(deps)
	}

	return map[string]interface{}{
		"total_services":     len(dm.graph.services),
		"total_dependencies": totalEdges,
		"last_updated":       dm.graph.lastUpdated,
		"has_cycles":         len(dm.DetectCircularDependencies()) > 0,
	}
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
