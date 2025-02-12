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
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	// Next handler that simply writes OK.
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	res := rec.Result()
	defer res.Body.Close()

	cookieHeader := res.Header.Get("cookie")
	expectedHeader := "token=test-token;"
	if cookieHeader != expectedHeader {
		t.Errorf("expected cookie header to be %q, got %q", expectedHeader, cookieHeader)
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