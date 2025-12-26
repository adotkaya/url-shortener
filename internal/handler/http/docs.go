package http

import (
	"net/http"
	"path/filepath"
)

// ServeSwagger serves the Swagger UI documentation
func ServeSwagger(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join("web", "templates", "swagger.html"))
}

// ServeOpenAPISpec serves the OpenAPI JSON specification
func ServeOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	http.ServeFile(w, r, filepath.Join("api", "openapi.json"))
}
