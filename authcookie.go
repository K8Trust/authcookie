package authcookie

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
func New(_ context.Context, next http.Handler, cfg *Config, name string) (http.Handler, error) {
	if cfg.AuthEndpoint == "" {
		return nil, fmt.Errorf("missing auth endpoint")
	}

	logger := log.New(os.Stdout, "[AuthPlugin] ", log.LstdFlags)
	logger.Printf("=== Plugin Initialization ===")
	logger.Printf("Plugin Name: %s", name)
	logger.Printf("Auth Endpoint: %s", cfg.AuthEndpoint)
	logger.Printf("Timeout: %v", cfg.Timeout)
	logger.Printf("=== Initialization Complete ===\n")

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

// dumpRequest logs detailed request information
func dumpRequest(req *http.Request, logger *log.Logger, prefix string) {
	logger.Printf("%s === Request Dump ===", prefix)
	logger.Printf("%s Method: %s", prefix, req.Method)
	logger.Printf("%s URL: %s", prefix, req.URL.String())
	logger.Printf("%s Proto: %s", prefix, req.Proto)
	logger.Printf("%s Host: %s", prefix, req.Host)
	logger.Printf("%s RemoteAddr: %s", prefix, req.RemoteAddr)
	logger.Printf("%s Headers:", prefix)
	for k, v := range req.Header {
		logger.Printf("%s   %s: %v", prefix, k, v)
	}
	logger.Printf("%s Cookies:", prefix)
	for _, cookie := range req.Cookies() {
		logger.Printf("%s   %s: %s", prefix, cookie.Name, maskSensitive(cookie.Value))
	}
	logger.Printf("%s ==================", prefix)
}

// ServeHTTP processes incoming requests.
func (a *AuthPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	a.logger.Printf("=== Starting New Request Processing ===")
	dumpRequest(req, a.logger, "INITIAL")

	apiKey := req.Header.Get("x-api-key")
	tenant := req.Header.Get("x-account")

	a.logger.Printf("Extracted Headers - x-api-key: %s, x-account: %s",
		maskSensitive(apiKey),
		tenant)

	if apiKey == "" || tenant == "" {
		a.logger.Printf("ERROR: Missing required headers - apiKey present: %v, tenant present: %v",
			apiKey != "",
			tenant != "")
		http.Error(rw, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	a.logger.Printf("Creating auth request to: %s", a.authEndpoint)
	authReq, err := http.NewRequest(http.MethodGet, a.authEndpoint, nil)
	if err != nil {
		a.logger.Printf("ERROR: Failed to create auth request: %v", err)
		http.Error(rw, `{"error": "Internal error"}`, http.StatusInternalServerError)
		return
	}

	authReq.Header.Set("x-api-key", apiKey)
	authReq.Header.Set("x-account", tenant)
	a.logger.Printf("Auth request headers: %v", authReq.Header)

	a.logger.Printf("Sending auth request...")
	client := &http.Client{Timeout: Timeout}
	resp, err := client.Do(authReq)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			a.logger.Printf("ERROR: Auth request timed out after %v: %v", Timeout, err)
			http.Error(rw, `{"error": "Auth service timeout"}`, http.StatusGatewayTimeout)
			return
		}
		a.logger.Printf("ERROR: Auth request failed: %v", err)
		http.Error(rw, `{"error": "Internal error"}`, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		a.logger.Printf("ERROR: Failed to read auth response: %v", err)
		http.Error(rw, `{"error": "Internal error"}`, http.StatusInternalServerError)
		return
	}

	a.logger.Printf("Auth Response Status: %d", resp.StatusCode)
	a.logger.Printf("Auth Response Headers: %v", resp.Header)
	a.logger.Printf("Auth Response Body: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		a.logger.Printf("ERROR: Auth server returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
		rw.WriteHeader(resp.StatusCode)
		_, _ = rw.Write(body)
		return
	}

	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		a.logger.Printf("ERROR: Failed to parse auth response: %v", err)
		a.logger.Printf("Raw response body: %s", string(body))
		http.Error(rw, `{"error": "Internal error"}`, http.StatusInternalServerError)
		return
	}

	a.logger.Printf("Current request headers before modification: %v", req.Header)

	// Set cookie header in request (using proper capitalization)
	cookieHeader := fmt.Sprintf("token=%s;", authResp.AccessToken)
	req.Header.Set("Cookie", cookieHeader)
	a.logger.Printf("Added Cookie header to request: %s", maskSensitive(cookieHeader))

	// Set Authorization header as backup
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authResp.AccessToken))
	a.logger.Printf("Added Authorization header as backup")

	dumpRequest(req, a.logger, "MODIFIED")

	a.logger.Printf("Auth successful for account: %s, passing to next handler", tenant)
	a.next.ServeHTTP(rw, req)

	a.logger.Printf("=== Request Complete ===\n")
}