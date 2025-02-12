package main

import (
	"context"
	"log"
	"net/http"
	"os"

	authcookie "github.com/K8Trust/authcookie"
)

func main() {
	logger := log.New(os.Stdout, "[AuthPlugin Main] ", log.LstdFlags)

	// Define a next handler with debug logging
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("=== Next Handler ===")
		logger.Printf("Received headers: %v", r.Header)
		logger.Printf("Received cookies: %v", r.Cookies())
		logger.Printf("Method: %s, Path: %s", r.Method, r.URL.Path)
		
		w.Write([]byte("OK"))
		
		logger.Printf("=== Next Handler Complete ===")
	})

	cfg := authcookie.CreateConfig()
	
	// Log configuration
	logger.Printf("=== Starting Server ===")
	logger.Printf("Auth Endpoint: %s", cfg.AuthEndpoint)
	logger.Printf("Timeout: %v", cfg.Timeout)

	handler, err := authcookie.New(context.Background(), nextHandler, cfg, "auth-cookie")
	if err != nil {
		logger.Fatalf("Failed to create plugin: %v", err)
	}

	addr := ":8080"
	logger.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}