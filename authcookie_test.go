package authcookie_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	authcookie "github.com/K8Trust/authcookie"
)

// fakeAuthServer creates a test HTTP server simulating the auth server.
func fakeAuthServer(t *testing.T, status int, token string) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log incoming request headers for debugging
		t.Logf("Auth server received headers: %v", r.Header)
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		resp := authcookie.AuthResponse{AccessToken: token}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("could not encode response: %v", err)
		}
	})
	return httptest.NewServer(handler)
}

func TestAuthPluginSuccess(t *testing.T) {
	fakeServer := fakeAuthServer(t, http.StatusOK, "test-token")
	defer fakeServer.Close()

	var capturedRequest *http.Request
	
	// Next handler that captures the request for verification
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r.Clone(r.Context()) // Clone the request to inspect it
		w.Write([]byte("OK"))
	})

	cfg := &authcookie.Config{
		AuthEndpoint: fakeServer.URL,
		Timeout:      authcookie.Timeout,
	}

	plugin, err := authcookie.New(context.Background(), nextHandler, cfg, "auth_cookie")
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("x-api-key", "dummy")
	req.Header.Set("x-account", "dummy")

	rec := httptest.NewRecorder()
	plugin.ServeHTTP(rec, req)
	
	if capturedRequest == nil {
		t.Fatal("next handler was not called")
	}

	// Check Cookie header
	cookieHeader := capturedRequest.Header.Get("Cookie")
	expectedCookie := "token=test-token;"
	if cookieHeader != expectedCookie {
		t.Errorf("expected Cookie header to be %q, got %q", expectedCookie, cookieHeader)
	}

	// Check Authorization header
	authHeader := capturedRequest.Header.Get("Authorization")
	expectedAuth := "Bearer test-token"
	if authHeader != expectedAuth {
		t.Errorf("expected Authorization header to be %q, got %q", expectedAuth, authHeader)
	}

	// Verify response status
	if rec.Code != http.StatusOK {
		t.Errorf("expected status OK, got %v", rec.Code)
	}
}

func TestAuthPluginUnauthorized(t *testing.T) {
	// Next handler should not be called for unauthorized requests.
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for unauthorized requests")
	})

	cfg := authcookie.CreateConfig()

	plugin, err := authcookie.New(context.Background(), nextHandler, cfg, "auth_cookie")
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()
	plugin.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d for unauthorized request, got %d", http.StatusUnauthorized, rec.Result().StatusCode)
	}
}