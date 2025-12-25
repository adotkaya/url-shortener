package http

import (
	"encoding/json"
	"net/http"
)

// Response helpers for consistent API responses

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// SuccessResponse represents a successful response
type SuccessResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// If encoding fails, log it but don't try to send another response
		// (headers are already sent)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// respondError sends an error response
func respondError(w http.ResponseWriter, statusCode int, message string) {
	respondJSON(w, statusCode, ErrorResponse{
		Error: message,
	})
}

// respondSuccess sends a success response
func respondSuccess(w http.ResponseWriter, statusCode int, data interface{}, message string) {
	respondJSON(w, statusCode, SuccessResponse{
		Data:    data,
		Message: message,
	})
}
