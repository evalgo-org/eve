# Ziti Proxy - Generic Configuration-Driven Ziti Network Proxy

A production-ready, configuration-driven HTTP proxy with load balancing, health checks, and authentication for OpenZiti zero-trust networks.

## Features

- **Configuration-Driven**: JSON-based configuration for routes, backends, and policies
- **Load Balancing**: Round-robin, weighted round-robin, and least-connections strategies
- **Health Checks**: Automatic backend health monitoring with configurable intervals
- **Authentication**: API key, JWT, and basic authentication support
- **CORS**: Full CORS support with configurable origins and methods
- **Retry Logic**: Exponential backoff with configurable retryable status codes
- **Circuit Breaker**: Automatic failure detection and recovery
- **Hot Reload**: Configuration reload without service interruption
- **Logging**: Structured JSON or text logging with request tracking
- **Middleware Chain**: Extensible middleware architecture
- **Zero-Trust Networking**: All traffic routed through OpenZiti overlay network

## Quick Start

### 1. Create Configuration File

Create `proxy-config.json`:

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8880,
    "read_timeout": "30s",
    "write_timeout": "30s",
    "idle_timeout": "60s"
  },
  "auth": {
    "type": "api-key",
    "header": "X-API-Key",
    "keys": ["your-secret-api-key"],
    "bypass": ["/health"]
  },
  "routes": [
    {
      "path": "/api/v1/*",
      "methods": ["GET", "POST"],
      "backends": [
        {
          "ziti_service": "backend-service",
          "identity_file": "./ziti-identity.json",
          "timeout": "30s"
        }
      ]
    }
  ]
}
```

### 2. Use in Your Go Application

```go
package main

import (
    "context"
    "log"
    "eve.evalgo.org/network"
)

func main() {
    // Create proxy
    proxy, err := network.NewZitiProxy("proxy-config.json")
    if err != nil {
        log.Fatal(err)
    }

    // Start proxy
    if err := proxy.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### 3. Or Build Standalone Binary

```bash
cd network/examples
go build -o ziti-proxy ziti-proxy-example.go
./ziti-proxy -config proxy-config.json
```

## Configuration Reference

### Server Configuration

```json
{
  "server": {
    "host": "0.0.0.0",           // Bind address
    "port": 8880,                 // Bind port
    "read_timeout": "30s",        // Read timeout
    "write_timeout": "30s",       // Write timeout
    "idle_timeout": "60s"         // Idle connection timeout
  }
}
```

### Authentication Configuration

#### API Key Authentication

```json
{
  "auth": {
    "type": "api-key",
    "header": "X-API-Key",
    "keys": ["key1", "key2"],
    "bypass": ["/health", "/metrics"]
  }
}
```

#### JWT Authentication

```json
{
  "auth": {
    "type": "jwt",
    "header": "Authorization",
    "jwt": {
      "secret": "your-jwt-secret",
      "algorithm": "HS256",
      "issuer": "your-issuer",
      "audience": ["your-audience"],
      "required_claims": ["sub", "exp"]
    }
  }
}
```

#### Basic Authentication

```json
{
  "auth": {
    "type": "basic",
    "keys": ["username:password"]
  }
}
```

### Route Configuration

```json
{
  "routes": [
    {
      "path": "/api/v1/*",              // Path pattern (supports wildcards and :params)
      "methods": ["GET", "POST"],        // Allowed HTTP methods
      "backends": [                      // Backend services
        {
          "ziti_service": "service-1",   // Ziti service name
          "identity_file": "./id1.json", // Ziti identity file
          "weight": 2,                   // Load balancing weight
          "priority": 1,                 // Failover priority
          "timeout": "30s",              // Request timeout
          "max_retries": 3               // Max retry attempts
        }
      ],
      "load_balancing": "weighted-round-robin",
      "strip_prefix": false,             // Strip matched prefix
      "add_prefix": "",                  // Add prefix to forwarded path
      "rewrite_host": false,             // Rewrite Host header
      "timeout": "30s"                   // Route-specific timeout
    }
  ]
}
```

### Load Balancing Strategies

- **round-robin**: Distribute requests evenly across backends
- **weighted-round-robin**: Distribute based on backend weights
- **least-connections**: Route to backend with fewest active connections

### Health Check Configuration

```json
{
  "health_check": {
    "enabled": true,
    "interval": "30s",           // Check interval
    "timeout": "5s",             // Health check timeout
    "path": "/health",           // Health check endpoint
    "expected_status": 200,      // Expected HTTP status
    "failure_count": 3,          // Failures before unhealthy
    "success_count": 2           // Successes before healthy
  }
}
```

### Retry Configuration

```json
{
  "retry": {
    "max_attempts": 3,
    "initial_interval": "1s",
    "max_interval": "10s",
    "multiplier": 2.0,
    "retryable_status": [502, 503, 504]
  }
}
```

### Circuit Breaker Configuration

```json
{
  "circuit_breaker": {
    "enabled": true,
    "failure_threshold": 5,      // Failures before opening
    "success_threshold": 2,      // Successes to close
    "timeout": "60s",            // Timeout before half-open
    "half_open_requests": 3      // Test requests in half-open
  }
}
```

### CORS Configuration

```json
{
  "cors": {
    "enabled": true,
    "allowed_origins": ["*"],
    "allowed_methods": ["GET", "POST", "PUT", "DELETE"],
    "allowed_headers": ["Content-Type", "Authorization"],
    "exposed_headers": ["Content-Length"],
    "allow_credentials": false,
    "max_age": 3600
  }
}
```

### Logging Configuration

```json
{
  "logging": {
    "enabled": true,
    "level": "info",
    "format": "json",
    "include_body": false,
    "exclude_paths": ["/health", "/metrics"]
  }
}
```

## Path Patterns

The proxy supports flexible path matching:

- **Static paths**: `/api/v1/users` - Exact match
- **Wildcards**: `/api/v1/*` - Match any subpath
- **Parameters**: `/users/:id` - Capture path parameters
- **Combined**: `/api/:version/users/*` - Multiple patterns

### Path Rewriting

```json
{
  "path": "/external/api/*",
  "strip_prefix": true,         // Removes "/external/api"
  "add_prefix": "/internal",    // Adds "/internal"
  "backends": [...]
  // Request to /external/api/users -> /internal/users
}
```

## Advanced Usage

### Multiple Backends with Failover

```json
{
  "backends": [
    {
      "ziti_service": "primary-service",
      "identity_file": "./primary.json",
      "priority": 1,
      "weight": 3
    },
    {
      "ziti_service": "secondary-service",
      "identity_file": "./secondary.json",
      "priority": 2,
      "weight": 1
    }
  ],
  "load_balancing": "weighted-round-robin"
}
```

### Route-Specific Authentication

```json
{
  "routes": [
    {
      "path": "/public/*",
      "auth": {
        "type": "none"
      }
    },
    {
      "path": "/admin/*",
      "auth": {
        "type": "jwt",
        "jwt": {
          "secret": "admin-secret",
          "required_claims": ["admin"]
        }
      }
    }
  ]
}
```

### Configuration Hot Reload

```go
proxy, _ := network.NewZitiProxy("config.json")

// Later, reload configuration without stopping
if err := proxy.Reload("config.json"); err != nil {
    log.Printf("Reload failed: %v", err)
}
```

### Status Monitoring

```go
status := proxy.GetStatus()
fmt.Printf("Proxy status: %+v\n", status)

// Output:
// {
//   "status": "running",
//   "routes": [
//     {
//       "path": "/api/v1/*",
//       "backends_total": 2,
//       "backends_healthy": 2,
//       "load_balancing": "round-robin"
//     }
//   ]
// }
```

## Migration from Custom Proxy

If you're migrating from a custom proxy (like the `caches` Ziti proxy), here's the mapping:

### Before (Custom Proxy - 178 lines)

```go
// Initialize Ziti
cfg, _ := ziti.NewConfigFromFile(identityFile)
zitiContext, _ := ziti.NewContext(cfg)

zitiTransport := &http.Transport{
    DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
        return zitiContext.Dial(serviceName)
    },
}

client := &http.Client{Transport: zitiTransport}

// Set up routes
e.GET("/pim/api/v1/product-hierarchy", func(c echo.Context) error {
    resp, err := client.Get("http://service" + c.Request().URL.Path)
    // ... handle response
})
```

### After (Generic Proxy - 30 lines)

**proxy-config.json:**
```json
{
  "server": {"host": "0.0.0.0", "port": 8880},
  "routes": [{
    "path": "/pim/api/v1/product-hierarchy",
    "methods": ["GET"],
    "backends": [{
      "ziti_service": "dev.caches.rest.px",
      "identity_file": "./btp.user.json"
    }]
  }]
}
```

**main.go:**
```go
func main() {
    proxy, _ := network.NewZitiProxy("proxy-config.json")
    proxy.Start()
}
```

## Performance Considerations

- **Connection Pooling**: Ziti connections are pooled per backend
- **Health Checks**: Run in separate goroutines, don't block requests
- **Load Balancing**: Atomic operations for thread-safe backend selection
- **Memory**: ~50-100MB base, scales with number of routes and backends
- **CPU**: Minimal overhead, mostly Ziti SDK and HTTP proxying

## Security Best Practices

1. **Store Identity Files Securely**: Use encrypted storage or secret management
2. **Rotate API Keys**: Regular key rotation via config reload
3. **Use HTTPS**: Terminate TLS before the proxy or use reverse proxy
4. **Limit CORS Origins**: Don't use `*` in production
5. **Enable Request Logging**: Monitor for suspicious patterns
6. **Use JWT**: For stronger authentication than API keys
7. **Health Check Paths**: Ensure they don't expose sensitive information

## Troubleshooting

### Backend Connection Failures

Check Ziti service configuration and identity file:
```bash
ziti edge list services
ziti edge list identities
```

### All Backends Unhealthy

Verify health check configuration:
- Is the health check path correct?
- Is the expected status code correct?
- Are timeouts appropriate?

### High Memory Usage

Reduce concurrent connections or increase timeout values to release connections faster.

### Configuration Errors

Validate JSON syntax and required fields. The proxy will log detailed error messages on startup.

## Architecture

```
┌─────────────────────────────────────────────┐
│           HTTP Request                      │
└──────────────┬──────────────────────────────┘
               ↓
┌─────────────────────────────────────────────┐
│      Middleware Chain                       │
│  ┌─────────────────────────────────┐       │
│  │ 1. Recovery (panic handling)    │       │
│  │ 2. Logging (request/response)   │       │
│  │ 3. CORS (cross-origin)          │       │
│  │ 4. Auth (API key/JWT/Basic)     │       │
│  └─────────────────────────────────┘       │
└──────────────┬──────────────────────────────┘
               ↓
┌─────────────────────────────────────────────┐
│      Router (path matching)                 │
│  - Static paths                             │
│  - Wildcard patterns                        │
│  - Parameter capture                        │
└──────────────┬──────────────────────────────┘
               ↓
┌─────────────────────────────────────────────┐
│      Load Balancer                          │
│  - Backend selection                        │
│  - Health tracking                          │
│  - Connection counting                      │
└──────────────┬──────────────────────────────┘
               ↓
┌─────────────────────────────────────────────┐
│      Ziti Transport                         │
│  - Zero-trust networking                    │
│  - Encrypted connections                    │
│  - Identity-based access                    │
└──────────────┬──────────────────────────────┘
               ↓
┌─────────────────────────────────────────────┐
│      Backend Service                        │
└─────────────────────────────────────────────┘
```

## Files

- `proxy.go` - Main proxy server implementation
- `proxy_config.go` - Configuration structures and loading
- `proxy_router.go` - Route matching logic
- `proxy_balancer.go` - Load balancing and health checks
- `proxy_middleware.go` - Authentication, CORS, logging
- `proxy-config.example.json` - Example configuration
- `examples/ziti-proxy-example.go` - Standalone proxy example

## License

Part of the EVE library - see main EVE README for license details.
