# EVE CouchDB Examples

This document provides practical examples of using EVE's CouchDB features for real-world scenarios.

## Table of Contents

1. [Container Orchestration](#container-orchestration)
2. [Service Discovery](#service-discovery)
3. [Dependency Management](#dependency-management)
4. [Real-Time Monitoring](#real-time-monitoring)
5. [Batch Operations](#batch-operations)
6. [Graph-Based Queries](#graph-based-queries)
7. [Full-Text Search](#full-text-search)
8. [Data Migration](#data-migration)

## Container Orchestration

### Scenario: Managing Container Lifecycle

Track containers across multiple hosts with status monitoring and relationship management.

```go
package main

import (
    "fmt"
    "log"
    "eve.evalgo.org/db"
)

type Container struct {
    ID          string   `json:"_id,omitempty"`
    Rev         string   `json:"_rev,omitempty"`
    Type        string   `json:"@type"`
    Name        string   `json:"name"`
    Image       string   `json:"image"`
    Status      string   `json:"status"`
    HostedOn    string   `json:"hostedOn"`
    DependsOn   []string `json:"dependsOn,omitempty"`
    Port        int      `json:"port,omitempty"`
    Environment map[string]string `json:"environment,omitempty"`
}

func main() {
    // Initialize service
    config := db.CouchDBConfig{
        URL:             "http://localhost:5984",
        Database:        "orchestration",
        Username:        "admin",
        Password:        "password",
        CreateIfMissing: true,
    }

    service, err := db.NewCouchDBServiceFromConfig(config)
    if err != nil {
        log.Fatal(err)
    }

    // Create indexes for common queries
    err = service.CreateIndex(db.Index{
        Name:   "status-host-index",
        Fields: []string{"status", "hostedOn"},
        Type:   "json",
    })
    if err != nil {
        log.Printf("Index creation: %v", err)
    }

    // Deploy a web application stack
    containers := []Container{
        {
            ID:       "nginx-web",
            Type:     "SoftwareApplication",
            Name:     "nginx-web",
            Image:    "nginx:latest",
            Status:   "running",
            HostedOn: "host-01",
            Port:     80,
            DependsOn: []string{"app-backend"},
        },
        {
            ID:       "app-backend",
            Type:     "SoftwareApplication",
            Name:     "app-backend",
            Image:    "myapp:v1.0",
            Status:   "running",
            HostedOn: "host-02",
            Port:     8080,
            DependsOn: []string{"postgres-db"},
            Environment: map[string]string{
                "DB_HOST": "postgres-db",
                "DB_PORT": "5432",
            },
        },
        {
            ID:       "postgres-db",
            Type:     "SoftwareApplication",
            Name:     "postgres-db",
            Image:    "postgres:14",
            Status:   "running",
            HostedOn: "host-03",
            Port:     5432,
        },
    }

    // Bulk deploy all containers
    var docs []interface{}
    for _, c := range containers {
        docs = append(docs, c)
    }

    results, err := service.BulkSaveDocuments(docs)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Deployed %d containers\n", len(results))
    for _, r := range results {
        if r.OK {
            fmt.Printf("  ✓ %s\n", r.ID)
        } else {
            fmt.Printf("  ✗ %s: %s\n", r.ID, r.Reason)
        }
    }

    // Query all running containers on specific host
    query := db.NewQueryBuilder().
        Where("@type", "$eq", "SoftwareApplication").
        And().
        Where("status", "$eq", "running").
        And().
        Where("hostedOn", "$eq", "host-02").
        Build()

    hostContainers, err := db.FindTyped[Container](service, query)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("\nRunning containers on host-02: %d\n", len(hostContainers))
    for _, c := range hostContainers {
        fmt.Printf("  - %s (%s)\n", c.Name, c.Image)
    }

    // Get dependency graph for nginx-web
    graph, err := service.GetRelationshipGraph("nginx-web", "dependsOn", 5)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("\nDependency graph for nginx-web:\n")
    printGraph(graph, 0)
}

func printGraph(graph *db.RelationshipGraph, indent int) {
    prefix := ""
    for i := 0; i < indent; i++ {
        prefix += "  "
    }

    fmt.Printf("%s- %s\n", prefix, graph.NodeID)
    for _, child := range graph.Children {
        printGraph(child, indent+1)
    }
}
```

## Service Discovery

### Scenario: Dynamic Service Registry

Maintain a registry of microservices with health checks and automatic discovery.

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "time"
    "eve.evalgo.org/db"
)

type Service struct {
    ID           string    `json:"_id,omitempty"`
    Rev          string    `json:"_rev,omitempty"`
    Type         string    `json:"@type"`
    ServiceName  string    `json:"serviceName"`
    Version      string    `json:"version"`
    Host         string    `json:"host"`
    Port         int       `json:"port"`
    Protocol     string    `json:"protocol"`
    HealthCheck  string    `json:"healthCheck"`
    Status       string    `json:"status"`
    LastSeen     time.Time `json:"lastSeen"`
    Tags         []string  `json:"tags,omitempty"`
}

func main() {
    service, _ := db.NewCouchDBServiceFromConfig(db.CouchDBConfig{
        URL:             "http://localhost:5984",
        Database:        "services",
        CreateIfMissing: true,
    })

    // Create view for service discovery by name
    designDoc := db.DesignDoc{
        ID:       "_design/discovery",
        Language: "javascript",
        Views: map[string]db.View{
            "by_service": {
                Map: `function(doc) {
                    if (doc['@type'] === 'Service' && doc.status === 'healthy') {
                        emit(doc.serviceName, {
                            host: doc.host,
                            port: doc.port,
                            version: doc.version
                        });
                    }
                }`,
            },
            "by_tag": {
                Map: `function(doc) {
                    if (doc['@type'] === 'Service' && doc.tags) {
                        doc.tags.forEach(function(tag) {
                            emit(tag, {
                                serviceName: doc.serviceName,
                                host: doc.host,
                                port: doc.port
                            });
                        });
                    }
                }`,
            },
            "health_summary": {
                Map: `function(doc) {
                    if (doc['@type'] === 'Service') {
                        emit(doc.status, 1);
                    }
                }`,
                Reduce: "_sum",
            },
        },
    }

    if err := service.CreateDesignDoc(designDoc); err != nil {
        log.Printf("Design doc creation: %v", err)
    }

    // Register multiple service instances
    services := []Service{
        {
            ID:          "api-gateway-1",
            Type:        "Service",
            ServiceName: "api-gateway",
            Version:     "v2.1.0",
            Host:        "10.0.1.10",
            Port:        8080,
            Protocol:    "http",
            HealthCheck: "/health",
            Status:      "healthy",
            LastSeen:    time.Now(),
            Tags:        []string{"gateway", "production"},
        },
        {
            ID:          "api-gateway-2",
            Type:        "Service",
            ServiceName: "api-gateway",
            Version:     "v2.1.0",
            Host:        "10.0.1.11",
            Port:        8080,
            Protocol:    "http",
            HealthCheck: "/health",
            Status:      "healthy",
            LastSeen:    time.Now(),
            Tags:        []string{"gateway", "production"},
        },
        {
            ID:          "user-service-1",
            Type:        "Service",
            ServiceName: "user-service",
            Version:     "v1.3.2",
            Host:        "10.0.2.20",
            Port:        9000,
            Protocol:    "grpc",
            HealthCheck: "/grpc.health.v1.Health/Check",
            Status:      "healthy",
            LastSeen:    time.Now(),
            Tags:        []string{"backend", "users"},
        },
    }

    var docs []interface{}
    for _, s := range services {
        docs = append(docs, s)
    }

    service.BulkSaveDocuments(docs)

    // Discover all instances of api-gateway
    result, err := service.QueryView("discovery", "by_service", db.ViewOptions{
        Key: "api-gateway",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("API Gateway instances:")
    for _, row := range result.Rows {
        var instance map[string]interface{}
        json.Unmarshal([]byte(fmt.Sprintf("%v", row.Value)), &instance)
        fmt.Printf("  - %s:%v (version: %s)\n",
            instance["host"], instance["port"], instance["version"])
    }

    // Find all services with 'production' tag
    result, _ = service.QueryView("discovery", "by_tag", db.ViewOptions{
        Key: "production",
    })

    fmt.Println("\nProduction services:")
    for _, row := range result.Rows {
        fmt.Printf("  - %s\n", row.ID)
    }

    // Get health summary
    result, _ = service.QueryView("discovery", "health_summary", db.ViewOptions{
        Reduce: true,
        Group:  true,
    })

    fmt.Println("\nHealth summary:")
    for _, row := range result.Rows {
        fmt.Printf("  %s: %v\n", row.Key, row.Value)
    }

    // Monitor service changes in real-time
    go monitorServiceChanges(service)

    // Keep running
    select {}
}

func monitorServiceChanges(service *db.CouchDBService) {
    opts := db.ChangesFeedOptions{
        Since:       "now",
        Feed:        "continuous",
        IncludeDocs: true,
        Heartbeat:   60000,
        Selector: map[string]interface{}{
            "@type": "Service",
        },
    }

    changeChan, errChan, stop := service.WatchChanges(opts)
    defer stop()

    for {
        select {
        case change := <-changeChan:
            if change.Deleted {
                fmt.Printf("[%s] Service deregistered: %s\n",
                    time.Now().Format("15:04:05"), change.ID)
            } else {
                var svc Service
                json.Unmarshal(change.Doc, &svc)
                fmt.Printf("[%s] Service updated: %s (%s)\n",
                    time.Now().Format("15:04:05"), svc.ServiceName, svc.Status)
            }
        case err := <-errChan:
            log.Printf("Change feed error: %v", err)
            return
        }
    }
}
```

## Dependency Management

### Scenario: Track and Resolve Dependencies

Manage complex dependency relationships between components.

```go
package main

import (
    "fmt"
    "log"
    "eve.evalgo.org/db"
)

type Component struct {
    ID       string   `json:"_id,omitempty"`
    Rev      string   `json:"_rev,omitempty"`
    Type     string   `json:"@type"`
    Name     string   `json:"name"`
    Version  string   `json:"version"`
    Requires []string `json:"requires,omitempty"`
    UsedBy   []string `json:"usedBy,omitempty"`
}

func main() {
    service, _ := db.NewCouchDBServiceFromConfig(db.CouchDBConfig{
        URL:      "http://localhost:5984",
        Database: "dependencies",
    })

    // Define component dependencies
    components := []Component{
        {
            ID:       "frontend",
            Type:     "Component",
            Name:     "Frontend App",
            Version:  "3.0.0",
            Requires: []string{"api-client", "ui-library"},
        },
        {
            ID:       "api-client",
            Type:     "Component",
            Name:     "API Client",
            Version:  "2.5.0",
            Requires: []string{"http-lib"},
        },
        {
            ID:       "ui-library",
            Type:     "Component",
            Name:     "UI Library",
            Version:  "1.8.0",
            Requires: []string{"css-framework"},
        },
        {
            ID:      "http-lib",
            Type:    "Component",
            Name:    "HTTP Library",
            Version: "4.2.0",
        },
        {
            ID:      "css-framework",
            Type:    "Component",
            Name:    "CSS Framework",
            Version: "5.0.1",
        },
    }

    // Save all components
    var docs []interface{}
    for _, c := range components {
        docs = append(docs, c)
    }
    service.BulkSaveDocuments(docs)

    // Get all dependencies for frontend (deep traversal)
    fmt.Println("Frontend dependencies:")
    deps, err := service.GetDependencies("frontend", []string{"requires"})
    if err != nil {
        log.Fatal(err)
    }

    for id, depData := range deps {
        var comp Component
        json.Unmarshal(depData, &comp)
        fmt.Printf("  - %s (v%s)\n", comp.Name, comp.Version)
    }

    // Get full dependency tree
    graph, err := service.GetRelationshipGraph("frontend", "requires", 10)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\nFull dependency tree:")
    printDependencyTree(graph, 0)

    // Find what depends on http-lib (reverse lookup)
    dependents, err := service.GetDependents("http-lib", "requires")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\nComponents using http-lib:")
    for _, depData := range dependents {
        var comp Component
        json.Unmarshal(depData, &comp)
        fmt.Printf("  - %s\n", comp.Name)
    }

    // Simulate upgrading http-lib - find impacted components
    impactedIDs := findImpactedComponents(service, "http-lib")
    fmt.Printf("\nUpgrading http-lib will impact %d components:\n", len(impactedIDs))
    for _, id := range impactedIDs {
        fmt.Printf("  - %s\n", id)
    }
}

func printDependencyTree(graph *db.RelationshipGraph, level int) {
    indent := ""
    for i := 0; i < level; i++ {
        indent += "  "
    }
    fmt.Printf("%s└─ %s\n", indent, graph.NodeID)

    for _, child := range graph.Children {
        printDependencyTree(child, level+1)
    }
}

func findImpactedComponents(service *db.CouchDBService, componentID string) []string {
    // Use graph traversal to find all upstream dependents
    var impacted []string

    // Start with direct dependents
    directDeps, _ := service.GetDependents(componentID, "requires")
    for _, depData := range directDeps {
        var comp Component
        json.Unmarshal(depData, &comp)
        impacted = append(impacted, comp.ID)

        // Recursively find their dependents
        upstream := findImpactedComponents(service, comp.ID)
        impacted = append(impacted, upstream...)
    }

    return impacted
}
```

## Real-Time Monitoring

### Scenario: Live Container Status Dashboard

Monitor container status changes in real-time for a dashboard.

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "sync"
    "time"
    "eve.evalgo.org/db"
)

type ContainerStatus struct {
    ID         string    `json:"_id,omitempty"`
    Rev        string    `json:"_rev,omitempty"`
    Type       string    `json:"@type"`
    Name       string    `json:"name"`
    Status     string    `json:"status"`
    CPUUsage   float64   `json:"cpuUsage"`
    MemoryMB   int       `json:"memoryMB"`
    UpdatedAt  time.Time `json:"updatedAt"`
}

type Dashboard struct {
    mu         sync.RWMutex
    containers map[string]ContainerStatus
    stats      DashboardStats
}

type DashboardStats struct {
    Total   int
    Running int
    Stopped int
    Failed  int
}

func main() {
    service, _ := db.NewCouchDBServiceFromConfig(db.CouchDBConfig{
        URL:      "http://localhost:5984",
        Database: "monitoring",
    })

    // Initialize dashboard
    dashboard := &Dashboard{
        containers: make(map[string]ContainerStatus),
    }

    // Load initial state
    query := db.NewQueryBuilder().
        Where("@type", "$eq", "ContainerStatus").
        Build()

    containers, _ := db.FindTyped[ContainerStatus](service, query)
    for _, c := range containers {
        dashboard.updateContainer(c)
    }

    fmt.Println("Initial dashboard state:")
    dashboard.printStats()

    // Start real-time monitoring
    go monitorContainerChanges(service, dashboard)

    // Periodic stats display
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        dashboard.printStats()
    }
}

func monitorContainerChanges(service *db.CouchDBService, dashboard *Dashboard) {
    opts := db.ChangesFeedOptions{
        Since:       "now",
        Feed:        "continuous",
        IncludeDocs: true,
        Heartbeat:   30000,
        Selector: map[string]interface{}{
            "@type": "ContainerStatus",
        },
    }

    err := service.ListenChanges(opts, func(change db.Change) {
        if change.Deleted {
            dashboard.removeContainer(change.ID)
            fmt.Printf("[%s] Container removed: %s\n",
                time.Now().Format("15:04:05"), change.ID)
        } else {
            var status ContainerStatus
            if err := json.Unmarshal(change.Doc, &status); err == nil {
                dashboard.updateContainer(status)
                fmt.Printf("[%s] Container updated: %s (%s) CPU: %.1f%% Mem: %dMB\n",
                    time.Now().Format("15:04:05"),
                    status.Name,
                    status.Status,
                    status.CPUUsage,
                    status.MemoryMB)
            }
        }
    })

    if err != nil {
        log.Printf("Changes feed error: %v", err)
    }
}

func (d *Dashboard) updateContainer(c ContainerStatus) {
    d.mu.Lock()
    defer d.mu.Unlock()

    d.containers[c.ID] = c
    d.recalculateStats()
}

func (d *Dashboard) removeContainer(id string) {
    d.mu.Lock()
    defer d.mu.Unlock()

    delete(d.containers, id)
    d.recalculateStats()
}

func (d *Dashboard) recalculateStats() {
    d.stats = DashboardStats{}
    for _, c := range d.containers {
        d.stats.Total++
        switch c.Status {
        case "running":
            d.stats.Running++
        case "stopped":
            d.stats.Stopped++
        case "failed":
            d.stats.Failed++
        }
    }
}

func (d *Dashboard) printStats() {
    d.mu.RLock()
    defer d.mu.RUnlock()

    fmt.Printf("\n[%s] Dashboard Stats:\n", time.Now().Format("15:04:05"))
    fmt.Printf("  Total:   %d\n", d.stats.Total)
    fmt.Printf("  Running: %d\n", d.stats.Running)
    fmt.Printf("  Stopped: %d\n", d.stats.Stopped)
    fmt.Printf("  Failed:  %d\n", d.stats.Failed)
}
```

## Batch Operations

### Scenario: Configuration Rollout

Roll out configuration changes to multiple services efficiently.

```go
package main

import (
    "fmt"
    "log"
    "eve.evalgo.org/db"
)

type ServiceConfig struct {
    ID            string            `json:"_id,omitempty"`
    Rev           string            `json:"_rev,omitempty"`
    Type          string            `json:"@type"`
    ServiceName   string            `json:"serviceName"`
    Environment   string            `json:"environment"`
    Configuration map[string]string `json:"configuration"`
    Version       int               `json:"version"`
}

func main() {
    service, _ := db.NewCouchDBServiceFromConfig(db.CouchDBConfig{
        URL:      "http://localhost:5984",
        Database: "configs",
    })

    // Bulk update configuration for all production services
    selector := map[string]interface{}{
        "@type":       "ServiceConfig",
        "environment": "production",
    }

    count, err := db.BulkUpdate[ServiceConfig](service, selector, func(cfg *ServiceConfig) error {
        // Add new configuration parameter
        if cfg.Configuration == nil {
            cfg.Configuration = make(map[string]string)
        }
        cfg.Configuration["LOG_LEVEL"] = "INFO"
        cfg.Configuration["ENABLE_METRICS"] = "true"
        cfg.Version++
        return nil
    })

    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Updated %d service configurations\n", count)

    // Bulk upsert configurations (create or update)
    newConfigs := []ServiceConfig{
        {
            ID:          "api-gateway-prod",
            Type:        "ServiceConfig",
            ServiceName: "api-gateway",
            Environment: "production",
            Configuration: map[string]string{
                "PORT":           "8080",
                "MAX_CONNECTIONS": "1000",
                "TIMEOUT":        "30s",
            },
            Version: 1,
        },
        {
            ID:          "user-service-prod",
            Type:        "ServiceConfig",
            ServiceName: "user-service",
            Environment: "production",
            Configuration: map[string]string{
                "PORT":      "9000",
                "DB_POOL":   "50",
                "CACHE_TTL": "300s",
            },
            Version: 1,
        },
    }

    results, err := db.BulkUpsert(service, newConfigs, func(c ServiceConfig) string {
        return c.ID
    })

    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\nUpsert results:")
    for _, r := range results {
        if r.OK {
            fmt.Printf("  ✓ %s (rev: %s)\n", r.ID, r.Rev)
        } else {
            fmt.Printf("  ✗ %s: %s\n", r.ID, r.Reason)
        }
    }

    // Bulk retrieve configurations
    ids := []string{"api-gateway-prod", "user-service-prod", "missing-config"}
    configs, errors, err := db.BulkGet[ServiceConfig](service, ids)

    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\nRetrieved configurations:")
    for id, cfg := range configs {
        fmt.Printf("  %s (v%d): %d parameters\n",
            id, cfg.Version, len(cfg.Configuration))
    }

    if len(errors) > 0 {
        fmt.Println("\nErrors:")
        for id, err := range errors {
            fmt.Printf("  %s: %v\n", id, err)
        }
    }
}
```

## Graph-Based Queries

### Scenario: Infrastructure Topology

Query and visualize infrastructure relationships.

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "eve.evalgo.org/db"
)

type InfraNode struct {
    ID         string   `json:"_id,omitempty"`
    Rev        string   `json:"_rev,omitempty"`
    Type       string   `json:"@type"`
    NodeType   string   `json:"nodeType"` // datacenter, rack, host, vm, container
    Name       string   `json:"name"`
    ParentNode string   `json:"parentNode,omitempty"`
    ChildNodes []string `json:"childNodes,omitempty"`
    Metadata   map[string]string `json:"metadata,omitempty"`
}

func main() {
    service, _ := db.NewCouchDBServiceFromConfig(db.CouchDBConfig{
        URL:      "http://localhost:5984",
        Database: "infrastructure",
    })

    // Build infrastructure hierarchy
    nodes := []InfraNode{
        {
            ID:       "dc-us-east",
            Type:     "InfraNode",
            NodeType: "datacenter",
            Name:     "US East Datacenter",
        },
        {
            ID:         "rack-01",
            Type:       "InfraNode",
            NodeType:   "rack",
            Name:       "Rack 01",
            ParentNode: "dc-us-east",
        },
        {
            ID:         "host-01",
            Type:       "InfraNode",
            NodeType:   "host",
            Name:       "Host 01",
            ParentNode: "rack-01",
            Metadata: map[string]string{
                "cpu":    "32 cores",
                "memory": "128GB",
            },
        },
        {
            ID:         "vm-web-01",
            Type:       "InfraNode",
            NodeType:   "vm",
            Name:       "Web Server VM",
            ParentNode: "host-01",
            Metadata: map[string]string{
                "vcpu":   "4",
                "memory": "16GB",
            },
        },
        {
            ID:         "container-nginx",
            Type:       "InfraNode",
            NodeType:   "container",
            Name:       "NGINX Container",
            ParentNode: "vm-web-01",
        },
    }

    var docs []interface{}
    for _, n := range nodes {
        docs = append(docs, n)
    }
    service.BulkSaveDocuments(docs)

    // Traverse from container up to datacenter
    fmt.Println("Infrastructure path from container-nginx:")
    path, err := service.Traverse(db.TraversalOptions{
        StartID:        "container-nginx",
        RelationField:  "parentNode",
        Direction:      "outbound",
        MaxDepth:       10,
        IncludeStart:   true,
    })

    if err != nil {
        log.Fatal(err)
    }

    for i, nodeData := range path {
        var node InfraNode
        json.Unmarshal(nodeData, &node)
        indent := ""
        for j := 0; j < i; j++ {
            indent += "  "
        }
        fmt.Printf("%s└─ %s (%s)\n", indent, node.Name, node.NodeType)
    }

    // Find all containers in datacenter
    graph, err := service.GetRelationshipGraph("dc-us-east", "parentNode", 10)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\nDatacenter hierarchy:")
    printInfraGraph(service, graph, 0)

    // Count nodes by type
    query := db.NewQueryBuilder().
        Where("@type", "$eq", "InfraNode").
        Build()

    allNodes, _ := db.FindTyped[InfraNode](service, query)

    typeCounts := make(map[string]int)
    for _, node := range allNodes {
        typeCounts[node.NodeType]++
    }

    fmt.Println("\nInfrastructure summary:")
    for nodeType, count := range typeCounts {
        fmt.Printf("  %s: %d\n", nodeType, count)
    }
}

func printInfraGraph(service *db.CouchDBService, graph *db.RelationshipGraph, level int) {
    indent := ""
    for i := 0; i < level; i++ {
        indent += "  "
    }

    // Get node details
    var node InfraNode
    service.GetGenericDocument(graph.NodeID, &node)

    fmt.Printf("%s└─ %s (%s)\n", indent, node.Name, node.NodeType)

    for _, child := range graph.Children {
        printInfraGraph(service, child, level+1)
    }
}
```

## Full-Text Search

### Scenario: Log Search and Analysis

Search through application logs using CouchDB views and Mango queries.

```go
package main

import (
    "fmt"
    "log"
    "time"
    "eve.evalgo.org/db"
)

type LogEntry struct {
    ID        string    `json:"_id,omitempty"`
    Rev       string    `json:"_rev,omitempty"`
    Type      string    `json:"@type"`
    Timestamp time.Time `json:"timestamp"`
    Level     string    `json:"level"`
    Service   string    `json:"service"`
    Message   string    `json:"message"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func main() {
    service, _ := db.NewCouchDBServiceFromConfig(db.CouchDBConfig{
        URL:      "http://localhost:5984",
        Database: "logs",
    })

    // Create indexes for common queries
    service.CreateIndex(db.Index{
        Name:   "timestamp-level-index",
        Fields: []string{"timestamp", "level"},
        Type:   "json",
    })

    service.CreateIndex(db.Index{
        Name:   "service-level-index",
        Fields: []string{"service", "level"},
        Type:   "json",
    })

    // Create view for error aggregation
    designDoc := db.DesignDoc{
        ID:       "_design/logs",
        Language: "javascript",
        Views: map[string]db.View{
            "errors_by_service": {
                Map: `function(doc) {
                    if (doc['@type'] === 'LogEntry' && doc.level === 'ERROR') {
                        emit(doc.service, 1);
                    }
                }`,
                Reduce: "_sum",
            },
            "recent_errors": {
                Map: `function(doc) {
                    if (doc['@type'] === 'LogEntry' && doc.level === 'ERROR') {
                        emit(doc.timestamp, {
                            service: doc.service,
                            message: doc.message
                        });
                    }
                }`,
            },
        },
    }
    service.CreateDesignDoc(designDoc)

    // Query: Find all ERROR logs from last hour for specific service
    oneHourAgo := time.Now().Add(-1 * time.Hour)

    query := db.NewQueryBuilder().
        Where("@type", "$eq", "LogEntry").
        And().
        Where("service", "$eq", "api-gateway").
        And().
        Where("level", "$eq", "ERROR").
        And().
        Where("timestamp", "$gte", oneHourAgo).
        Sort("timestamp", "desc").
        Limit(100).
        Build()

    errors, err := db.FindTyped[LogEntry](service, query)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d errors in api-gateway (last hour)\n", len(errors))
    for _, entry := range errors {
        fmt.Printf("[%s] %s\n",
            entry.Timestamp.Format("15:04:05"),
            entry.Message)
    }

    // Query: Error count by service
    result, _ := service.QueryView("logs", "errors_by_service", db.ViewOptions{
        Reduce: true,
        Group:  true,
    })

    fmt.Println("\nError counts by service:")
    for _, row := range result.Rows {
        fmt.Printf("  %s: %v errors\n", row.Key, row.Value)
    }

    // Query: Search for specific error message pattern
    searchQuery := db.NewQueryBuilder().
        Where("@type", "$eq", "LogEntry").
        And().
        Where("level", "$eq", "ERROR").
        And().
        Where("message", "$regex", "(?i)connection.*timeout").
        Sort("timestamp", "desc").
        Limit(50).
        Build()

    timeoutErrors, _ := db.FindTyped[LogEntry](service, searchQuery)

    fmt.Printf("\nConnection timeout errors: %d\n", len(timeoutErrors))

    // Count total logs by level
    levels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
    fmt.Println("\nLog level distribution:")

    for _, level := range levels {
        count, _ := service.Count(map[string]interface{}{
            "@type": "LogEntry",
            "level":  level,
        })
        fmt.Printf("  %s: %d\n", level, count)
    }
}
```

## Data Migration

### Scenario: Migrate Data Between Environments

Efficiently migrate documents between databases with transformation.

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "eve.evalgo.org/db"
)

type Application struct {
    ID          string            `json:"_id,omitempty"`
    Rev         string            `json:"_rev,omitempty"`
    Type        string            `json:"@type"`
    Name        string            `json:"name"`
    Environment string            `json:"environment"`
    Config      map[string]string `json:"config"`
}

func main() {
    // Source (staging) database
    sourceService, _ := db.NewCouchDBServiceFromConfig(db.CouchDBConfig{
        URL:      "http://localhost:5984",
        Database: "staging",
        Username: "admin",
        Password: "password",
    })

    // Target (production) database
    targetService, _ := db.NewCouchDBServiceFromConfig(db.CouchDBConfig{
        URL:             "http://localhost:5984",
        Database:        "production",
        Username:        "admin",
        Password:        "password",
        CreateIfMissing: true,
    })

    // Get all applications from staging
    query := db.NewQueryBuilder().
        Where("@type", "$eq", "Application").
        Where("environment", "$eq", "staging").
        Build()

    stagingApps, err := db.FindTyped[Application](sourceService, query)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d applications in staging\n", len(stagingApps))

    // Transform and prepare for production
    var prodApps []interface{}
    for _, app := range stagingApps {
        // Reset ID and revision for new database
        app.Rev = ""

        // Transform environment
        app.Environment = "production"

        // Transform configuration
        if app.Config == nil {
            app.Config = make(map[string]string)
        }

        // Update environment-specific config
        app.Config["ENVIRONMENT"] = "production"
        app.Config["LOG_LEVEL"] = "WARN" // Less verbose in production

        // Remove staging-specific config
        delete(app.Config, "DEBUG_MODE")

        prodApps = append(prodApps, app)
    }

    // Bulk insert to production
    results, err := targetService.BulkSaveDocuments(prodApps)
    if err != nil {
        log.Fatal(err)
    }

    // Report results
    successCount := 0
    failCount := 0

    fmt.Println("\nMigration results:")
    for _, r := range results {
        if r.OK {
            successCount++
            fmt.Printf("  ✓ %s\n", r.ID)
        } else {
            failCount++
            fmt.Printf("  ✗ %s: %s\n", r.ID, r.Reason)
        }
    }

    fmt.Printf("\nSummary: %d succeeded, %d failed\n", successCount, failCount)

    // Verify migration with count
    prodCount, _ := targetService.Count(map[string]interface{}{
        "@type":      "Application",
        "environment": "production",
    })

    fmt.Printf("Production database now has %d applications\n", prodCount)

    // Create backup of migrated data
    backupChanges(sourceService, stagingApps)
}

func backupChanges(service *db.CouchDBService, apps []Application) {
    // Create backup document
    backup := map[string]interface{}{
        "_id":         fmt.Sprintf("migration-backup-%d", time.Now().Unix()),
        "@type":       "MigrationBackup",
        "timestamp":   time.Now(),
        "recordCount": len(apps),
        "records":     apps,
    }

    backupJSON, _ := json.Marshal(backup)
    var backupMap map[string]interface{}
    json.Unmarshal(backupJSON, &backupMap)

    _, err := service.SaveGenericDocument(backupMap)
    if err != nil {
        log.Printf("Failed to create backup: %v", err)
    } else {
        fmt.Println("\nBackup created successfully")
    }
}
```

## Additional Resources

- **EVE Documentation**: See [README.md](README.md) for complete API reference
- **CouchDB Documentation**: https://docs.couchdb.org/
- **Kivik Documentation**: https://github.com/go-kivik/kivik
- **JSON-LD Specification**: https://json-ld.org/

## Contributing

To add more examples, please submit a pull request with your use case following the format:
1. Scenario description
2. Complete, runnable code example
3. Explanation of key concepts used
