package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Dependency Injection
	store := NewURLStore()

	// Route Registration
	http.HandleFunc("/", loggingMiddleware(store.RedirectHandler))
	http.HandleFunc("/create", loggingMiddleware(store.CreateHandler))

	fmt.Println("Server starting on :8080...")
	http.ListenAndServe(":8080", nil)
}
