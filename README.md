# Auth Token Cookie Plugin for Traefik

A Traefik middleware plugin that authenticates requests using an internal authentication service and sets a token in the cookie header.

## Configuration

### Static Configuration (traefik.yml)

```yaml
experimental:
  plugins:
    authcookie:
      moduleName: "github.com/k8trust/authcookie"
      version: "v1.0.0"
```

### Dynamic Configuration

```yaml
# Dynamic configuration
http:
  middlewares:
    auth-token:
      plugin:
        authcookie:
          authEndpoint: "http://internal-auth.example.local/auth"
          timeout: "5s"
```

## Plugin Configuration Options

| Option       | Type     | Default                                   | Description                             |
|-------------|----------|-------------------------------------------|-----------------------------------------|
| authEndpoint | String   | "http://localhost:9000/test/auth/api-key" | Full URL of the authentication service  |
| timeout     | Duration | "5s"                                      | Timeout for authentication requests     |

## Required Headers

The plugin expects the following headers in incoming requests:

- `x-api-key`: API key for authentication
- `x-account`: Tenant identifier

## Development

### Project Structure
```
.
├── Dockerfile
├── Makefile
├── README.md
├── authcookie.go          # Main plugin implementation
├── authcookie_test.go     # Plugin tests
├── cmd
│   └── server
│       └── main.go        # Standalone server for testing
├── fake_auth_server.go    # Mock auth server for testing
└── go.mod
```

### Local Testing

1. Run the fake auth server:
```bash
go run fake_auth_server.go
```

2. Run the test server:
```bash
go run cmd/server/main.go
```

3. Test with curl:
```bash
# Test with valid headers
curl -v -H "x-api-key: test-key" -H "x-account: test-account" http://localhost:8080

# Test without headers (should get unauthorized)
curl -v http://localhost:8080
```

### Running Tests
```bash
go test -v ./...
```

## License

MIT License

## Contributing

Contributions are welcome! Please submit a pull request or open an issue for discussion.

### Contributors

- [Yousef Shamshoum](https://github.com/yousef-shamshoum)
- [Shay](https://github.com/shayktrust)

