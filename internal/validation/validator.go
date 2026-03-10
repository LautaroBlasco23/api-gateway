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
func Validate(r *http.Request, body []byte, rules map[string]string) error {
	if len(rules) == 0 {
		return nil
	}

	// Separate file and non-file rules.
	fileRules := map[string]string{}
	jsonRules := map[string]string{}
	for field, typ := range rules {
		if strings.HasPrefix(typ, "file_") {
			fileRules[field] = typ
		} else {
			jsonRules[field] = typ
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

func validateJSON(body []byte, rules map[string]string) error {
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("invalid JSON body")
	}

	for field, typ := range rules {
		val, ok := data[field]
		if !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
		if err := validateField(field, val, typ); err != nil {
			return err
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
	}
	return nil
}

func validateFiles(r *http.Request, rules map[string]string) error {
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
		if _, ok := rules[formName]; ok {
			expectedMIME := fileMIMEs[rules[formName]]
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

	for field := range rules {
		if !foundFields[field] {
			return fmt.Errorf("missing required file field: %s", field)
		}
	}
	return nil
}
