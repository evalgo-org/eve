# Contributing to Eve

Thank you for your interest in contributing to Eve! This document provides guidelines and information about our development process.

## Development Setup

1. **Prerequisites**
   - Go 1.21 or higher
   - Git
   - Make (optional, but recommended)

2. **Clone the repository**
   ```bash
   git clone https://github.com/evalgo-org/eve.git
   cd eve
   ```

3. **Install dependencies**
   ```bash
   go mod download
   ```

4. **Install development tools** (optional)
   ```bash
   make install-tools
   ```

## Testing Requirements

All contributions must include appropriate tests and maintain or improve code coverage.

### Running Tests

```bash
# Quick test run
make test

# Full coverage report
make coverage

# View HTML coverage report
make coverage-html

# Run benchmarks
make benchmark
```

### Coverage Requirements

- **Minimum coverage**: 60% overall
- **New code**: Should aim for 80%+ coverage
- **Critical paths**: Must have 100% coverage
- Coverage reports are automatically generated on pull requests

### Writing Tests

1. **Test files**: Use `*_test.go` naming convention
2. **Test naming**: Use `TestFunctionName` for functions, `TestTypeName_Method` for methods
3. **Table-driven tests**: Preferred for testing multiple scenarios
4. **Mocking**: Use `httptest` for HTTP tests, `miniredis` for Redis tests
5. **Examples**: Add runnable examples for public APIs

Example test structure:
```go
func TestGetXSUAAToken(t *testing.T) {
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

## CI/CD Pipeline

Our GitHub Actions workflow runs automatically on:
- Every push to `main` or `develop` branches
- All pull requests
- Manual workflow dispatch

### Workflow Jobs

1. **Test and Coverage**
   - Runs on Go 1.21 and 1.22
   - Executes all tests with race detection
   - Generates coverage reports
   - Uploads to Codecov
   - Comments coverage on PRs

2. **Benchmarks**
   - Runs performance benchmarks
   - Reports results in workflow summary

3. **Lint**
   - Runs golangci-lint with standard configuration
   - Checks code quality and style

4. **Security**
   - Runs Gosec security scanner
   - Uploads SARIF results to GitHub Security

### Pre-commit Checks

Before submitting a pull request, ensure:

```bash
# Run all checks
make all

# Or run individually
make test         # All tests pass
make coverage     # Coverage meets threshold
make lint         # No linting errors
make security     # No security issues
```

## Pull Request Process

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Write code and tests**
   - Follow Go best practices
   - Add comprehensive tests
   - Update documentation

3. **Verify all checks pass**
   ```bash
   make all
   ```

4. **Commit your changes**
   ```bash
   git add .
   git commit -m "Add feature: description"
   ```

5. **Push to your fork**
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Create a pull request**
   - Use a clear, descriptive title
   - Reference related issues
   - Describe what changed and why
   - Wait for CI checks to complete

### Pull Request Requirements

✅ **Required for merge:**
- All tests pass on Go 1.21 and 1.22
- Coverage maintained or improved
- No linting errors
- No security issues
- Code review approval
- Documentation updated

## Code Style

We follow standard Go conventions:

- Run `gofmt` on all code
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `golangci-lint` for additional checks
- Write clear, concise comments
- Add godoc comments for all exported functions

## Documentation

- Update README.md for user-facing changes
- Add godoc comments for all public APIs
- Include runnable examples when appropriate
- Update CHANGELOG.md (if exists)

## Commit Message Guidelines

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

**Examples:**
```
feat(db): add BaseX client implementation

Implements BaseX XML database client with support for:
- Database creation
- Document upload
- XQuery execution

Closes #123

test(security): add XSUAA token validation tests

Adds comprehensive test coverage for XSUAA OAuth 2.0 flow
including token validation, scope checking, and error cases.

Coverage increased from 45% to 78% in security package.
```

## Questions?

- Open an issue for bugs or feature requests
- Start a discussion for questions
- Check existing issues and discussions first

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.
