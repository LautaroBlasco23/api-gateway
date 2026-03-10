package gateway

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// proxyTo forwards the request to the target URL using httputil.ReverseProxy.
func proxyTo(target string, w http.ResponseWriter, r *http.Request) {
	targetURL, err := url.Parse(target)
	if err != nil {
		http.Error(w, "invalid backend URL", http.StatusInternalServerError)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ServeHTTP(w, r)
}
