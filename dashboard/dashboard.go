// Package dashboard provides monitoring and statistics for EVE deployments.
// This package offers real-time visibility into containers, stacks, and JSON-LD
// deployments across Docker environments.
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"

	"eve.evalgo.org/common"
	"eve.evalgo.org/db"
	"github.com/docker/docker/client"
)

// Stats represents dashboard statistics for containers, stacks, and deployments.
//
// This structure provides a comprehensive view of the deployment ecosystem:
//   - Container distribution shows running containers by status
//   - Stack counts track multi-container application deployments
//   - JSON-LD deployments track semantic data deployments in CouchDB
//
// Example:
//
//	stats := Stats{
//	    ContainerDistribution: map[string]int{
//	        "running": 15,
//	        "stopped": 3,
//	        "failed": 1,
//	    },
//	    StackCount: 5,
//	    JsonLdDeploymentCount: 23,
//	}
type Stats struct {
	// ContainerDistribution maps container status to count
	ContainerDistribution map[string]int `json:"containerDistribution"`
	// StackCount is the total number of deployed stacks
	StackCount int `json:"stackCount"`
	// JsonLdDeploymentCount is the number of JSON-LD documents in CouchDB
	JsonLdDeploymentCount int `json:"jsonLdDeploymentCount"`
	// TotalContainers is the total number of containers across all statuses
	TotalContainers int `json:"totalContainers"`
}

// GetContainerDistribution retrieves container distribution by status from Docker.
//
// This function connects to Docker and categorizes containers by their current
// status (running, exited, paused, etc.), providing visibility into the
// container ecosystem health.
//
// Parameters:
//   - ctx: Context for Docker API operations
//   - cli: Docker client for querying container information
//
// Returns:
//   - map[string]int: Map of status to container count
//   - error: Docker API errors or connection issues
//
// Container Statuses:
//   - running: Containers currently executing
//   - exited: Stopped containers
//   - paused: Containers in paused state
//   - restarting: Containers currently restarting
//   - removing: Containers being removed
//   - created: Containers created but not started
//   - dead: Containers that failed to stop
//
// Example:
//
//	ctx, cli, err := common.CtxCli("unix:///var/run/docker.sock")
//	if err != nil {
//	    return fmt.Errorf("failed to create Docker client: %w", err)
//	}
//	defer cli.Close()
//
//	distribution, err := GetContainerDistribution(ctx, cli)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for status, count := range distribution {
//	    fmt.Printf("%s: %d\n", status, count)
//	}
func GetContainerDistribution(ctx context.Context, cli *client.Client) (map[string]int, error) {
	distribution := make(map[string]int)

	// Get all containers (including stopped ones)
	containers, err := cli.ContainerList(ctx, common.ContainerListAllOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Count by status
	for _, container := range containers {
		status := container.State
		distribution[status]++
	}

	return distribution, nil
}

// GetStackCount retrieves the number of deployed stacks from CouchDB.
//
// This function queries CouchDB for documents of type "Stack", providing
// visibility into multi-container application deployments managed through
// the stacks package.
//
// Parameters:
//   - service: CouchDB service instance for querying stack documents
//
// Returns:
//   - int: Number of stack deployments
//   - error: CouchDB query errors or connection issues
//
// Stack Definition:
//
//	Stacks are multi-container applications defined in JSON-LD format with:
//	- Container orchestration configurations
//	- Dependency relationships
//	- Health check definitions
//	- Network and volume configurations
//
// Example:
//
//	service, err := db.NewCouchDBServiceFromConfig(db.CouchDBConfig{
//	    URL:      "http://localhost:5984",
//	    Database: "deployments",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer service.Close()
//
//	count, err := GetStackCount(service)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Total stacks: %d\n", count)
func GetStackCount(service *db.CouchDBService) (int, error) {
	// Query for documents with @type = "Stack"
	selector := map[string]interface{}{
		"@type": "Stack",
	}

	count, err := service.Count(selector)
	if err != nil {
		return 0, fmt.Errorf("failed to count stacks: %w", err)
	}

	return count, nil
}

// GetJsonLdDeploymentCount retrieves the total number of JSON-LD documents.
//
// This function counts all documents in CouchDB that follow the JSON-LD
// specification (documents with @type, @context, or @id fields), providing
// visibility into semantic data deployments.
//
// Parameters:
//   - service: CouchDB service instance for querying JSON-LD documents
//
// Returns:
//   - int: Number of JSON-LD documents
//   - error: CouchDB query errors or connection issues
//
// JSON-LD Documents:
//
//	Documents following JSON-LD specification include:
//	- @type: Semantic type identifier
//	- @context: Vocabulary and namespace definitions
//	- @id: Unique identifier for the resource
//	- Additional semantic properties
//
// Use Cases:
//   - Tracking semantic data deployments
//   - Monitoring knowledge graph growth
//   - Analyzing linked data infrastructure
//   - Auditing JSON-LD compliance
//
// Example:
//
//	service, err := db.NewCouchDBServiceFromConfig(db.CouchDBConfig{
//	    URL:      "http://localhost:5984",
//	    Database: "semantic-data",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer service.Close()
//
//	count, err := GetJsonLdDeploymentCount(service)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Total JSON-LD documents: %d\n", count)
func GetJsonLdDeploymentCount(service *db.CouchDBService) (int, error) {
	// Query for documents with @type field (indicates JSON-LD document)
	selector := map[string]interface{}{
		"@type": map[string]interface{}{
			"$exists": true,
		},
	}

	count, err := service.Count(selector)
	if err != nil {
		return 0, fmt.Errorf("failed to count JSON-LD documents: %w", err)
	}

	return count, nil
}

// GetDashboardStats retrieves comprehensive dashboard statistics.
//
// This function aggregates data from multiple sources (Docker, CouchDB) to
// provide a complete view of the deployment ecosystem. It's the primary
// function for dashboard implementations.
//
// Parameters:
//   - ctx: Context for Docker API operations
//   - dockerClient: Docker client for container information
//   - couchdbService: CouchDB service for stack and JSON-LD data (optional, can be nil)
//
// Returns:
//   - *Stats: Comprehensive dashboard statistics
//   - error: Aggregation errors from any source
//
// Data Sources:
//   - Docker: Container distribution and status
//   - CouchDB: Stack deployments and JSON-LD documents
//
// Error Handling:
//
//	If CouchDB service is nil, stack and JSON-LD counts will be zero.
//	Partial failures are returned as errors to ensure data accuracy.
//
// Example:
//
//	ctx, cli, err := common.CtxCli("unix:///var/run/docker.sock")
//	if err != nil {
//	    return fmt.Errorf("failed to create Docker client: %w", err)
//	}
//	defer cli.Close()
//
//	couchdb, err := db.NewCouchDBServiceFromConfig(db.CouchDBConfig{
//	    URL:      "http://localhost:5984",
//	    Database: "deployments",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer couchdb.Close()
//
//	stats, err := GetDashboardStats(ctx, cli, couchdb)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Dashboard Stats:\n")
//	fmt.Printf("  Total Containers: %d\n", stats.TotalContainers)
//	fmt.Printf("  Stacks: %d\n", stats.StackCount)
//	fmt.Printf("  JSON-LD Documents: %d\n", stats.JsonLdDeploymentCount)
//	fmt.Printf("  Container Distribution:\n")
//	for status, count := range stats.ContainerDistribution {
//	    fmt.Printf("    %s: %d\n", status, count)
//	}
func GetDashboardStats(ctx context.Context, dockerClient *client.Client, couchdbService *db.CouchDBService) (*Stats, error) {
	stats := &Stats{
		ContainerDistribution: make(map[string]int),
	}

	// Get container distribution
	distribution, err := GetContainerDistribution(ctx, dockerClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get container distribution: %w", err)
	}
	stats.ContainerDistribution = distribution

	// Calculate total containers
	for _, count := range distribution {
		stats.TotalContainers += count
	}

	// Get stack and JSON-LD counts if CouchDB service is provided
	if couchdbService != nil {
		stackCount, err := GetStackCount(couchdbService)
		if err != nil {
			return nil, fmt.Errorf("failed to get stack count: %w", err)
		}
		stats.StackCount = stackCount

		jsonLdCount, err := GetJsonLdDeploymentCount(couchdbService)
		if err != nil {
			return nil, fmt.Errorf("failed to get JSON-LD deployment count: %w", err)
		}
		stats.JsonLdDeploymentCount = jsonLdCount
	}

	return stats, nil
}

// ToJSON serializes dashboard stats to JSON string.
//
// This function provides a convenient way to convert statistics to JSON
// format for API responses, logging, or persistence.
//
// Parameters:
//   - stats: Dashboard statistics to serialize
//
// Returns:
//   - string: JSON representation of statistics
//   - error: JSON marshaling errors
//
// Output Format:
//
//	{
//	  "containerDistribution": {
//	    "running": 15,
//	    "exited": 3
//	  },
//	  "stackCount": 5,
//	  "jsonLdDeploymentCount": 23,
//	  "totalContainers": 18
//	}
//
// Example:
//
//	stats, _ := GetDashboardStats(ctx, cli, couchdb)
//	jsonStr, err := ToJSON(stats)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(jsonStr)
func ToJSON(stats *Stats) (string, error) {
	jsonBytes, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal stats to JSON: %w", err)
	}
	return string(jsonBytes), nil
}
