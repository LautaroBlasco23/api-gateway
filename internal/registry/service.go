package registry

type Features struct {
	RateLimiter bool `json:"ratelimiter"`
	Injection   bool `json:"injection"`
	CORS        bool `json:"cors"`
	Cache       bool `json:"cache"`
}

type Service struct {
	Name     string   `json:"name"`
	URL      string   `json:"url"`
	Routes   []string `json:"routes"`
	Features Features `json:"features"`
}

type EndpointValidation struct {
	Route      string            `json:"route"`
	Method     string            `json:"method"`
	Validation map[string]string `json:"validation"`
}
