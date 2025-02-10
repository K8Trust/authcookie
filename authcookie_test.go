package authtokencookie_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	authtokencookie "github.com/K8Trust/authtokencookie"
)

// fakeAuthServer creates a test HTTP server simulating the auth server.
func fakeAuthServer(t *testing.T, status int, token string) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Set cookie before writing status
		cookie := &http.Cookie{
			Name:  "token",
			Value: token,
		}
		http.SetCookie(w, cookie)
		w.WriteHeader(status)
		resp := authtokencookie.AuthResponse{AccessToken: token}
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

	cfg := &authtokencookie.Config{
		AuthEndpoint: fakeServer.URL,
		Timeout:      authtokencookie.Timeout,
	}

	plugin, err := authtokencookie.New(context.Background(), nextHandler, cfg, "auth_cookie")
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

	cookies := res.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected a cookie to be set in the response")
	}

	found := false
	for _, cookie := range cookies {
		if cookie.Name == "token" && cookie.Value == "test-token" {
			found = true
			if !cookie.HttpOnly {
				t.Error("expected cookie to be HttpOnly")
			}
			if !cookie.Secure {
				t.Error("expected cookie to be Secure")
			}
		}
	}
	if !found {
		t.Errorf("expected cookie with token 'test-token' not found")
	}
}

func TestAuthPluginUnauthorized(t *testing.T) {
	// Next handler should not be called for unauthorized requests.
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for unauthorized requests")
	})

	cfg := authtokencookie.CreateConfig()

	plugin, err := authtokencookie.New(context.Background(), nextHandler, cfg, "auth_cookie")
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
