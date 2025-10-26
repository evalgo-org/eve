# Integration Testing Guide

This guide explains how to run integration tests for the EVE project both locally and in CI/CD.

## Overview

The project has two types of tests:

1. **Unit Tests**: Fast tests with no external dependencies
2. **Integration Tests**: Tests that require real services (PostgreSQL, CouchDB, RabbitMQ)

Integration tests use the `integration` build tag and can run with either:
- **testcontainers-go**: Automatic container management (works in CI and locally)
- **Docker Compose**: Manual container management (local development)

## Quick Start

### Run All Tests Locally

```bash
# 1. Start test containers
task containers:up

# 2. Wait for containers to be ready
task containers:wait

# 3. Run integration tests
task test:integration:local

# 4. Stop containers when done
task containers:down
```

Or run everything in one command:

```bash
task test:all
```

## Available Tasks

### Container Management

```bash
# Start test containers
task containers:up

# Stop containers
task containers:down

# Restart containers
task containers:restart

# View container logs
task containers:logs

# Check container status
task containers:status

# Wait for containers to be healthy
task containers:wait

# Clean containers and volumes (fresh start)
task containers:clean
```

### Running Tests

```bash
# Unit tests only (fast, no containers needed)
task test

# Integration tests (requires containers)
task test:integration

# Integration tests with automatic container setup
task test:integration:local

# All tests (unit + integration)
task test:all

# Quick tests without race detection
task test:quick
```

### Coverage Reports

```bash
# Unit test coverage
task coverage

# Integration test coverage (with containers)
task coverage:integration

# View HTML coverage report
task coverage:html

# Coverage by package
task coverage:by-package

# Check if coverage meets 60% threshold
task coverage:check
```

## Test Containers Configuration

### Docker Setup

The taskfile manages three test containers using direct Docker commands:

| Service | Port | Credentials | Purpose |
|---------|------|-------------|---------|
| PostgreSQL | 5433 | user: `testuser`<br>pass: `testpass`<br>db: `testdb` | Database integration tests |
| CouchDB | 5985 | user: `admin`<br>pass: `testpass` | Document storage tests |
| RabbitMQ | 5673 (AMQP)<br>15673 (Management) | user: `guest`<br>pass: `guest` | Message queue tests |

**Note**: Ports are offset to avoid conflicts with locally running services:
- PostgreSQL: 5433 (instead of 5432)
- CouchDB: 5985 (instead of 5984)
- RabbitMQ: 5673 (instead of 5672)

### Environment Variables

Integration tests use these environment variables:

```bash
export POSTGRES_URL="host=localhost port=5433 user=testuser password=testpass dbname=testdb sslmode=disable"
export COUCHDB_URL="http://admin:testpass@localhost:5985"
export RABBITMQ_URL="amqp://guest:guest@localhost:5673/"
```

These are automatically set by `task test:integration:local`.

## Writing Integration Tests

### File Naming

Integration test files must:
1. Have the suffix `_integration_test.go`
2. Include the build tag at the top:

```go
//go:build integration
// +build integration

package mypackage
```

### Example: CouchDB Integration Test

```go
//go:build integration
// +build integration

package db

import (
    "context"
    "fmt"
    "testing"
    "time"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
    "github.com/stretchr/testify/require"
)

func setupCouchDBContainer(t *testing.T) (string, func()) {
    ctx := context.Background()

    req := testcontainers.ContainerRequest{
        Image:        "couchdb:3.3",
        ExposedPorts: []string{"5984/tcp"},
        Env: map[string]string{
            "COUCHDB_USER":     "admin",
            "COUCHDB_PASSWORD": "testpass",
        },
        WaitingFor: wait.ForHTTP("/_up").WithPort("5984/tcp"),
    }

    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    require.NoError(t, err)

    host, _ := container.Host(ctx)
    port, _ := container.MappedPort(ctx, "5984")

    url := fmt.Sprintf("http://admin:testpass@%s:%s", host, port.Port())

    cleanup := func() {
        container.Terminate(ctx)
    }

    return url, cleanup
}

func TestCouchDB_Integration(t *testing.T) {
    url, cleanup := setupCouchDBContainer(t)
    defer cleanup()

    // Your test code here
    service, err := NewCouchDBService(eve.FlowConfig{
        CouchDBURL:   url,
        DatabaseName: "test_db",
    })
    require.NoError(t, err)
    defer service.Close()

    // Test operations...
}
```

### Example: Using Taskfile-Managed Containers

If you prefer to use the taskfile-managed containers instead of testcontainers:

```go
//go:build integration
// +build integration

package db

import (
    "os"
    "testing"
)

func TestCouchDB_WithTaskfile(t *testing.T) {
    // Get URL from environment (set by taskfile)
    url := os.Getenv("COUCHDB_URL")
    if url == "" {
        url = "http://admin:testpass@localhost:5985"
    }

    service, err := NewCouchDBService(eve.FlowConfig{
        CouchDBURL:   url,
        DatabaseName: "test_db",
    })
    require.NoError(t, err)
    defer service.Close()

    // Test operations...
}
```

## CI/CD Integration

### GitHub Actions

The `.github/workflows/tests.yml` workflow runs automatically on push and pull requests:

```yaml
name: Tests and Coverage

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  unit-tests:
    # Runs fast unit tests without containers

  integration-tests:
    # Runs integration tests with testcontainers
    # Docker is pre-installed in GitHub Actions runners

  combined-coverage:
    # Generates combined coverage report
    # Fails if coverage < 60%
```

**Key features:**
- Separate unit and integration test jobs for faster feedback
- Automatic Docker image pre-pulling to avoid timeouts
- Combined coverage report with 60% threshold
- Code quality checks (linting, security scanning)
- Coverage upload to Codecov

### testcontainers in CI

testcontainers-go works seamlessly in GitHub Actions because:

1. **Docker is pre-installed** on GitHub-hosted runners
2. **No special configuration needed** - testcontainers detects CI environment
3. **Automatic cleanup** - containers are removed after tests
4. **Parallel test execution** - each test can have its own containers

## Troubleshooting

### Containers Won't Start

```bash
# Check Docker is running
docker ps

# Check for port conflicts
lsof -i :5433  # PostgreSQL
lsof -i :5985  # CouchDB
lsof -i :5673  # RabbitMQ

# View individual container logs
docker logs eve-postgres-test
docker logs eve-couchdb-test
docker logs eve-rabbitmq-test

# Clean up and retry
task containers:clean
task containers:up
```

### Tests Timeout

```bash
# Increase container startup timeout in test
WaitingFor: wait.ForHTTP("/_up").
    WithPort("5984/tcp").
    WithStartupTimeout(120 * time.Second)  // Increase from 60s

# Or wait explicitly
task containers:wait
```

### Permission Denied

```bash
# Ensure your user can access Docker
sudo usermod -aG docker $USER
newgrp docker

# Or run with sudo (not recommended for development)
sudo task containers:up
```

### Containers Not Stopping

```bash
# Force remove containers
docker rm -f eve-postgres-test eve-couchdb-test eve-rabbitmq-test

# Remove volumes
docker volume rm eve-postgres-test-data eve-couchdb-test-data eve-rabbitmq-test-data

# Or use the clean task
task containers:clean
```

### Tests Pass Locally But Fail in CI

This usually means:
1. **Race conditions**: Use `-race` flag locally
2. **Timing issues**: Add proper wait conditions
3. **Resource limits**: CI may be slower, increase timeouts
4. **Environment differences**: Check logs in GitHub Actions

```bash
# Run with race detection locally
go test -race -tags=integration ./...

# Check GitHub Actions logs
# Go to Actions tab -> Click failed workflow -> View logs
```

## Best Practices

### 1. Use testcontainers for Isolation

```go
// Good: Each test has its own container
func TestFeature(t *testing.T) {
    url, cleanup := setupCouchDBContainer(t)
    defer cleanup()
    // Test code
}
```

### 2. Always Clean Up Resources

```go
// Good: Explicit cleanup
service, err := NewRabbitMQService(config)
require.NoError(t, err)
defer service.Close()  // Always defer cleanup
```

### 3. Wait for Services to Be Ready

```go
// Good: Wait for health check
WaitingFor: wait.ForHTTP("/_up").
    WithPort("5984/tcp").
    WithStartupTimeout(60 * time.Second)
```

### 4. Use Descriptive Test Names

```go
// Good: Clear intent
func TestCouchDBService_Integration_SaveDocument(t *testing.T) {
    t.Run("save new document", func(t *testing.T) {
        // Test code
    })

    t.Run("update existing document", func(t *testing.T) {
        // Test code
    })
}
```

### 5. Test Both Success and Failure Cases

```go
t.Run("successful operation", func(t *testing.T) {
    // Happy path
})

t.Run("handle connection error", func(t *testing.T) {
    // Error case
})
```

## Performance Tips

### 1. Run Unit Tests First

```bash
# Fast feedback loop during development
task test        # ~2 seconds
task test:all    # ~30 seconds (includes container startup)
```

### 2. Keep Containers Running During Development

```bash
# Start once
task containers:up

# Run tests multiple times (faster - no container restart)
go test -tags=integration ./db/...
go test -tags=integration ./queue/...

# View container status
task containers:status

# Stop when done
task containers:down
```

### 3. Use Test Caching

```bash
# Tests are cached if code doesn't change
go test -tags=integration ./...  # First run: slow
go test -tags=integration ./...  # Cached: fast

# Force re-run all tests
go test -count=1 -tags=integration ./...
```

### 4. Run Specific Tests

```bash
# Run single test
go test -tags=integration -run TestCouchDB_Integration ./db/...

# Run test package
go test -tags=integration ./db/...

# Run with verbose output
go test -v -tags=integration ./db/...
```

## Resources

- [testcontainers-go Documentation](https://golang.testcontainers.org/)
- [Docker CLI Documentation](https://docs.docker.com/engine/reference/commandline/cli/)
- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Assertions](https://github.com/stretchr/testify)
- [Task Documentation](https://taskfile.dev/)

## Summary

**Local Development (Taskfile approach):**
```bash
task containers:up    # Start containers with Docker
task test:all         # Run all tests
task containers:down  # Stop containers
```

**Local Development (testcontainers approach):**
```bash
go test -tags=integration ./...  # Containers managed automatically
```

**CI/CD:**
- Tests run automatically on push/PR
- testcontainers manages containers in GitHub Actions
- Coverage must be ≥ 60%
- All checks must pass

**Writing Tests:**
- Add `//go:build integration` tag
- Use testcontainers for isolation (recommended for CI)
- Or use taskfile containers for local development
- Clean up resources with `defer`
- Test both success and error cases
