package registry

import (
	"encoding/json"
	"fmt"
)

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

// ValidationRule describes how a single request field should be validated.
// Required is a pointer so that omitting it in JSON results in nil, which
// IsRequired() treats as true (required by default).
type ValidationRule struct {
	Type     string   `json:"type"`
	Required *bool    `json:"required,omitempty"`
	Values   []string `json:"values,omitempty"`
}

// IsRequired returns whether the field must be present in the request body.
// Defaults to true when Required is not explicitly set.
func (v ValidationRule) IsRequired() bool {
	if v.Required == nil {
		return true
	}
	return *v.Required
}

type EndpointValidation struct {
	Route      string                    `json:"route"`
	Method     string                    `json:"method"`
	Validation map[string]ValidationRule `json:"validation"`
}

// UnmarshalJSON handles both the new map[string]ValidationRule format and the
// old map[string]string format so existing registry.json files and existing
// clients continue to work after the schema change.
func (ev *EndpointValidation) UnmarshalJSON(data []byte) error {
	// Capture the validation field as raw JSON so we can detect its format.
	var raw struct {
		Route         string          `json:"route"`
		Method        string          `json:"method"`
		RawValidation json.RawMessage `json:"validation"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	ev.Route = raw.Route
	ev.Method = raw.Method

	if len(raw.RawValidation) == 0 {
		return nil
	}

	// Try new format: map[string]ValidationRule
	var newFmt map[string]ValidationRule
	if err := json.Unmarshal(raw.RawValidation, &newFmt); err == nil {
		ev.Validation = newFmt
		return nil
	}

	// Fall back to old format: map[string]string
	var oldFmt map[string]string
	if err := json.Unmarshal(raw.RawValidation, &oldFmt); err != nil {
		return fmt.Errorf("validation field has unrecognized format")
	}
	ev.Validation = make(map[string]ValidationRule, len(oldFmt))
	for field, typ := range oldFmt {
		ev.Validation[field] = ValidationRule{Type: typ}
	}
	return nil
}
