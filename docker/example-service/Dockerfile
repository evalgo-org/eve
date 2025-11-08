# Multi-stage build for EVE example service

FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy root go.mod and go.sum for the eve module
COPY go.mod go.sum ./

# Copy the entire tracing package (part of eve.evalgo.org module)
COPY tracing/ ./tracing/

# Copy db package (dependency for tracing)
COPY db/ ./db/

# Copy semantic package (dependency for tracing)
COPY semantic/ ./semantic/

# Copy statemanager package (dependency for tracing)
COPY statemanager/ ./statemanager/

# Now copy example service source
COPY docker/example-service/*.go ./cmd/example-service/

# Download all dependencies (let Go download required toolchain automatically)
ENV GOTOOLCHAIN=auto
RUN go mod download

# Build the binary from the cmd/example-service directory
WORKDIR /build/cmd/example-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /build/example-service .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/example-service .

# Create logs directory
RUN mkdir -p /app/logs

# Expose ports
EXPOSE 8080 9091

# Run the service
CMD ["/app/example-service"]
