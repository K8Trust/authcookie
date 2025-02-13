package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/test/auth/api-key", func(w http.ResponseWriter, r *http.Request) {
		token := map[string]string{"accessToken": "test-token"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(token)
	})

	log.Println("Fake auth server running on :9000")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
