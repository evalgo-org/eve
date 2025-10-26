# eve

[![Go Tests and Coverage](https://github.com/evalgo-org/eve/actions/workflows/tests.yml/badge.svg)](https://github.com/evalgo-org/eve/actions/workflows/tests.yml)
[![codecov](https://codecov.io/gh/evalgo-org/eve/branch/main/graph/badge.svg)](https://codecov.io/gh/evalgo-org/eve)
[![Go Report Card](https://goreportcard.com/badge/github.com/evalgo-org/eve)](https://goreportcard.com/report/github.com/evalgo-org/eve)

A comprehensive Go library for flow service management with integrated testing and CI/CD.

## CI/CD and Testing

This project uses GitHub Actions for continuous integration with comprehensive testing and coverage reporting:

### Automated Testing
- **Unit Tests**: Runs on every push and pull request
- **Coverage Reporting**: Automatically generates and uploads coverage reports to Codecov
- **Multi-Version Testing**: Tests against Go 1.24 and 1.25
- **Race Detection**: All tests run with `-race` flag to detect race conditions
- **Benchmarks**: Automated benchmark testing for performance monitoring
- **Security Scanning**: Gosec security scanner checks for common security issues
- **Linting**: golangci-lint ensures code quality

### Running Tests Locally

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out

# Run tests with race detection
go test -race ./...

# Run benchmarks
go test -bench=. -benchmem ./...
```

### Current Coverage
- **Overall**: ~54%
- **Target**: 60%+ for new code
- Coverage reports are automatically uploaded to Codecov
