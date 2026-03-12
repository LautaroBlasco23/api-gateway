package registry

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
)

// RouteMatch holds the result of a successful route lookup.
type RouteMatch struct {
	Service *Service
	Route   string            // matched pattern (e.g. /api/users/{id})
	Params  map[string]string // extracted path parameters
}

// matchPattern returns extracted path params and true when path matches pattern.
// Pattern segments like {id} match any non-empty path segment.
func matchPattern(pattern, path string) (map[string]string, bool) {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(patternParts) != len(pathParts) {
		return nil, false
	}
	params := map[string]string{}
	for i, seg := range patternParts {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			if pathParts[i] == "" {
				return nil, false
			}
			params[seg[1:len(seg)-1]] = pathParts[i]
		} else if seg != pathParts[i] {
			return nil, false
		}
	}
	return params, true
}

// specificity returns the number of literal (non-parameterized) segments.
// Higher specificity patterns are preferred when multiple patterns match.
func specificity(pattern string) int {
	count := 0
	for _, seg := range strings.Split(strings.Trim(pattern, "/"), "/") {
		if !strings.HasPrefix(seg, "{") {
			count++
		}
	}
	return count
}

type Registry struct {
	mu        sync.RWMutex
	services  []*Service
	endpoints []*EndpointValidation
	filePath  string
}

type registrySnapshot struct {
	Services  []*Service            `json:"services"`
	Endpoints []*EndpointValidation `json:"endpoints"`
}

func New(filePath string) *Registry {
	r := &Registry{filePath: filePath}
	_ = r.load()
	return r
}

func (r *Registry) load() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var snap registrySnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}
	r.services = snap.Services
	r.endpoints = snap.Endpoints
	return nil
}

func (r *Registry) save() error {
	snap := registrySnapshot{
		Services:  r.services,
		Endpoints: r.endpoints,
	}
	data, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	tmp := r.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, r.filePath)
}

// Register adds a new service to the registry.
// Returns false if a service with the same name or URL already exists.
func (r *Registry) Register(svc *Service) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.services {
		if s.Name == svc.Name || s.URL == svc.URL {
			return false
		}
	}
	r.services = append(r.services, svc)
	_ = r.save()
	return true
}

// FindByRoute returns the best matching service for the given path using pattern matching.
// More specific patterns (more literal segments) win over parameterized ones.
func (r *Registry) FindByRoute(path string) *RouteMatch {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var best *RouteMatch
	bestSpec := -1
	for _, svc := range r.services {
		for _, route := range svc.Routes {
			params, ok := matchPattern(route, path)
			if !ok {
				continue
			}
			if s := specificity(route); s > bestSpec {
				bestSpec = s
				best = &RouteMatch{Service: svc, Route: route, Params: params}
			}
		}
	}
	return best
}

func (r *Registry) RegisterEndpoint(ep *EndpointValidation) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, e := range r.endpoints {
		if e.Route == ep.Route && strings.EqualFold(e.Method, ep.Method) {
			r.endpoints[i] = ep
			_ = r.save()
			return
		}
	}
	r.endpoints = append(r.endpoints, ep)
	_ = r.save()
}

// FindEndpoint returns the best matching endpoint validation rule for the given path and method.
// Returns the matched endpoint and extracted path parameters.
func (r *Registry) FindEndpoint(path, method string) (*EndpointValidation, map[string]string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var bestEp *EndpointValidation
	var bestParams map[string]string
	bestSpec := -1
	for _, ep := range r.endpoints {
		if !strings.EqualFold(ep.Method, method) {
			continue
		}
		params, ok := matchPattern(ep.Route, path)
		if !ok {
			continue
		}
		if s := specificity(ep.Route); s > bestSpec {
			bestSpec = s
			bestEp = ep
			bestParams = params
		}
	}
	return bestEp, bestParams
}
