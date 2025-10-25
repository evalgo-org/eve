# eve

[![Go Tests and Coverage](https://github.com/evalgo-org/eve/actions/workflows/go-test-coverage.yml/badge.svg)](https://github.com/evalgo-org/eve/actions/workflows/go-test-coverage.yml)
[![codecov](https://codecov.io/gh/evalgo-org/eve/branch/main/graph/badge.svg)](https://codecov.io/gh/evalgo-org/eve)
[![Go Report Card](https://goreportcard.com/badge/github.com/evalgo-org/eve)](https://goreportcard.com/report/github.com/evalgo-org/eve)

A comprehensive Go library for flow service management with integrated testing and CI/CD.

## CI/CD and Testing

This project uses GitHub Actions for continuous integration with comprehensive testing and coverage reporting:

### Automated Testing
- **Unit Tests**: Runs on every push and pull request
- **Coverage Reporting**: Automatically generates and uploads coverage reports to Codecov
- **Multi-Version Testing**: Tests against Go 1.21 and 1.22
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
- **Overall**: ~58% (db module)
- **Target**: 60%+ for new code
- Coverage reports are automatically commented on pull requests

## run sample flow-service publisher service
```
go run main.go --config example.config.yaml
```

## generate JWT token for the service
```
JWT_TOKEN=$(curl -XPOST -d '{"user_id": "1234567890"}' -H "Content-Type: application/json" -s http://localhost:8080/auth/token | jq -r .token)
```

## publish a process_id with a given state
```
curl -XPOST -d '{"process_id":"1234567890", "state": "started"}' -H "Content-Type: application/json" -H "Authorization: Bearer ${JWT_TOKEN}" -s http://localhost:8080/api/publish | jq .
```

## run consumer to deal witht the messaes from the publisher
```
go run main.go consume --config example.config.yaml
```

## get a process list
```
 curl -XGET -H "Authorization: Bearer ${JWT_TOKEN}" -s http://localhost:8080/api/processes | jq .
```

## sample response of the process list
```
{
  "count": 2,
  "processes": [
    {
      "_id": "process_0123456789",
      "_rev": "4-c71f1347463fe0a951c38d032fb3a832",
      "process_id": "0123456789",
      "state": "successful",
      "created_at": "2025-08-26T20:29:44.01016197+02:00",
      "updated_at": "2025-08-26T20:37:15.232752777+02:00",
      "history": [
        {
          "state": "started",
          "timestamp": "2025-08-26T20:29:44.01016197+02:00"
        },
        {
          "state": "running",
          "timestamp": "2025-08-26T20:36:48.804930215+02:00"
        },
        {
          "state": "running",
          "timestamp": "2025-08-26T20:36:55.976230255+02:00"
        },
        {
          "state": "successful",
          "timestamp": "2025-08-26T20:37:15.232752777+02:00"
        }
      ]
    },
    {
      "_id": "process_1234567890",
      "_rev": "3-049c68cedf3a8046d26b219f28119157",
      "process_id": "1234567890",
      "state": "failed",
      "created_at": "2025-08-26T20:08:54.165061549+02:00",
      "updated_at": "2025-08-26T20:29:20.10776054+02:00",
      "history": [
        {
          "state": "started",
          "timestamp": "2025-08-26T20:08:54.165061549+02:00"
        },
        {
          "state": "running",
          "timestamp": "2025-08-26T20:23:00.097861464+02:00"
        },
        {
          "state": "failed",
          "timestamp": "2025-08-26T20:29:20.10776054+02:00"
        }
      ]
    }
  ]
}
```
