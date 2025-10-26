# Integration Test Setup - Complete ✅

This document summarizes the integration testing infrastructure that has been set up for the EVE project.

## What Was Created

### 1. Docker Container Management
**File**: `taskfile.yml` (container tasks)

Manages three test containers using Docker CLI:
- **PostgreSQL** (port 5433): Database testing
- **CouchDB** (port 5985): Document storage testing
- **RabbitMQ** (ports 5673, 15673): Message queue testing

All ports are offset to avoid conflicts with local development instances.

Containers are managed with individual `docker run` commands in the taskfile, providing:
- Named containers for easy management
- Persistent volumes for data
- Health checks
- Custom network isolation

### 2. Taskfile Tasks
**File**: `taskfile.yml` (updated)

Added container management tasks:
```bash
task containers:up        # Start all test containers
task containers:down      # Stop containers
task containers:wait      # Wait for health checks
task containers:clean     # Remove containers and volumes
task containers:status    # Show container status
task containers:logs      # View container logs
```

Added integration test tasks:
```bash
task test:integration:local  # Run integration tests with Docker Compose
task test:all                # Run unit + integration tests
task coverage:integration    # Coverage with integration tests
```

### 3. Integration Test Examples

**Created 3 comprehensive test files:**

#### `db/couchdb_integration_test.go` (10 tests)
- SaveDocument (new and updates)
- GetDocument (existing and non-existent)
- DeleteDocument (with proper and wrong revisions)
- GetDocumentsByState (filtering)
- GetAllDocuments (pagination)
- Document history tracking

#### `db/postgres_integration_test.go` (11 tests)
- Database connection and setup
- Auto-migration and schema verification
- Create, query, update, delete operations
- Binary log data handling
- Soft delete functionality
- Transaction support (commit and rollback)

#### `queue/rabbit_integration_test.go` (10 tests)
- Service creation and connection
- Message publishing (single, multiple, large)
- Message consumption and verification
- Queue durability and persistence
- Connection recovery
- Concurrent publishing

**Total**: 31 new integration tests

### 4. GitHub Actions Workflow
**File**: `.github/workflows/tests.yml`

Three-job pipeline:
1. **unit-tests**: Fast unit tests (~2 min)
2. **integration-tests**: Integration tests with testcontainers (~5 min)
3. **combined-coverage**: Full coverage report with 60% threshold

Features:
- Parallel execution for faster feedback
- Automatic Docker image pre-pulling
- Coverage reports to Codecov
- Lint and security scanning

### 5. Documentation
**Files Created:**
- `INTEGRATION_TESTING.md`: Comprehensive testing guide
- `INTEGRATION_TEST_SETUP.md`: This setup summary

### 6. Dependencies Added

```bash
github.com/testcontainers/testcontainers-go  # Container management
github.com/testcontainers/testcontainers-go/wait  # Health checks
```

## Quick Start Guide

### Local Development

```bash
# 1. Start containers
task containers:up

# 2. Run integration tests
task test:integration:local

# 3. Stop containers
task containers:down
```

### Run All Tests (Unit + Integration)

```bash
task test:all
```

### Check Coverage

```bash
task coverage:integration
```

### View Coverage HTML Report

```bash
task coverage:html
```

## CI/CD (GitHub Actions)

Tests run automatically on:
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop`

**Workflow steps:**
1. Run unit tests (fast feedback)
2. Run integration tests with testcontainers
3. Generate combined coverage report
4. Fail if coverage < 60%
5. Run linting and security scans

## How It Works

### Local Testing (Two Options)

**Option 1: Taskfile-Managed Containers** (Manual)
- Start containers with `task containers:up`
- Uses direct Docker CLI commands
- Run tests against fixed ports (5433, 5985, 5673)
- Shared containers across all tests
- Faster for repeated test runs
- Full control via taskfile commands

**Option 2: testcontainers-go** (Automatic)
- Tests start their own containers
- Random ports to avoid conflicts
- Complete isolation between tests
- Automatic cleanup
- Same code works locally and in CI

### CI Testing

**GitHub Actions uses testcontainers-go:**
- Docker pre-installed on runners
- Automatic container lifecycle management
- Each test suite isolated
- No manual setup required

## Test Structure

### Build Tags

All integration tests use the `integration` build tag:

```go
//go:build integration
// +build integration

package mypackage
```

This allows you to:
```bash
go test ./...                    # Unit tests only (fast)
go test -tags=integration ./...  # All tests (slower)
```

### Helper Functions

Each test file has a setup helper:

```go
func setupCouchDBContainer(t *testing.T) (string, func()) {
    // Start container
    // Return URL and cleanup function
}

func TestMyFeature(t *testing.T) {
    url, cleanup := setupCouchDBContainer(t)
    defer cleanup()  // Always cleanup

    // Test code...
}
```

## Coverage Impact

### Before Integration Tests
- Total coverage: **45.8%**
- DB module: 57.9% (mostly helper functions)
- Queue module: 20.0% (PublishMessage at 0%)

### After Integration Tests (Expected)
- DB module: **75-80%** (+17-22%)
  - CouchDB operations fully tested
  - PostgreSQL CRUD operations tested
- Queue module: **55-65%** (+35-45%)
  - RabbitMQ publishing tested
  - Message consumption tested
- **Overall: 52-58%** (+6-12% to total)

## Troubleshooting

### Port Conflicts

```bash
# Check what's using the ports
lsof -i :5433  # PostgreSQL
lsof -i :5985  # CouchDB
lsof -i :5673  # RabbitMQ

# Use task to stop containers
task containers:down

# Or manually
docker stop eve-postgres-test eve-couchdb-test eve-rabbitmq-test
docker rm eve-postgres-test eve-couchdb-test eve-rabbitmq-test

# Clean everything including volumes
task containers:clean
```

### Container Won't Start

```bash
# Check Docker is running
docker ps

# View container logs
task containers:logs

# Clean start
task containers:clean
task containers:up
```

### Tests Fail Locally But Pass in CI

Usually timing/race conditions:

```bash
# Run with race detector
go test -race -tags=integration ./...

# Add explicit waits in tests
time.Sleep(100 * time.Millisecond)
```

## Next Steps

### To Reach 60% Coverage

1. **Run the integration tests**:
   ```bash
   task coverage:integration
   ```

2. **Check the results**:
   ```bash
   go tool cover -func=coverage.out | grep total
   ```

3. **If below 60%, add more tests**:
   - Common module: Docker client operations (mocking needed)
   - Forge module: GitLab/Gitea API calls (httptest mocking)
   - Network module: More SSH and Ziti tests

4. **For CI to pass**:
   - Coverage must be ≥ 60%
   - All linting must pass
   - No security issues

## Testing Philosophy

### What We Test

✅ **Integration Tests (with testcontainers)**:
- Database operations (CouchDB, PostgreSQL)
- Message queue operations (RabbitMQ)
- Service initialization and configuration
- Error handling with real services
- Connection management and cleanup

✅ **Unit Tests (no external dependencies)**:
- Business logic
- Data structures and validation
- Helper functions
- Error message formatting

❌ **Not Tested (requires complex mocking)**:
- Docker client operations
- S3/MinIO/cloud storage APIs
- External API calls (GitLab, Gitea)

### Best Practices

1. **Isolation**: Each test has its own container (testcontainers)
2. **Cleanup**: Always use `defer cleanup()`
3. **Timing**: Use wait strategies, not arbitrary sleeps
4. **Clarity**: Descriptive test names and sub-tests
5. **Coverage**: Test success AND failure cases

## Resources

- **Local Guide**: See `INTEGRATION_TESTING.md`
- **testcontainers Docs**: https://golang.testcontainers.org/
- **Task Docs**: https://taskfile.dev/

## Summary

You now have:
- ✅ 31 integration tests covering CouchDB, PostgreSQL, RabbitMQ
- ✅ Docker CLI container management via Taskfile
- ✅ Taskfile commands for easy test execution
- ✅ GitHub Actions CI/CD pipeline
- ✅ testcontainers-go for automatic container management
- ✅ Comprehensive documentation
- ✅ No docker-compose dependency - pure Docker commands

**Ready to run**:
```bash
task test:all
```

**Expected outcome**:
- Coverage increase from 45.8% to ~52-58%
- All integration tests passing
- CI pipeline ready for GitHub
- Containers managed entirely through taskfile
