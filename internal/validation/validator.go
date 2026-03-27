package validation

import (
	"encoding/json"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"api-gateway/internal/registry"
)

var (
	reEmail    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	reUsername = regexp.MustCompile(`^[a-zA-Z0-9_\-]{3,32}$`)
	reUUID     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

	fileMIMEs = map[string]string{
		TypeFilePNG: "image/png",
		TypeFileJPG: "image/jpeg",
		TypeFilePDF: "application/pdf",
	}
)

// Validate checks the request body against the declared validation rules.
func Validate(r *http.Request, body []byte, rules map[string]registry.ValidationRule) error {
	if len(rules) == 0 {
		return nil
	}

	// Separate file and non-file rules.
	fileRules := map[string]registry.ValidationRule{}
	jsonRules := map[string]registry.ValidationRule{}
	for field, rule := range rules {
		if strings.HasPrefix(rule.Type, "file_") {
			fileRules[field] = rule
		} else {
			jsonRules[field] = rule
		}
	}

	if len(jsonRules) > 0 {
		if err := validateJSON(body, jsonRules); err != nil {
			return err
		}
	}

	if len(fileRules) > 0 {
		if err := validateFiles(r, fileRules); err != nil {
			return err
		}
	}

	return nil
}

func validateJSON(body []byte, rules map[string]registry.ValidationRule) error {
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("invalid JSON body")
	}

	for field, rule := range rules {
		val, ok := data[field]
		if !ok {
			if rule.IsRequired() {
				return fmt.Errorf("missing required field: %s", field)
			}
			continue // field is optional and absent -- skip
		}
		// A JSON null value is not a valid field value regardless of required flag.
		// Optional only means the field can be absent, not that it can be null.
		if val == nil {
			return fmt.Errorf("field %q must not be null", field)
		}
		if err := validateField(field, val, rule.Type); err != nil {
			return err
		}
		// Enum check: validate the value is in the allowed set.
		if rule.Type == TypeEnum {
			str, isStr := val.(string)
			if !isStr {
				return fmt.Errorf("field %q must be a string for enum validation", field)
			}
			if !contains(rule.Values, str) {
				return fmt.Errorf("field %q must be one of %v", field, rule.Values)
			}
		}
	}
	return nil
}

func validateField(field string, val any, typ string) error {
	str, isStr := val.(string)

	switch typ {
	case TypeEmail:
		if !isStr || !reEmail.MatchString(str) {
			return fmt.Errorf("field %q must be a valid email", field)
		}
	case TypePassword:
		if !isStr || len(str) < 8 {
			return fmt.Errorf("field %q must be a string with at least 8 characters", field)
		}
	case TypeUsername:
		if !isStr || !reUsername.MatchString(str) {
			return fmt.Errorf("field %q must be a valid username (3-32 alphanumeric chars)", field)
		}
	case TypeUUID:
		if !isStr || !reUUID.MatchString(str) {
			return fmt.Errorf("field %q must be a valid UUID", field)
		}
	case TypeInteger:
		switch v := val.(type) {
		case float64:
			if v != float64(int64(v)) {
				return fmt.Errorf("field %q must be an integer", field)
			}
		case string:
			if _, err := strconv.ParseInt(v, 10, 64); err != nil {
				return fmt.Errorf("field %q must be an integer", field)
			}
		default:
			return fmt.Errorf("field %q must be an integer", field)
		}
	case TypeString:
		if !isStr {
			return fmt.Errorf("field %q must be a string", field)
		}
	case TypeEnum:
		// Enum type validation (value membership) is handled in validateJSON
		// because it requires access to rule.Values. Nothing to do here.
	}
	return nil
}

func validateFiles(r *http.Request, rules map[string]registry.ValidationRule) error {
	ct := r.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return fmt.Errorf("expected multipart/form-data for file upload")
	}

	mr := multipart.NewReader(r.Body, params["boundary"])
	// We need to read parts to validate, but the body will be consumed.
	// For V1, we validate and signal the proxy should not try to re-read the body.
	// This is acceptable for V1 as file endpoint validation is a gateway responsibility.

	foundFields := map[string]bool{}
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		formName := part.FormName()
		if formName == "" {
			formName = part.FileName()
		}
		if rule, ok := rules[formName]; ok {
			expectedMIME := fileMIMEs[rule.Type]
			fileCT := part.Header.Get("Content-Type")
			if fileCT == "" {
				fileCT = "application/octet-stream"
			}
			if !strings.HasPrefix(fileCT, expectedMIME) {
				return fmt.Errorf("field %q must be %s, got %s", formName, expectedMIME, fileCT)
			}
			foundFields[formName] = true
		}
		part.Close()
	}

	for field, rule := range rules {
		if !foundFields[field] && rule.IsRequired() {
			return fmt.Errorf("missing required file field: %s", field)
		}
	}
	return nil
}

// contains reports whether val is present in slice (case-sensitive).
func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
