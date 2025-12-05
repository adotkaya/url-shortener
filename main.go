package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

type Url struct {
	ID          string `json:"id"`
	OriginalUrl string `json:"original_url"`
}

var urlStore = make(map[string]string)
var mu sync.Mutex

func createUrl(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	originalURL := r.FormValue("url")
	if originalURL == "" {
		http.Error(w, "URL parameter is missing", http.StatusBadRequest)
		return
	}

	id := generateShortID()
	mu.Lock()
	defer mu.Unlock()
	urlStore[id] = originalURL

	fmt.Fprintf(w, "Shortened URL: http://localhost:8080/%s", id)
}

func generateShortID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 6

	rand.Seed(time.Now().UnixNano())

	shortKey := make([]byte, length)

	for i := range shortKey {
		shortKey[i] = charset[rand.Intn(len(charset))]
	}

	return string(shortKey)
}

func redirectURL(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[1:]
	mu.Lock()
	targetURL, exists := urlStore[id]
	mu.Unlock()

	if !exists {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, targetURL, http.StatusFound)
}

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next(w, r)

		duration := time.Since(start)

		slog.Info("Request processed",
			"method", r.Method,
			"url", r.URL.String(),
			"duration", duration,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	http.HandleFunc("/create", loggingMiddleware(createUrl))
	http.HandleFunc("/", loggingMiddleware(redirectURL))

	fmt.Println("Listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
