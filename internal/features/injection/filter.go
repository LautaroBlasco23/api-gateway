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
		r.URL.RawQuery,
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
