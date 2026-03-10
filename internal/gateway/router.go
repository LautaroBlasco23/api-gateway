package gateway

import (
	"net/http"

	"api-gateway/internal/registry"
)

// NewRouter sets up the HTTP routes for the gateway.
func NewRouter(reg *registry.Registry) http.Handler {
	h := newHandler(reg)
	mux := http.NewServeMux()

	// Gateway management endpoints
	mux.HandleFunc("POST /register", h.register)
	mux.HandleFunc("POST /register/endpoint", h.registerEndpoint)

	// Catch-all: proxy to registered services
	mux.HandleFunc("/", h.proxy)

	return mux
}
