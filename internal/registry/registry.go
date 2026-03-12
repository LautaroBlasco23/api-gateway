package registry

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
)

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

// FindByRoute returns the service with a route that exactly matches the path.
func (r *Registry) FindByRoute(path string) *Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, svc := range r.services {
		for _, route := range svc.Routes {
			if path == route {
				return svc
			}
		}
	}
	return nil
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

func (r *Registry) FindEndpoint(path, method string) *EndpointValidation {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, ep := range r.endpoints {
		if ep.Route == path && strings.EqualFold(ep.Method, method) {
			return ep
		}
	}
	return nil
}
