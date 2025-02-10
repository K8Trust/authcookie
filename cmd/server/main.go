package main

import (
	"context"
	"log"
	"net/http"
	"os"

	authtokencookie "github.com/K8Trust/authtokencookie"
)

func main() {
	logger := log.New(os.Stdout, "[AuthPlugin] ", log.LstdFlags)

	// Define a simple next handler.
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	cfg := authtokencookie.CreateConfig()

	handler, err := authtokencookie.New(context.Background(), nextHandler, cfg, "auth-cookie")
	if err != nil {
		logger.Fatalf("Failed to create plugin: %v", err)
	}

	addr := ":8080"
	logger.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}
