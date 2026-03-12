package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"api-gateway/internal/features/cache"
	"api-gateway/internal/features/cors"
	"api-gateway/internal/features/injection"
	"api-gateway/internal/features/ratelimiter"
	"api-gateway/internal/registry"
	"api-gateway/internal/validation"
)

type handler struct {
	reg         *registry.Registry
	rateLimiter *ratelimiter.Limiter
	cache       *cache.Cache
}

func newHandler(reg *registry.Registry) *handler {
	return &handler{
		reg:         reg,
		rateLimiter: ratelimiter.New(),
		cache:       cache.New(),
	}
}

func (h *handler) register(w http.ResponseWriter, r *http.Request) {
	var svc registry.Service
	if err := json.NewDecoder(r.Body).Decode(&svc); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if svc.Name == "" || svc.URL == "" || len(svc.Routes) == 0 {
		http.Error(w, "name, url and routes are required", http.StatusBadRequest)
		return
	}
	if !h.reg.Register(&svc) {
		http.Error(w, `{"error":"service already registered"}`, http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "registered", "service": svc.Name}) //nolint:errcheck
}

func (h *handler) registerEndpoint(w http.ResponseWriter, r *http.Request) {
	var ep registry.EndpointValidation
	if err := json.NewDecoder(r.Body).Decode(&ep); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if ep.Route == "" || ep.Method == "" {
		http.Error(w, "route and method are required", http.StatusBadRequest)
		return
	}
	h.reg.RegisterEndpoint(&ep)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "registered", "route": ep.Route}) //nolint:errcheck
}

func (h *handler) proxy(w http.ResponseWriter, r *http.Request) {
	svc := h.reg.FindByRoute(r.URL.Path)
	if svc == nil {
		http.Error(w, "no service registered for this route", http.StatusBadRequest)
		return
	}

	// Read body once so multiple features can inspect it.
	var bodyBytes []byte
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusInternalServerError)
			return
		}
		r.Body.Close()
	}
	// Restore body for downstream use.
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// CORS
	if svc.Features.CORS {
		if cors.Handle(w, r) {
			return // preflight handled
		}
	}

	// Rate limiting
	if svc.Features.RateLimiter {
		if !h.rateLimiter.Allow(svc.Name, r.RemoteAddr) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
	}

	// Injection filtering
	if svc.Features.Injection {
		if blocked, reason := injection.Filter(r, bodyBytes); blocked {
			http.Error(w, "request blocked: "+reason, http.StatusBadRequest)
			return
		}
	}

	// Endpoint-level input validation
	ep := h.reg.FindEndpoint(r.URL.Path, r.Method)
	if ep != nil {
		if err := validation.Validate(r, bodyBytes, ep.Validation); err != nil {
			http.Error(w, "validation error: "+err.Error(), http.StatusBadRequest)
			return
		}
		// Restore body after validation may have consumed it (e.g. multipart parsing).
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	// Cache (GET and HEAD only)
	if svc.Features.Cache && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		key := cache.Key(r)
		if cached := h.cache.Get(key); cached != nil {
			cached.WriteTo(w)
			return
		}
		rec := cache.NewRecorder(w)
		proxyTo(svc.URL, rec, r)
		h.cache.Set(key, rec.Result())
		return
	}

	proxyTo(svc.URL, w, r)
}
