package registry

import (
	"testing"
)

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern    string
		path       string
		wantMatch  bool
		wantParams map[string]string
	}{
		// Exact matches
		{"/echo", "/echo", true, map[string]string{}},
		{"/api/users", "/api/users", true, map[string]string{}},

		// Single param
		{"/api/users/{id}", "/api/users/123", true, map[string]string{"id": "123"}},
		{"/api/users/{id}", "/api/users/abc", true, map[string]string{"id": "abc"}},

		// Multiple params
		{"/api/users/{uid}/posts/{pid}", "/api/users/42/posts/7", true, map[string]string{"uid": "42", "pid": "7"}},

		// No match: different length
		{"/api/users/{id}", "/api/users", false, nil},
		{"/api/users/{id}", "/api/users/123/extra", false, nil},

		// No match: literal segment mismatch
		{"/api/users/{id}", "/api/orders/123", false, nil},
		{"/api/users/admin", "/api/users/123", false, nil},

		// Exact beats param (literal segment must match exactly)
		{"/api/users/admin", "/api/users/admin", true, map[string]string{}},

		// Empty param segment not allowed
		{"/api/users/{id}", "/api/users/", false, nil},
	}

	for _, tt := range tests {
		params, ok := matchPattern(tt.pattern, tt.path)
		if ok != tt.wantMatch {
			t.Errorf("matchPattern(%q, %q): got match=%v, want %v", tt.pattern, tt.path, ok, tt.wantMatch)
			continue
		}
		if !tt.wantMatch {
			continue
		}
		for k, v := range tt.wantParams {
			if params[k] != v {
				t.Errorf("matchPattern(%q, %q): param %q = %q, want %q", tt.pattern, tt.path, k, params[k], v)
			}
		}
		if len(params) != len(tt.wantParams) {
			t.Errorf("matchPattern(%q, %q): got %d params, want %d", tt.pattern, tt.path, len(params), len(tt.wantParams))
		}
	}
}

func TestSpecificity(t *testing.T) {
	tests := []struct {
		pattern string
		want    int
	}{
		{"/echo", 1},
		{"/api/users", 2},
		{"/api/users/{id}", 2},
		{"/api/users/{uid}/posts/{pid}", 3},
		{"/api/users/admin", 3},
	}
	for _, tt := range tests {
		if got := specificity(tt.pattern); got != tt.want {
			t.Errorf("specificity(%q) = %d, want %d", tt.pattern, got, tt.want)
		}
	}
}

func TestFindByRoute(t *testing.T) {
	reg := &Registry{}
	reg.services = []*Service{
		{Name: "echo", URL: "http://echo:8080", Routes: []string{"/echo"}},
		{Name: "users", URL: "http://users:8080", Routes: []string{"/api/users", "/api/users/{id}"}},
		{Name: "admin", URL: "http://admin:8080", Routes: []string{"/api/users/admin"}},
	}

	tests := []struct {
		path        string
		wantService string
		wantRoute   string
		wantParams  map[string]string
	}{
		{"/echo", "echo", "/echo", map[string]string{}},
		{"/api/users", "users", "/api/users", map[string]string{}},
		{"/api/users/123", "users", "/api/users/{id}", map[string]string{"id": "123"}},
		// Exact /api/users/admin beats /api/users/{id}
		{"/api/users/admin", "admin", "/api/users/admin", map[string]string{}},
	}

	for _, tt := range tests {
		m := reg.FindByRoute(tt.path)
		if m == nil {
			t.Errorf("FindByRoute(%q): got nil, want service %q", tt.path, tt.wantService)
			continue
		}
		if m.Service.Name != tt.wantService {
			t.Errorf("FindByRoute(%q): service = %q, want %q", tt.path, m.Service.Name, tt.wantService)
		}
		if m.Route != tt.wantRoute {
			t.Errorf("FindByRoute(%q): route = %q, want %q", tt.path, m.Route, tt.wantRoute)
		}
		for k, v := range tt.wantParams {
			if m.Params[k] != v {
				t.Errorf("FindByRoute(%q): param %q = %q, want %q", tt.path, k, m.Params[k], v)
			}
		}
	}

	// No match
	if m := reg.FindByRoute("/not/registered"); m != nil {
		t.Errorf("FindByRoute(%q): expected nil, got %v", "/not/registered", m)
	}
}

func TestFindEndpoint(t *testing.T) {
	reg := &Registry{}
	reg.endpoints = []*EndpointValidation{
		{Route: "/api/users", Method: "POST", Validation: map[string]ValidationRule{
			"email": {Type: "email"},
		}},
		{Route: "/api/users/{id}", Method: "GET", Validation: map[string]ValidationRule{
			"id": {Type: "string"},
		}},
	}

	ep, params := reg.FindEndpoint("/api/users", "POST")
	if ep == nil {
		t.Fatal("expected endpoint for POST /api/users")
	}
	if ep.Validation["email"].Type != "email" {
		t.Errorf("unexpected validation: %v", ep.Validation)
	}
	if len(params) != 0 {
		t.Errorf("expected no params, got %v", params)
	}

	ep, params = reg.FindEndpoint("/api/users/42", "GET")
	if ep == nil {
		t.Fatal("expected endpoint for GET /api/users/{id}")
	}
	if params["id"] != "42" {
		t.Errorf("expected param id=42, got %v", params)
	}

	ep, _ = reg.FindEndpoint("/api/users/42", "POST")
	if ep != nil {
		t.Errorf("expected no endpoint for POST /api/users/42, got %v", ep)
	}
}
