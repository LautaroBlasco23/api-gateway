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

func (r *Registry) Register(svc *Service) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, s := range r.services {
		if s.Name == svc.Name {
			r.services[i] = svc
			_ = r.save()
			return
		}
	}
	r.services = append(r.services, svc)
	_ = r.save()
}

// FindByRoute returns the service whose route prefix best matches path.
// Longest prefix wins.
func (r *Registry) FindByRoute(path string) *Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var best *Service
	bestLen := -1
	for _, svc := range r.services {
		for _, route := range svc.Routes {
			if strings.HasPrefix(path, route) && len(route) > bestLen {
				best = svc
				bestLen = len(route)
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
