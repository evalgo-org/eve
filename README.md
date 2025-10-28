# eve

[![Go Tests and Coverage](https://github.com/evalgo-org/eve/actions/workflows/tests.yml/badge.svg)](https://github.com/evalgo-org/eve/actions/workflows/tests.yml)
[![codecov](https://codecov.io/gh/evalgo-org/eve/branch/main/graph/badge.svg)](https://codecov.io/gh/evalgo-org/eve)
[![Go Report Card](https://goreportcard.com/badge/github.com/evalgo-org/eve)](https://goreportcard.com/report/github.com/evalgo-org/eve)

A comprehensive Go library for flow service management with integrated testing and CI/CD.

## Quick Start

### Prerequisites
- Go 1.24 or higher
- Git
- Docker (for integration tests)
- Task (optional, install from https://taskfile.dev)
- OpenZiti Controller v1.6.5+ (for Ziti network features)

## OpenZiti Compatibility

This project uses OpenZiti for zero-trust networking. **Version compatibility between the OpenZiti Controller and SDK is critical.**

**Current Configuration:**
- **SDK Version:** `github.com/openziti/sdk-golang v1.2.2`
- **Minimum Controller Version:** v1.6.0
- **Recommended Controller Version:** v1.6.5 - v1.6.7
- **Status:** ✅ Production Ready

⚠️ **Important:** SDK versions v1.2.3 and later require OpenZiti Controller v1.6.8+ due to HA/OIDC authentication changes. See [OPENZITI_COMPATIBILITY.md](./OPENZITI_COMPATIBILITY.md) for detailed compatibility information and upgrade paths.

### Version Checking

The `network` package includes automatic version checking:

```go
import "eve.evalgo.org/network"

// Check compatibility (logs warnings/recommendations)
network.LogCompatibilityCheck("path/to/identity.json")

// Enforce compatibility (panics if incompatible)
network.MustBeCompatible("path/to/identity.json")
```

### Context Caching & Duplicate Connection Messages

The `network` package implements automatic Ziti context caching for efficiency. When multiple routes/backends use the same Ziti identity, they automatically share a single connection.

**Expected Behavior:** You may see INFO messages like `"connection to tls:controller:port already established, closing duplicate connection"` during startup. **This is normal and correct behavior** - the Ziti SDK is detecting and cleaning up duplicate connection attempts, keeping only one active connection per identity.

This optimization means:
- ✅ Multiple routes share one Ziti context
- ✅ Reduced memory and network overhead
- ✅ Automatic duplicate detection by Ziti SDK
- ℹ️ Informational messages are expected, not errors

### Installation

```bash
git clone https://github.com/evalgo-org/eve.git
cd eve
go mod download
```

### Running Tests

```bash
# Unit tests only (fast)
go test ./...

# All tests including integration (requires Docker)
task test:all

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Testing

### Test Types

The project has two types of tests:

1. **Unit Tests**: Fast tests with no external dependencies
2. **Integration Tests**: Tests requiring real services (PostgreSQL, CouchDB, RabbitMQ)

Integration tests use the `integration` build tag:

```go
//go:build integration
// +build integration
```

### Running Tests Locally

```bash
# Unit tests only
go test ./...

# Integration tests with testcontainers (automatic)
go test -tags=integration ./...

# Or use taskfile for managed containers
task containers:up       # Start test containers
task test:integration    # Run integration tests
task containers:down     # Stop containers
```

### Test Containers

Integration tests use these services:

| Service | Port | Credentials | Purpose |
|---------|------|-------------|---------|
| PostgreSQL | 5433 | user: `testuser`<br>pass: `testpass` | Database testing |
| CouchDB | 5985 | user: `admin`<br>pass: `testpass` | Document storage |
| RabbitMQ | 5673 (AMQP)<br>15673 (Mgmt) | user: `guest`<br>pass: `guest` | Message queue |

**Note**: Ports are offset to avoid conflicts with local services.

### Container Management

```bash
# Start all test containers
task containers:up

# Check container status
task containers:status

# View logs
task containers:logs

# Stop containers
task containers:down

# Clean everything (containers + volumes)
task containers:clean
```

### Coverage

- **Current**: ~54%
- **Target**: 60%+ for new code
- Coverage reports automatically upload to Codecov

```bash
# Generate coverage report
task coverage

# View HTML report
task coverage:html

# Check coverage threshold
task coverage:check
```

### Writing Tests

**Test file naming:**
- Unit tests: `*_test.go`
- Integration tests: `*_integration_test.go` with `//go:build integration` tag

**Example integration test:**

```go
//go:build integration
// +build integration

package db

import (
    "testing"
    "github.com/testcontainers/testcontainers-go"
    "github.com/stretchr/testify/require"
)

func TestCouchDB_Integration(t *testing.T) {
    // testcontainers automatically manages container lifecycle
    url, cleanup := setupCouchDBContainer(t)
    defer cleanup()

    service, err := NewCouchDBService(config)
    require.NoError(t, err)
    defer service.Close()

    // Test operations...
}
```

## CI/CD

### GitHub Actions

Automated testing runs on:
- Every push to `main` or `develop`
- All pull requests
- Manual workflow dispatch

### Workflow Jobs

1. **Unit Tests** - Fast tests without external dependencies
   - Tests against Go 1.24 and 1.25
   - Race detection enabled
   - ~2 minutes

2. **Integration Tests** - Tests with testcontainers
   - Real service instances (PostgreSQL, CouchDB, RabbitMQ)
   - Docker images pre-pulled to avoid timeouts
   - ~5 minutes

3. **Combined Coverage** - Comprehensive coverage report
   - Uploads to Codecov
   - Fails if coverage < 60%
   - Comments on pull requests

4. **Code Quality Checks**
   - **Linting**: golangci-lint with standard config
   - **Security**: Gosec security scanner
   - **Benchmarks**: Performance monitoring

## Contributing

### Development Workflow

1. **Fork and clone** the repository

2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Write code and tests**
   - Follow Go best practices
   - Add comprehensive tests (aim for 80%+ coverage)
   - Update documentation

4. **Run all checks locally**
   ```bash
   task test         # All tests pass
   task coverage     # Coverage meets threshold
   task lint         # No linting errors
   ```

5. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

6. **Push and create PR**
   ```bash
   git push origin feature/your-feature-name
   ```

### Pull Request Requirements

✅ **Required for merge:**
- All tests pass on Go 1.24 and 1.25
- Coverage maintained or improved
- No linting errors
- No security issues
- Code review approval
- Documentation updated

### Commit Message Format

Follow conventional commits:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Maintenance tasks

**Example:**
```
feat(db): add BaseX client implementation

Implements BaseX XML database client with support for:
- Database creation
- Document upload
- XQuery execution

Closes #123
```

### Code Style

- Run `gofmt` and `goimports` on all code
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Add godoc comments for all exported functions
- Use table-driven tests for multiple scenarios

### Testing Requirements

- **Minimum coverage**: 60% overall
- **New code**: Aim for 80%+ coverage
- **Critical paths**: Must have 100% coverage

**Example table-driven test:**

```go
func TestGetToken(t *testing.T) {
    tests := []struct {
        name        string
        tokenURL    string
        expectError bool
    }{
        {"valid request", "https://example.com/oauth/token", false},
        {"invalid URL", "://invalid", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Docker Container Reference

### Quick Commands

```bash
# Start containers
task containers:up

# Stop containers
task containers:down

# Restart
task containers:restart

# Clean everything
task containers:clean

# Monitor
task containers:status
task containers:logs
task containers:logs:follow
```

### Container Details

#### PostgreSQL
```bash
# Container name
eve-postgres-test

# Connection string
host=localhost port=5433 user=testuser password=testpass dbname=testdb sslmode=disable

# Interactive access
docker exec -it eve-postgres-test psql -U testuser -d testdb

# View logs
docker logs eve-postgres-test
```

#### CouchDB
```bash
# Container name
eve-couchdb-test

# URL
http://admin:testpass@localhost:5985

# Interactive access
curl http://admin:testpass@localhost:5985/_all_dbs

# View logs
docker logs eve-couchdb-test
```

#### RabbitMQ
```bash
# Container name
eve-rabbitmq-test

# AMQP URL
amqp://guest:guest@localhost:5673/

# Management UI
http://localhost:15673
(user: guest, password: guest)

# View status
docker exec -it eve-rabbitmq-test rabbitmqctl status

# View logs
docker logs eve-rabbitmq-test
```

### Troubleshooting

#### Port Conflicts
```bash
# Check what's using the ports
lsof -i :5433  # PostgreSQL
lsof -i :5985  # CouchDB
lsof -i :5673  # RabbitMQ

# Clean and restart
task containers:clean
task containers:up
```

#### Container Won't Start
```bash
# Check Docker is running
docker ps

# View container logs
task containers:logs

# Clean start
task containers:clean
task containers:up
```

#### Tests Fail Locally
```bash
# Run with race detector
go test -race -tags=integration ./...

# Check container health
task containers:status
docker inspect eve-postgres-test --format='{{.State.Health.Status}}'
```

## CouchDB Features

The `eve.evalgo.org/db` package provides comprehensive CouchDB integration with support for advanced querying, graph traversal, real-time updates, and JSON-LD validation.

### Quick Start

```go
import "eve.evalgo.org/db"

// Create service from config
config := db.CouchDBConfig{
    URL:             "http://localhost:5984",
    Database:        "myapp",
    Username:        "admin",
    Password:        "password",
    CreateIfMissing: true,
}

service, err := db.NewCouchDBServiceFromConfig(config)
if err != nil {
    log.Fatal(err)
}
defer service.Close()
```

### Generic Document Operations

Type-safe document operations using Go generics:

```go
type Container struct {
    ID       string `json:"_id,omitempty"`
    Rev      string `json:"_rev,omitempty"`
    Type     string `json:"@type"`
    Name     string `json:"name"`
    Status   string `json:"status"`
    HostedOn string `json:"hostedOn"`
}

// Save document with type safety
container := Container{
    ID:       "container-123",
    Type:     "SoftwareApplication",
    Name:     "nginx",
    Status:   "running",
    HostedOn: "host-456",
}

response, err := db.SaveDocument(service, container)

// Retrieve with type safety
retrieved, err := db.GetDocument[Container](service, "container-123")

// Query by type
containers, err := db.GetDocumentsByType[Container](service, "SoftwareApplication")
```

### MapReduce Views

Create and query CouchDB views for efficient data access:

```go
// Create design document with views
designDoc := db.DesignDoc{
    ID:       "_design/graphium",
    Language: "javascript",
    Views: map[string]db.View{
        "containers_by_host": {
            Map: `function(doc) {
                if (doc['@type'] === 'SoftwareApplication' && doc.hostedOn) {
                    emit(doc.hostedOn, {name: doc.name, status: doc.status});
                }
            }`,
        },
        "container_count": {
            Map: `function(doc) {
                if (doc['@type'] === 'SoftwareApplication') {
                    emit(doc.hostedOn, 1);
                }
            }`,
            Reduce: "_sum",
        },
    },
}

err := service.CreateDesignDoc(designDoc)

// Query view
opts := db.ViewOptions{
    Key:         "host-123",
    IncludeDocs: true,
    Limit:       50,
}
result, err := service.QueryView("graphium", "containers_by_host", opts)
```

### Mango Queries with QueryBuilder

Build complex queries with a fluent API:

```go
// Fluent query builder
query := db.NewQueryBuilder().
    Where("status", "eq", "running").
    And().
    Where("location", "regex", "^us-east").
    Select("_id", "name", "status", "hostedOn").
    Sort("name", "asc").
    Limit(50).
    Build()

results, err := service.Find(query)

// Type-safe queries
containers, err := db.FindTyped[Container](service, query)

// Simple count
count, err := service.Count(map[string]interface{}{
    "status": "running",
    "@type":  "SoftwareApplication",
})
```

### Index Management

Create indexes for query performance:

```go
// Create compound index
index := db.Index{
    Name:   "status-location-index",
    Fields: []string{"status", "location"},
    Type:   "json",
}
err := service.CreateIndex(index)

// List all indexes
indexes, err := service.ListIndexes()

// Ensure index exists (idempotent)
created, err := service.EnsureIndex(index)
```

### Graph Traversal

Navigate relationships between documents:

```go
// Find all containers on a host (reverse traversal)
opts := db.TraversalOptions{
    StartID:       "host-123",
    Depth:         1,
    RelationField: "hostedOn",
    Direction:     "reverse",
}
containers, err := service.Traverse(opts)

// Find host and datacenter for container (forward traversal)
opts = db.TraversalOptions{
    StartID:       "container-456",
    Depth:         2,
    RelationField: "hostedOn",
    Direction:     "forward",
}
related, err := service.Traverse(opts)

// Type-safe traversal
typedContainers, err := db.TraverseTyped[Container](service, opts)

// Find dependents
dependents, err := service.GetDependents("host-123", "hostedOn")

// Find dependencies
dependencies, err := service.GetDependencies("container-456",
    []string{"hostedOn", "dependsOn", "network"})

// Build relationship graph
graph, err := service.GetRelationshipGraph("container-456", "hostedOn", 3)
fmt.Printf("Graph has %d nodes and %d edges\n",
    len(graph.Nodes), len(graph.Edges))
```

### Bulk Operations

Efficient batch processing:

```go
// Bulk save
containers := []interface{}{
    Container{ID: "c1", Name: "nginx", Status: "running"},
    Container{ID: "c2", Name: "redis", Status: "running"},
    Container{ID: "c3", Name: "postgres", Status: "stopped"},
}

results, err := service.BulkSaveDocuments(containers)
for _, result := range results {
    if result.OK {
        fmt.Printf("Saved %s with rev %s\n", result.ID, result.Rev)
    }
}

// Bulk delete
deleteOps := []db.BulkDeleteDoc{
    {ID: "c1", Rev: "1-abc", Deleted: true},
    {ID: "c2", Rev: "2-def", Deleted: true},
}
results, err = service.BulkDeleteDocuments(deleteOps)

// Bulk get with type safety
ids := []string{"c1", "c2", "c3"}
docs, errors, err := db.BulkGet[Container](service, ids)

// Bulk update with function
selector := map[string]interface{}{"status": "running"}
count, err := db.BulkUpdate[Container](service, selector,
    func(c *Container) error {
        c.Status = "stopped"
        return nil
    })
```

### Real-Time Changes

Monitor database changes in real-time:

```go
// Listen to changes (blocking)
opts := db.ChangesFeedOptions{
    Since:       "now",
    Feed:        "continuous",
    IncludeDocs: true,
    Selector: map[string]interface{}{
        "@type": "SoftwareApplication",
    },
}

err := service.ListenChanges(opts, func(change db.Change) {
    if change.Deleted {
        fmt.Printf("Container %s deleted\n", change.ID)
    } else {
        fmt.Printf("Container %s changed\n", change.ID)
        var container Container
        json.Unmarshal(change.Doc, &container)
        fmt.Printf("  Status: %s\n", container.Status)
    }
})

// Channel-based watching (non-blocking)
changeChan, errChan, stop := service.WatchChanges(opts)
defer stop()

for {
    select {
    case change := <-changeChan:
        // Process change
    case err := <-errChan:
        log.Printf("Error: %v", err)
        return
    }
}

// Polling-based sync
lastSeq := "0"
for {
    opts := db.ChangesFeedOptions{
        Since: lastSeq,
        Feed:  "normal",
        Limit: 100,
    }
    changes, newSeq, err := service.GetChanges(opts)
    // Process changes
    lastSeq = newSeq
    time.Sleep(5 * time.Second)
}
```

### JSON-LD Support

Validate and manipulate JSON-LD documents:

```go
// Validate JSON-LD
doc := map[string]interface{}{
    "@context": "https://schema.org",
    "@type":    "SoftwareApplication",
    "@id":      "urn:container:nginx-1",
    "name":     "nginx",
}

err := db.ValidateJSONLD(doc, "https://schema.org")

// Expand JSON-LD
expanded, err := db.ExpandJSONLD(doc)

// Compact JSON-LD
compacted, err := db.CompactJSONLD(expanded, "https://schema.org")

// Normalize for hashing/comparison
normalized, err := db.NormalizeJSONLD(doc)
hash := sha256.Sum256([]byte(normalized))

// Helper functions
docType, err := db.ExtractJSONLDType(doc)
doc = db.SetJSONLDContext(doc, "https://schema.org")
```

### Database Management

Manage databases and get statistics:

```go
// Create database
err := db.CreateDatabaseFromURL("http://admin:pass@localhost:5984", "newdb")

// Check existence
exists, err := db.DatabaseExistsFromURL("http://admin:pass@localhost:5984", "mydb")

// Get database info
info, err := service.GetDatabaseInfo()
fmt.Printf("Database: %s\n", info.DBName)
fmt.Printf("Documents: %d active, %d deleted\n",
    info.DocCount, info.DocDelCount)
fmt.Printf("Disk: %.2f MB, Data: %.2f MB\n",
    float64(info.DiskSize)/1024/1024,
    float64(info.DataSize)/1024/1024)

// Compact database
err = service.CompactDatabase()

// Monitor compaction
for {
    info, _ := service.GetDatabaseInfo()
    if !info.CompactRunning {
        break
    }
    time.Sleep(10 * time.Second)
}

// Delete database
err = db.DeleteDatabaseFromURL("http://admin:pass@localhost:5984", "olddb")
```

### Error Handling

Comprehensive error types:

```go
err := service.GetDocument("missing-doc")
if err != nil {
    if couchErr, ok := err.(*db.CouchDBError); ok {
        switch {
        case couchErr.IsNotFound():
            fmt.Println("Document not found")
        case couchErr.IsConflict():
            fmt.Println("Revision conflict - retry needed")
        case couchErr.IsUnauthorized():
            fmt.Println("Authentication failed")
        default:
            fmt.Printf("CouchDB error %d: %s\n",
                couchErr.StatusCode, couchErr.Reason)
        }
    }
}
```

### Configuration Options

Advanced configuration:

```go
config := db.CouchDBConfig{
    URL:             "https://couchdb.example.com:6984",
    Database:        "production",
    Username:        "admin",
    Password:        os.Getenv("COUCHDB_PASSWORD"),
    MaxConnections:  100,
    Timeout:         30000, // milliseconds
    CreateIfMissing: false,
    TLS: &db.TLSConfig{
        Enabled:  true,
        CAFile:   "/etc/ssl/certs/ca.crt",
        CertFile: "/etc/ssl/certs/client.crt",
        KeyFile:  "/etc/ssl/private/client.key",
    },
}

service, err := db.NewCouchDBServiceFromConfig(config)
```

### Feature Summary

| Category | Functions | Description |
|----------|-----------|-------------|
| **Generic Documents** | 8 functions | Type-safe CRUD with Go generics |
| **View Management** | 5 functions | MapReduce view creation and querying |
| **Mango Queries** | 3 functions + Builder | MongoDB-style declarative queries |
| **Index Management** | 4 functions | Performance optimization indexes |
| **Graph Traversal** | 5 functions | Navigate document relationships |
| **Bulk Operations** | 5 functions | Batch save/delete/update/upsert |
| **Change Feeds** | 4 functions | Real-time change notifications |
| **JSON-LD** | 6 functions | Semantic data validation |
| **Database Utils** | 5 functions | Management and statistics |
| **Error Handling** | 3 helpers | Structured error types |

## Available Tasks

```bash
# Testing
task test                     # Unit tests
task test:integration         # Integration tests with testcontainers
task test:integration:local   # Integration tests with Docker containers
task test:all                 # All tests (unit + integration)
task test:quick               # Quick tests without race detection

# Coverage
task coverage                 # Unit test coverage
task coverage:integration     # Integration test coverage
task coverage:html            # HTML coverage report
task coverage:check           # Check 60% threshold

# Container Management
task containers:up            # Start all containers
task containers:down          # Stop containers
task containers:restart       # Restart containers
task containers:status        # Show container status
task containers:logs          # View logs
task containers:logs:follow   # Follow logs in real-time
task containers:wait          # Wait for health checks
task containers:clean         # Remove containers and volumes

# Code Quality
task lint                     # Run golangci-lint
task security                 # Run security scan
task benchmark                # Run benchmarks
```

## Resources

- **Project Repository**: https://github.com/evalgo-org/eve
- **Go Documentation**: https://golang.org/doc/
- **testcontainers-go**: https://golang.testcontainers.org/
- **Task**: https://taskfile.dev/
- **Codecov**: https://codecov.io/gh/evalgo-org/eve

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.
