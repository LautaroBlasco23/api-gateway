package validation

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"api-gateway/internal/registry"
)

// boolPtr is a helper to obtain a pointer to a bool literal.
func boolPtr(b bool) *bool { return &b }

// makeRequest builds a minimal http.Request with the given JSON body.
func makeRequest(body []byte) *http.Request {
	r, _ := http.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestRequiredFieldPresent(t *testing.T) {
	rules := map[string]registry.ValidationRule{
		"email": {Type: TypeEmail},
	}
	body := mustMarshal(map[string]any{"email": "user@example.com"})
	if err := Validate(makeRequest(body), body, rules); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestRequiredFieldMissing(t *testing.T) {
	rules := map[string]registry.ValidationRule{
		"email": {Type: TypeEmail},
	}
	body := mustMarshal(map[string]any{})
	err := Validate(makeRequest(body), body, rules)
	if err == nil {
		t.Fatal("expected error for missing required field")
	}
}

func TestOptionalFieldMissing(t *testing.T) {
	rules := map[string]registry.ValidationRule{
		"email": {Type: TypeEmail, Required: boolPtr(false)},
	}
	body := mustMarshal(map[string]any{})
	if err := Validate(makeRequest(body), body, rules); err != nil {
		t.Errorf("expected no error for optional missing field, got: %v", err)
	}
}

func TestOptionalFieldPresentButInvalid(t *testing.T) {
	rules := map[string]registry.ValidationRule{
		"email": {Type: TypeEmail, Required: boolPtr(false)},
	}
	body := mustMarshal(map[string]any{"email": "not-an-email"})
	err := Validate(makeRequest(body), body, rules)
	if err == nil {
		t.Fatal("expected validation error for present but invalid optional field")
	}
}

func TestNullValueFailsValidation(t *testing.T) {
	rules := map[string]registry.ValidationRule{
		"email": {Type: TypeEmail, Required: boolPtr(false)},
	}
	// JSON null for an optional field must still fail.
	body := []byte(`{"email": null}`)
	err := Validate(makeRequest(body), body, rules)
	if err == nil {
		t.Fatal("expected error for null value on an optional field")
	}
}

func TestEnumFieldValidValue(t *testing.T) {
	rules := map[string]registry.ValidationRule{
		"status": {Type: TypeEnum, Values: []string{"active", "inactive", "pending"}},
	}
	body := mustMarshal(map[string]any{"status": "active"})
	if err := Validate(makeRequest(body), body, rules); err != nil {
		t.Errorf("expected no error for valid enum value, got: %v", err)
	}
}

func TestEnumFieldInvalidValue(t *testing.T) {
	rules := map[string]registry.ValidationRule{
		"status": {Type: TypeEnum, Values: []string{"active", "inactive"}},
	}
	body := mustMarshal(map[string]any{"status": "unknown"})
	err := Validate(makeRequest(body), body, rules)
	if err == nil {
		t.Fatal("expected error for invalid enum value")
	}
}

func TestEnumCaseSensitive(t *testing.T) {
	rules := map[string]registry.ValidationRule{
		"status": {Type: TypeEnum, Values: []string{"active"}},
	}
	body := mustMarshal(map[string]any{"status": "Active"})
	err := Validate(makeRequest(body), body, rules)
	if err == nil {
		t.Fatal("expected error: enum comparison must be case-sensitive")
	}
}

func TestEnumFieldNonStringValue(t *testing.T) {
	rules := map[string]registry.ValidationRule{
		"status": {Type: TypeEnum, Values: []string{"1", "2"}},
	}
	body := mustMarshal(map[string]any{"status": 1})
	err := Validate(makeRequest(body), body, rules)
	if err == nil {
		t.Fatal("expected error for non-string value on enum field")
	}
}

func TestEnumFieldMissingAndRequired(t *testing.T) {
	rules := map[string]registry.ValidationRule{
		"status": {Type: TypeEnum, Values: []string{"active"}},
	}
	body := mustMarshal(map[string]any{})
	err := Validate(makeRequest(body), body, rules)
	if err == nil {
		t.Fatal("expected error for missing required enum field")
	}
}

func TestEnumFieldMissingAndOptional(t *testing.T) {
	rules := map[string]registry.ValidationRule{
		"status": {Type: TypeEnum, Values: []string{"active"}, Required: boolPtr(false)},
	}
	body := mustMarshal(map[string]any{})
	if err := Validate(makeRequest(body), body, rules); err != nil {
		t.Errorf("expected no error for missing optional enum field, got: %v", err)
	}
}

func TestBackwardCompatOldFormatDeserialization(t *testing.T) {
	raw := []byte(`{"route":"/x","method":"POST","validation":{"email":"email","username":"username"}}`)
	var ev registry.EndpointValidation
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if ev.Route != "/x" {
		t.Errorf("expected route /x, got %s", ev.Route)
	}
	if ev.Method != "POST" {
		t.Errorf("expected method POST, got %s", ev.Method)
	}
	emailRule, ok := ev.Validation["email"]
	if !ok {
		t.Fatal("expected email rule to be present")
	}
	if emailRule.Type != "email" {
		t.Errorf("expected email rule type to be 'email', got %q", emailRule.Type)
	}
	// Rules migrated from old format should default to required.
	if !emailRule.IsRequired() {
		t.Error("expected migrated rule to be required by default")
	}
}

func TestNewFormatDeserialization(t *testing.T) {
	raw := []byte(`{"route":"/x","method":"POST","validation":{"email":{"type":"email","required":false}}}`)
	var ev registry.EndpointValidation
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	emailRule := ev.Validation["email"]
	if emailRule.Type != "email" {
		t.Errorf("expected type 'email', got %q", emailRule.Type)
	}
	if emailRule.IsRequired() {
		t.Error("expected field to be optional (required=false)")
	}
}

// mustMarshal encodes v to JSON and panics on error.
func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
