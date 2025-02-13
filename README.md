# Auth Token Cookie Plugin for Traefik

A Traefik middleware plugin that authenticates requests using an internal authentication service and sets a token in the cookie header.

## Features
- Kubernetes-native authentication service integration
- Secure token handling
- Configurable endpoint support
- Header-based authentication

## Configuration

### Static Configuration
```yaml
# traefik.yml
experimental:
  plugins:
    authcookie:
      moduleName: "github.com/k8trust/authcookie"
      version: "v1.0.6"
```

### Dynamic Configuration
```yaml
# Dynamic configuration
http:
  middlewares:
    auth-token:
      plugin:
        authcookie:
          authEndpoint: "http://k8s-service.namespace.svc.cluster.local/central/auth" 
```

## Configuration Options

| Option      | Type   | Default | Description |
|-------------|--------|---------|-------------|
| authEndpoint | String | `http://localhost:9000/test/auth/api-key` | Full URL of the Kubernetes authentication service |

## Authentication Headers

| Header | Description |
|--------|-------------|
| `x-api-key` | API key for authentication |
| `x-account` | Tenant identifier |

## Development

### Project Structure
```
.
├── Dockerfile             # Container build configuration
├── Makefile              # Build automation
├── README.md             # Documentation
├── authcookie.go         # Main plugin implementation
├── authcookie_test.go    # Plugin tests
├── cmd/
│   └── server/
│       └── main.go       # Standalone server for testing
├── fake_auth_server.go   # Mock auth server for testing
└── go.mod                # Go module definition
```

### Local Development

#### 1. Start the Mock Auth Server
```bash
go run fake_auth_server.go
```

#### 2. Run the Test Server
```bash
go run cmd/server/main.go
```

#### 3. Test the Endpoints
```bash
# Test with authentication headers
curl -v \
  -H "x-api-key: test-key" \
  -H "x-account: test-account" \
  http://localhost:8080

# Test unauthorized access
curl -v http://localhost:8080
```

### Running Tests
```bash
go test -v ./...
```

## Contributing

We welcome contributions! Please feel free to submit a Pull Request.

### Contributors
- [Yousef Shamshoum](https://github.com/yousef-shamshoum)
- [Shay](https://github.com/shayktrust)

## License

MIT License