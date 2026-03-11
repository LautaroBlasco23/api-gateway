package injection

import (
	"net/http"
	"strings"
)

var dangerousPatterns = []string{
	"<script",
	"</script>",
	"drop table",
	"drop database",
	"union select",
	"or 1=1",
	"'; --",
	"\"--",
	"; rm ",
	"&& rm ",
	"| rm ",
	"eval(",
	"exec(",
	"system(",
	"../",
	"..\\",
}

// Filter inspects the request URL, query string, and headers for dangerous patterns.
// Returns (true, reason) if the request should be blocked.
func Filter(r *http.Request, body []byte) (bool, string) {
	targets := []string{
		r.URL.Path,
	}

	// Include raw query string as-is.
	if r.URL.RawQuery != "" {
		targets = append(targets, r.URL.RawQuery)
	}

	// Include decoded query parameter keys and values so URL-encoded payloads
	// like "%3Cscript%3E" are inspected in their decoded form.
	for key, values := range r.URL.Query() {
		if key != "" {
			targets = append(targets, key)
		}
		for _, v := range values {
			if v != "" {
				targets = append(targets, v)
			}
		}
	}

	for _, values := range r.Header {
		for _, v := range values {
			targets = append(targets, v)
		}
	}

	if isTextContentType(r.Header.Get("Content-Type")) && len(body) > 0 {
		targets = append(targets, string(body))
	}

	for _, t := range targets {
		lower := strings.ToLower(t)
		for _, p := range dangerousPatterns {
			if strings.Contains(lower, p) {
				return true, "dangerous pattern detected"
			}
		}
	}
	return false, ""
}

func isTextContentType(ct string) bool {
	ct = strings.ToLower(ct)
	return strings.Contains(ct, "application/json") ||
		strings.Contains(ct, "text/") ||
		strings.Contains(ct, "application/x-www-form-urlencoded")
}
