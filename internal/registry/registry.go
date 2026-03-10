package registry

import (
	"strings"
	"sync"
)

type Registry struct {
	mu        sync.RWMutex
	services  []*Service
	endpoints []*EndpointValidation
}

func New() *Registry {
	return &Registry{}
}

func (r *Registry) Register(svc *Service) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, s := range r.services {
		if s.Name == svc.Name {
			r.services[i] = svc
			return
		}
	}
	r.services = append(r.services, svc)
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
			return
		}
	}
	r.endpoints = append(r.endpoints, ep)
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
