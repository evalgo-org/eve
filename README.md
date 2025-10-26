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
