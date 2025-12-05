package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

type Url struct {
	ID          string `json:"id"`
	OriginalUrl string `json:"original_url"`
}

var urlStore = make(map[string]string)

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

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[1:]

		targetURL, exists := urlStore[id]
		if !exists {
			http.Error(w, "URL not found", http.StatusNotFound)
			return
		}

		http.Redirect(w, r, targetURL, http.StatusFound)
	})

	fmt.Println("Listening on port 8080")
	http.HandleFunc("/create", createUrl)
	http.ListenAndServe(":8080", nil)
}
