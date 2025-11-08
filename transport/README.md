# WHEN Transport Layer

Multi-transport support for WHEN semantic action executor, enabling HTTP requests over different network layers: direct HTTP/HTTPS, SSH tunnels, and OpenZiti overlay networks.

## Features

- **HTTP/HTTPS Transport**: Direct connections with connection pooling
- **SSH Tunnel Transport**: HTTP over SSH tunnels for secure remote access
- **OpenZiti Transport**: Zero-trust overlay networking with identity-based access control
- **Automatic URL Scheme Routing**: Seamless transport selection based on URL scheme
- **Connection Pooling**: Efficient resource management across all transports

## URL Schemes

The transport manager automatically routes requests based on URL scheme:

| URL Scheme | Transport | Description |
|------------|-----------|-------------|
| `http://` | HTTPTransport | Standard HTTP |
| `https://` | HTTPTransport | Standard HTTPS |
| `ssh://` | SSHTunnelTransport | HTTP over SSH tunnel |
| `ssh+http://` | SSHTunnelTransport | Explicit HTTP over SSH |
| `ssh+https://` | SSHTunnelTransport | HTTPS over SSH tunnel |
| `ziti://` | ZitiTransport | HTTP over OpenZiti |
| `ziti+http://` | ZitiTransport | Explicit HTTP over OpenZiti |

## Configuration

### Environment Variables

#### HTTP Transport

```bash
# HTTP client configuration
WHEN_HTTP_TIMEOUT=30                    # Request timeout in seconds (default: 30)
WHEN_MAX_IDLE_CONNS=100                # Max idle connections (default: 100)
WHEN_MAX_IDLE_CONNS_PER_HOST=10        # Max idle per host (default: 10)
WHEN_IDLE_CONN_TIMEOUT=90              # Idle timeout in seconds (default: 90)
```

#### SSH Transport

```bash
# SSH tunnel configuration
WHEN_SSH_HOST=example.com              # SSH server hostname (required)
WHEN_SSH_USER=root                     # SSH username (default: root)
WHEN_SSH_PORT=22                       # SSH port (default: 22)
WHEN_SSH_KEY_FILE=/path/to/key         # Path to SSH private key
WHEN_SSH_PASSWORD=secret               # SSH password (if not using key)
WHEN_SSH_KNOWN_HOSTS=/path/to/known   # Path to known_hosts file
WHEN_SSH_TIMEOUT=30                    # SSH connection timeout (default: 30)
```

**Note**: Either `WHEN_SSH_KEY_FILE` or `WHEN_SSH_PASSWORD` must be set.

#### OpenZiti Transport

```bash
# Ziti overlay network configuration
WHEN_ZITI_IDENTITY_FILE=/path/to/identity.json    # Ziti identity file
WHEN_ZITI_IDENTITY_JSON='{"id":"...","ztAPI":...}' # Or identity as JSON
WHEN_ZITI_TIMEOUT=30                              # Request timeout (default: 30)
```

**Note**: Either `WHEN_ZITI_IDENTITY_FILE` or `WHEN_ZITI_IDENTITY_JSON` must be set.

## Usage Examples

### Direct HTTP/HTTPS

Standard HTTP requests work without any configuration:

```go
import "eve.evalgo.org/transport"

// Create manager with HTTP transport
mgr, err := transport.DefaultManager(ctx)
if err != nil {
    log.Fatal(err)
}
defer mgr.Close()

// Make HTTP request
req, _ := http.NewRequest("GET", "http://example.com/api/data", nil)
resp, err := mgr.RoundTrip(req)
```

### SSH Tunnel

Route HTTP requests through SSH tunnels:

```bash
# Set SSH configuration
export WHEN_SSH_HOST=jumphost.example.com
export WHEN_SSH_USER=admin
export WHEN_SSH_KEY_FILE=/home/user/.ssh/id_rsa
```

```go
// Use ssh:// URL scheme
req, _ := http.NewRequest("POST", "ssh://internal-service:8080/api/action", body)
resp, err := mgr.RoundTrip(req)
```

The request will automatically:
1. Connect to `jumphost.example.com` via SSH
2. Tunnel the HTTP request to `internal-service:8080`
3. Return the response

### OpenZiti Overlay Network

Use zero-trust networking with OpenZiti:

```bash
# Set Ziti identity
export WHEN_ZITI_IDENTITY_FILE=/etc/ziti/when-identity.json
```

```go
// Use ziti:// URL scheme
// The host is the Ziti service name, not a hostname
req, _ := http.NewRequest("POST", "ziti://workflowservice/v1/api/semantic/action", body)
resp, err := mgr.RoundTrip(req)
```

Benefits:
- Zero-trust authentication (no IP addresses, no ports)
- Automatic mutual TLS
- Identity-based access control
- Works across NATs and firewalls

## Architecture

### Transport Interface

All transports implement the `Transport` interface:

```go
type Transport interface {
    RoundTrip(*http.Request) (*http.Response, error)
    Close() error
}
```

### TransportManager

The manager routes requests to the appropriate transport:

```go
type Manager struct {
    transports map[TransportType]Transport
    configs    map[TransportType]*Config
}

func (m *Manager) RoundTrip(req *http.Request) (*http.Response, error) {
    transport, err := m.GetTransportForURL(req.URL.Scheme)
    return transport.RoundTrip(req)
}
```

### Integration with ActionExecutor

WHEN's `ActionExecutor` automatically uses the transport manager:

```go
// In executor.go
type ActionExecutor struct {
    transportMgr *transport.Manager
    registry     *executor.Registry
}

func (e *ActionExecutor) executeHTTP(ctx context.Context, action *semantic.SemanticScheduledAction, defaultMethod string) error {
    // ... create request ...
    resp, err := e.transportMgr.RoundTrip(req)
    // ... handle response ...
}
```

## SSH Transport Details

### Authentication Methods

The SSH transport supports multiple authentication methods:

1. **Public Key**: Recommended for production
   ```bash
   WHEN_SSH_KEY_FILE=/home/user/.ssh/id_rsa
   ```

2. **Encrypted Private Key**: With passphrase protection
   ```bash
   WHEN_SSH_KEY_FILE=/home/user/.ssh/id_rsa_encrypted
   WHEN_SSH_PASSWORD=passphrase
   ```

3. **Password Authentication**: For legacy systems
   ```bash
   WHEN_SSH_PASSWORD=sshpassword
   ```

### Host Key Verification

For security, use known_hosts file:

```bash
WHEN_SSH_KNOWN_HOSTS=/home/user/.ssh/known_hosts
```

Without this, host key verification is **disabled** (insecure).

### Connection Pooling

The SSH transport maintains a persistent SSH connection and creates tunnels on-demand for each HTTP request.

## OpenZiti Transport Details

### Identity Configuration

Ziti identities can be provided as:

1. **Identity File** (recommended):
   ```bash
   WHEN_ZITI_IDENTITY_FILE=/etc/ziti/when-identity.json
   ```

2. **Inline JSON**:
   ```bash
   WHEN_ZITI_IDENTITY_JSON='{"id":"when-client","ztAPI":"https://ctrl:1280",...}'
   ```

### Service Names

In Ziti URLs, the hostname is the **service name**, not a DNS hostname:

```go
// ✅ Correct: Service name
req, _ := http.NewRequest("POST", "ziti://workflowservice/v1/api/semantic/action", body)

// ❌ Wrong: DNS hostname
req, _ := http.NewRequest("POST", "ziti://workflow.example.com/v1/api/semantic/action", body)
```

### Ziti SDK Compatibility

- **SDK Version**: `github.com/openziti/sdk-golang v1.2.2`
- **Minimum Controller**: v1.6.0
- **Recommended Controller**: v1.6.5 - v1.6.7
- **Warning**: SDK v1.2.3+ requires controller v1.6.8+

See EVE README for detailed compatibility information.

## Error Handling

All transports return standard Go errors:

```go
resp, err := mgr.RoundTrip(req)
if err != nil {
    // Check error type
    switch {
    case strings.Contains(err.Error(), "SSH"):
        log.Printf("SSH tunnel error: %v", err)
    case strings.Contains(err.Error(), "Ziti"):
        log.Printf("Ziti network error: %v", err)
    default:
        log.Printf("HTTP error: %v", err)
    }
}
```

## Testing

### Unit Tests

Test individual transports:

```bash
go test ./transport/...
```

### Integration Tests

Test with real services:

```bash
# HTTP transport (no setup needed)
go test -v ./transport -run TestHTTPTransport

# SSH transport (requires SSH server)
export WHEN_SSH_HOST=localhost
export WHEN_SSH_USER=$USER
export WHEN_SSH_KEY_FILE=$HOME/.ssh/id_rsa
go test -v ./transport -run TestSSHTransport

# Ziti transport (requires Ziti network)
export WHEN_ZITI_IDENTITY_FILE=/path/to/identity.json
go test -v ./transport -run TestZitiTransport
```

## Performance

### Connection Pooling

All transports use connection pooling:

- **HTTP**: Standard `http.Transport` with configurable pool size
- **SSH**: Persistent SSH connection with on-demand tunnels
- **Ziti**: Ziti SDK manages connection pool internally

### Timeouts

Configure timeouts per transport:

```bash
WHEN_HTTP_TIMEOUT=30
WHEN_SSH_TIMEOUT=30
WHEN_ZITI_TIMEOUT=30
```

## Security Considerations

### HTTP/HTTPS
- Use HTTPS for production
- Validate TLS certificates
- Consider client certificate authentication

### SSH
- Always use public key authentication
- Enable host key verification with known_hosts
- Use strong SSH keys (RSA 4096, Ed25519)
- Consider SSH certificate authentication

### OpenZiti
- Identity-based access control (no IP addresses)
- Automatic mutual TLS
- Zero-trust architecture
- Centralized policy enforcement

## Troubleshooting

### SSH Connection Fails

```bash
# Test SSH connection manually
ssh -i $WHEN_SSH_KEY_FILE $WHEN_SSH_USER@$WHEN_SSH_HOST

# Enable debug logging
export WHEN_DEBUG=true
```

### Ziti Service Not Found

```bash
# List available Ziti services
ziti edge list services

# Check identity can access service
ziti edge list service-policies
```

### HTTP Timeout

```bash
# Increase timeout
export WHEN_HTTP_TIMEOUT=60
export WHEN_SSH_TIMEOUT=60
export WHEN_ZITI_TIMEOUT=60
```

## Future Enhancements

- [ ] HTTP/2 support for SSH and Ziti transports
- [ ] Automatic failover between transports
- [ ] Load balancing across multiple SSH jump hosts
- [ ] Metrics and observability (request counts, latencies)
- [ ] Circuit breaker pattern for failing transports
- [ ] WebSocket support over SSH and Ziti
- [ ] SOCKS5 proxy transport

## References

- [OpenZiti Documentation](https://openziti.io/docs)
- [OpenZiti SDK Go](https://github.com/openziti/sdk-golang)
- [golang.org/x/crypto/ssh](https://pkg.go.dev/golang.org/x/crypto/ssh)
- [net/http Transport](https://pkg.go.dev/net/http#Transport)
