package cache

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

const defaultTTL = 30 * time.Second

type entry struct {
	resp   *CachedResponse
	expiry time.Time
}

type Cache struct {
	mu    sync.RWMutex
	items map[string]*entry
}

func New() *Cache {
	c := &Cache{items: make(map[string]*entry)}
	go c.cleanup()
	return c
}

// Key generates a cache key from the request using the literal path.
func Key(r *http.Request) string {
	return r.Method + ":" + r.URL.Path + "?" + r.URL.RawQuery
}

// KeyWithRoute generates a cache key using a route pattern instead of the literal path.
// This ensures /api/users/123 and /api/users/456 share the same cache entry under /api/users/{id}.
func KeyWithRoute(r *http.Request, route string) string {
	return r.Method + ":" + route + "?" + r.URL.RawQuery
}

func (c *Cache) Get(key string) *CachedResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.items[key]
	if !ok || time.Now().After(e.expiry) {
		return nil
	}
	return e.resp
}

func (c *Cache) Set(key string, resp *CachedResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = &entry{resp: resp, expiry: time.Now().Add(defaultTTL)}
}

func (c *Cache) cleanup() {
	for range time.Tick(time.Minute) {
		now := time.Now()
		c.mu.Lock()
		for k, e := range c.items {
			if now.After(e.expiry) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}

// CachedResponse stores a captured HTTP response.
type CachedResponse struct {
	Status  int
	Headers http.Header
	Body    []byte
}

func (cr *CachedResponse) WriteTo(w http.ResponseWriter) {
	for k, vs := range cr.Headers {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(cr.Status)
	w.Write(cr.Body) //nolint:errcheck
}

// ResponseRecorder wraps http.ResponseWriter to capture the response while also writing it.
type ResponseRecorder struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func NewRecorder(w http.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{ResponseWriter: w, status: http.StatusOK}
}

func (rr *ResponseRecorder) WriteHeader(status int) {
	rr.status = status
	rr.ResponseWriter.WriteHeader(status)
}

func (rr *ResponseRecorder) Write(b []byte) (int, error) {
	rr.buf.Write(b)
	return rr.ResponseWriter.Write(b)
}

func (rr *ResponseRecorder) Result() *CachedResponse {
	return &CachedResponse{
		Status:  rr.status,
		Headers: rr.ResponseWriter.Header().Clone(),
		Body:    rr.buf.Bytes(),
	}
}
