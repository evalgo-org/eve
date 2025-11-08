# Makefile for eve project

.PHONY: help test coverage coverage-html benchmark lint security clean all
.PHONY: docker-up docker-down docker-logs docker-restart docker-clean docker-traffic docker-test
.PHONY: grafana prometheus alertmanager postgres minio traces status build metrics

# Default target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Development:"
	@echo "  make test           - Run all tests"
	@echo "  make coverage       - Run tests with coverage report"
	@echo "  make coverage-html  - Generate HTML coverage report and open in browser"
	@echo "  make benchmark      - Run benchmark tests"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make security       - Run security scanner (gosec)"
	@echo "  make clean          - Clean build artifacts and coverage reports"
	@echo "  make all            - Run tests, coverage, and linting"
	@echo ""
	@echo "Docker Compose:"
	@echo "  make docker-up      - Start all services"
	@echo "  make docker-down    - Stop all services"
	@echo "  make docker-restart - Restart all services"
	@echo "  make docker-logs    - View logs from all services"
	@echo "  make docker-clean   - Stop and remove all data"
	@echo "  make docker-traffic - Generate test traffic"
	@echo "  make docker-test    - Run smoke tests"
	@echo ""
	@echo "Service Access:"
	@echo "  make grafana        - Open Grafana in browser"
	@echo "  make prometheus     - Open Prometheus in browser"
	@echo "  make alertmanager   - Open AlertManager in browser"
	@echo "  make minio          - Open MinIO console in browser"
	@echo "  make postgres       - Connect to PostgreSQL"
	@echo "  make traces         - View recent traces"
	@echo "  make status         - Show service status"
	@echo "  make metrics        - View current metrics"

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

# ============================================================================
# Docker Compose Targets
# ============================================================================

# Start all services
docker-up:
	@echo "Starting EVE observability stack..."
	docker compose up -d
	@echo ""
	@echo "Services starting... Wait 30 seconds for initialization."
	@echo ""
	@echo "Access URLs:"
	@echo "  Grafana:      http://localhost:3000 (admin/admin)"
	@echo "  Prometheus:   http://localhost:9090"
	@echo "  AlertManager: http://localhost:9093"
	@echo "  MinIO:        http://localhost:9001 (minioadmin/minioadmin)"
	@echo "  Example API:  http://localhost:8080"
	@echo "  Metrics:      http://localhost:9091/metrics"
	@echo ""

# Stop all services
docker-down:
	@echo "Stopping EVE observability stack..."
	docker compose down

# Restart all services
docker-restart:
	@echo "Restarting EVE observability stack..."
	docker compose restart

# View logs
docker-logs:
	docker compose logs -f

# Complete cleanup
docker-clean:
	@echo "Stopping and removing all data..."
	@read -p "This will DELETE all traces, metrics, and logs. Continue? [y/N] " confirm && [ "$$confirm" = "y" ]
	docker compose down -v --rmi local
	@echo "Cleanup complete."

# Generate test traffic
docker-traffic:
	@echo "Generating test traffic to example-service..."
	@echo "Press Ctrl+C to stop"
	@bash -c 'while true; do \
		for i in {1..9}; do curl -s -X POST http://localhost:8080/v1/api/workflow/create > /dev/null & done; \
		curl -s -X POST http://localhost:8080/v1/api/workflow/slow > /dev/null & \
		if [ $$((RANDOM % 10)) -eq 0 ]; then curl -s -X POST http://localhost:8080/v1/api/workflow/error > /dev/null & fi; \
		sleep 1; \
	done'

# Run smoke tests
docker-test:
	@echo "Running smoke tests..."
	@echo ""
	@echo "1. Testing example-service health..."
	@curl -s http://localhost:8080/health | grep -q "healthy" && echo "✓ Service is healthy" || echo "✗ Service check failed"
	@echo ""
	@echo "2. Testing metrics endpoint..."
	@curl -s http://localhost:9091/metrics | grep -q "eve_tracing" && echo "✓ Metrics endpoint working" || echo "✗ Metrics check failed"
	@echo ""
	@echo "3. Testing Prometheus..."
	@curl -s http://localhost:9090/-/healthy | grep -q "Prometheus" && echo "✓ Prometheus is healthy" || echo "✗ Prometheus check failed"
	@echo ""
	@echo "4. Testing AlertManager..."
	@curl -s http://localhost:9093/-/healthy && echo "✓ AlertManager is healthy" || echo "✗ AlertManager check failed"
	@echo ""
	@echo "5. Testing MinIO..."
	@curl -s http://localhost:9000/minio/health/live && echo "✓ MinIO is healthy" || echo "✗ MinIO check failed"
	@echo ""
	@echo "6. Testing PostgreSQL..."
	@docker exec eve-postgres pg_isready -U eve_user -q && echo "✓ PostgreSQL is ready" || echo "✗ PostgreSQL check failed"
	@echo ""
	@echo "7. Creating test workflow..."
	@curl -s -X POST http://localhost:8080/v1/api/workflow/create | grep -q "completed" && echo "✓ Workflow creation successful" || echo "✗ Workflow creation failed"
	@echo ""
	@echo "8. Checking trace in database..."
	@docker exec eve-postgres psql -U eve_user -d eve_traces -tAc "SELECT COUNT(*) FROM action_executions WHERE started_at > NOW() - INTERVAL '1 minute'" | grep -q -v "^0$$" && echo "✓ Trace stored in database" || echo "✗ No recent traces found"
	@echo ""
	@echo "All tests complete!"

# Open Grafana
grafana:
	@echo "Opening Grafana (admin/admin)..."
	@which xdg-open > /dev/null 2>&1 && xdg-open http://localhost:3000 || \
	which open > /dev/null 2>&1 && open http://localhost:3000 || \
	echo "Please open http://localhost:3000"

# Open Prometheus
prometheus:
	@echo "Opening Prometheus..."
	@which xdg-open > /dev/null 2>&1 && xdg-open http://localhost:9090 || \
	which open > /dev/null 2>&1 && open http://localhost:9090 || \
	echo "Please open http://localhost:9090"

# Open AlertManager
alertmanager:
	@echo "Opening AlertManager..."
	@which xdg-open > /dev/null 2>&1 && xdg-open http://localhost:9093 || \
	which open > /dev/null 2>&1 && open http://localhost:9093 || \
	echo "Please open http://localhost:9093"

# Open MinIO
minio:
	@echo "Opening MinIO console (minioadmin/minioadmin)..."
	@which xdg-open > /dev/null 2>&1 && xdg-open http://localhost:9001 || \
	which open > /dev/null 2>&1 && open http://localhost:9001 || \
	echo "Please open http://localhost:9001"

# Connect to PostgreSQL
postgres:
	@echo "Connecting to PostgreSQL..."
	docker exec -it eve-postgres psql -U eve_user -d eve_traces

# View recent traces
traces:
	@echo "Recent traces (last 10):"
	@docker exec eve-postgres psql -U eve_user -d eve_traces -c "SELECT correlation_id, operation_id, service_id, action_type, action_status, duration_ms, started_at FROM action_executions ORDER BY started_at DESC LIMIT 10;"

# Show service status
status:
	@echo "Service Status:"
	@docker compose ps

# Build example service
build:
	@echo "Building example service..."
	docker compose build example-service

# View metrics
metrics:
	@echo "Current metrics from example-service:"
	@curl -s http://localhost:9091/metrics | grep "^eve_tracing"
