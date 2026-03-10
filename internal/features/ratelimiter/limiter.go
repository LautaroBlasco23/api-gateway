package ratelimiter

import (
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	requestsPerSecond = 10
	burst             = 20
)

type Limiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

func New() *Limiter {
	l := &Limiter{limiters: make(map[string]*rate.Limiter)}
	go l.cleanup()
	return l
}

// Allow returns true if the request from remoteAddr to service is within rate limits.
func (l *Limiter) Allow(service, remoteAddr string) bool {
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		ip = remoteAddr
	}
	key := service + ":" + ip

	l.mu.Lock()
	lim, ok := l.limiters[key]
	if !ok {
		lim = rate.NewLimiter(requestsPerSecond, burst)
		l.limiters[key] = lim
	}
	l.mu.Unlock()

	return lim.Allow()
}

// cleanup resets all limiters every 5 minutes to prevent unbounded growth.
func (l *Limiter) cleanup() {
	for range time.Tick(5 * time.Minute) {
		l.mu.Lock()
		l.limiters = make(map[string]*rate.Limiter)
		l.mu.Unlock()
	}
}
