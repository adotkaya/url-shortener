package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *URLStore) CreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req CreateUrlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL parameter is missing", http.StatusBadRequest)
		return
	}

	id := generateShortID()
	s.Put(id, req.URL)

	resp := CreateUrlResponse{
		ID:       id,
		ShortURL: fmt.Sprintf("http://localhost:8080/%s", id),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *URLStore) RedirectHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[1:]

	targetURL, exists := s.Get(id)
	if !exists {
		http.Error(w, "Short URL not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, targetURL, http.StatusFound)
}
