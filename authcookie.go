package authtokencookie

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

// Config holds the plugin configuration.
type Config struct {
	AuthEndpoint string        `json:"authEndpoint,omitempty"`
	Timeout      time.Duration `json:"timeout,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		AuthEndpoint: "http://localhost:9000/test/auth/api-key",
		Timeout:      Timeout,
	}
}

// AuthResponse represents the authentication response.
type AuthResponse struct {
	AccessToken string `json:"accessToken"`
}

// AuthPlugin is the middleware that handles authentication.
type AuthPlugin struct {
	next         http.Handler
	authEndpoint string
	name         string
	logger       *log.Logger
}

// Timeout is the default timeout used for HTTP requests.
const Timeout = 5 * time.Second

// New creates a new instance of the AuthPlugin.
// It performs minimal initialization and returns an http.Handler.
func New(_ context.Context, next http.Handler, cfg *Config, name string) (http.Handler, error) {
	if cfg.AuthEndpoint == "" {
		return nil, fmt.Errorf("missing auth endpoint")
	}

	logger := log.New(os.Stdout, "[AuthPlugin] ", log.LstdFlags)
	logger.Printf("Initializing plugin with endpoint: %s, timeout: %v", cfg.AuthEndpoint, cfg.Timeout)

	return &AuthPlugin{
		next:         next,
		authEndpoint: cfg.AuthEndpoint,
		logger:       logger,
		name:         name,
	}, nil
}

// maskSensitive masks sensitive string data.
func maskSensitive(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + "****"
}

// sameSiteToString converts the SameSite constant to a human-readable string.
func sameSiteToString(s http.SameSite) string {
	switch s {
	case http.SameSiteLaxMode:
		return "Lax"
	case http.SameSiteStrictMode:
		return "Strict"
	case http.SameSiteNoneMode:
		return "None"
	default:
		return "DefaultMode"
	}
}

// ServeHTTP processes incoming requests.
// It checks for required headers, calls the auth endpoint, sets a cookie, and passes the request to the next handler.
func (a *AuthPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	apiKey := req.Header.Get("x-api-key")
	tenant := req.Header.Get("x-account")

	a.logger.Printf("Received request from %s with headers: x-api-key: %s, x-account: %s",
		req.RemoteAddr,
		maskSensitive(apiKey),
		tenant)

	if apiKey == "" || tenant == "" {
		a.logger.Printf("Missing required headers from %s", req.RemoteAddr)
		http.Error(rw, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	authReq, err := http.NewRequest(http.MethodGet, a.authEndpoint, nil)
	if err != nil {
		a.logger.Printf("Failed to create auth request: %v", err)
		http.Error(rw, `{"error": "Internal error"}`, http.StatusInternalServerError)
		return
	}
	authReq.Header.Set("x-api-key", apiKey)
	authReq.Header.Set("x-account", tenant)

	client := &http.Client{Timeout: Timeout}
	resp, err := client.Do(authReq)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			a.logger.Printf("Auth request timed out after %v: %v", Timeout, err)
			http.Error(rw, `{"error": "Auth service timeout"}`, http.StatusGatewayTimeout)
			return
		}
		a.logger.Printf("Auth request failed: %v", err)
		http.Error(rw, `{"error": "Internal error"}`, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		a.logger.Printf("Failed to read auth response: %v", err)
		http.Error(rw, `{"error": "Internal error"}`, http.StatusInternalServerError)
		return
	}

	a.logger.Printf("Auth server response status: %d for account: %s", resp.StatusCode, tenant)

	if resp.StatusCode != http.StatusOK {
		a.logger.Printf("Auth server returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
		rw.WriteHeader(resp.StatusCode)
		_, _ = rw.Write(body)
		return
	}

	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		a.logger.Printf("Failed to parse auth response: %v", err)
		http.Error(rw, `{"error": "Internal error"}`, http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{
		Name:     "token",
		Value:    authResp.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(rw, cookie)

	a.logger.Printf("Setting cookie: %s=%s, HttpOnly: %v, Secure: %v, SameSite: %s",
		cookie.Name,
		maskSensitive(cookie.Value),
		cookie.HttpOnly,
		cookie.Secure,
		sameSiteToString(cookie.SameSite))

	a.logger.Printf("Auth successful for account: %s, passing request to next handler", tenant)
	a.next.ServeHTTP(rw, req)
}
