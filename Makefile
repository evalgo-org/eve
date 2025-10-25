# Makefile for eve project

.PHONY: help test coverage coverage-html benchmark lint security clean all

# Default target
help:
	@echo "Available targets:"
	@echo "  make test           - Run all tests"
	@echo "  make coverage       - Run tests with coverage report"
	@echo "  make coverage-html  - Generate HTML coverage report and open in browser"
	@echo "  make benchmark      - Run benchmark tests"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make security       - Run security scanner (gosec)"
	@echo "  make clean          - Clean build artifacts and coverage reports"
	@echo "  make all            - Run tests, coverage, and linting"

# Run all tests
test:
	@echo "Running tests..."
	go test -v -race ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "\nCoverage Summary:"
	go tool cover -func=coverage.out | grep total

# Generate HTML coverage report
coverage-html: coverage
	@echo "Generating HTML coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Opening coverage report in browser..."
	@which xdg-open > /dev/null && xdg-open coverage.html || \
	which open > /dev/null && open coverage.html || \
	echo "Please open coverage.html in your browser"

# Run benchmark tests
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem -run=^$$ ./...

# Run linter
lint:
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run --timeout=5m ./...

# Run security scanner
security:
	@echo "Running gosec security scanner..."
	@which gosec > /dev/null || (echo "gosec not installed. Run: go install github.com/securego/gosec/v2/cmd/gosec@latest" && exit 1)
	gosec -fmt=text -out=security-report.txt ./...
	@cat security-report.txt

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f coverage.out coverage.html coverage.txt security-report.txt
	go clean -testcache
	@echo "Clean complete"

# Run all checks
all: test coverage lint
	@echo "\n✓ All checks passed!"

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "✓ Development tools installed"

# Run tests in watch mode (requires entr)
watch:
	@which entr > /dev/null || (echo "entr not installed. Install with: apt-get install entr / brew install entr" && exit 1)
	@echo "Watching for changes... (press Ctrl+C to stop)"
	find . -name '*.go' | entr -c make test

# Show test coverage by package
coverage-by-package: coverage
	@echo "\nCoverage by package:"
	@go tool cover -func=coverage.out | grep -v "total:" | awk '{print $$1, $$3}' | column -t

# Check coverage threshold (60%)
coverage-check: coverage
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Total coverage: $$COVERAGE%"; \
	if [ "$$(echo "$$COVERAGE < 60" | bc)" -eq 1 ]; then \
		echo "❌ Coverage below 60% threshold"; \
		exit 1; \
	else \
		echo "✓ Coverage meets 60% threshold"; \
	fi
