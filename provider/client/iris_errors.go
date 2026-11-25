package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type IrisHTTPError struct {
	StatusCode int
	Code       string
	Fields     []string
	Detail     string
	RetryAfter time.Duration
}

func (e *IrisHTTPError) Error() string {
	message := fmt.Sprintf("Iris API returned HTTP %d (%s)", e.StatusCode, http.StatusText(e.StatusCode))
	if e.Code != "" {
		message += " [" + e.Code + "]"
	}
	if len(e.Fields) > 0 {
		message += ": " + strings.Join(e.Fields, "; ")
	} else if e.Detail != "" {
		message += ": " + e.Detail
	}
	if e.RetryAfter > 0 {
		message += "; retry after " + e.RetryAfter.String()
	}
	return message
}

// maxRetryAfter bounds how long a service hint can stall a caller, whether it
// arrives as seconds or as an HTTP date.
const maxRetryAfter = time.Minute

func retryAfter(header string) time.Duration {
	var wait time.Duration
	if seconds, err := strconv.ParseInt(header, 10, 32); err == nil {
		wait = time.Duration(seconds) * time.Second
	} else if date, err := http.ParseTime(header); err == nil {
		wait = time.Until(date)
	}
	return min(max(wait, 0), maxRetryAfter)
}

var irisErrorCode = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,80}$`)

func irisResponseError(response *http.Response, secrets []string) *IrisHTTPError {
	result := &IrisHTTPError{StatusCode: response.StatusCode, RetryAfter: retryAfter(response.Header.Get("Retry-After"))}
	// Parse bounded problem JSON. Validation details are useful when Iris cannot
	// identify a field, but must have request and authentication secrets redacted.
	raw, err := io.ReadAll(io.LimitReader(response.Body, (16<<10)+1))
	if err != nil || len(raw) > 16<<10 {
		return result
	}
	var problem struct {
		Code   string `json:"code"`
		Type   string `json:"type"`
		Detail string `json:"detail"`
		Errors []struct {
			Pointer   string `json:"pointer"`
			Parameter string `json:"parameter"`
			Detail    string `json:"detail"`
		} `json:"errors"`
	}
	if json.Unmarshal(raw, &problem) != nil {
		return result
	}
	code := problem.Code
	if code == "" && problem.Type != "" {
		if u, err := url.Parse(problem.Type); err == nil && u.RawQuery == "" && u.Fragment == "" {
			code = path.Base(u.Path)
		}
	}
	code = irisErrorText(code, secrets)
	if irisErrorCode.MatchString(code) {
		result.Code = code
	}
	if response.StatusCode == http.StatusBadRequest || response.StatusCode == http.StatusUnprocessableEntity {
		result.Detail = irisErrorText(problem.Detail, secrets)
	}
	for _, field := range problem.Errors {
		if len(result.Fields) == 8 {
			break
		}
		member := field.Pointer
		if member == "" {
			member = field.Parameter
		}
		if member == "" {
			continue
		}
		// Iris FieldError.detail is specified to contain fixed rule text, never the value.
		result.Fields = append(result.Fields, irisErrorText(member, secrets)+": "+irisErrorText(field.Detail, secrets))
	}
	return result
}
func irisErrorText(value string, secrets []string) string {
	for _, secret := range secrets {
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "[REDACTED]")
		}
	}
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
	runes := []rune(value)
	if len(runes) > 256 {
		value = string(runes[:256]) + "..."
	}
	return value
}
func irisRequestSecrets(payload []byte) []string {
	var root any
	if json.Unmarshal(payload, &root) != nil {
		return nil
	}
	secrets := []string{}
	var visit func(any)
	visit = func(value any) {
		switch v := value.(type) {
		case map[string]any:
			for name, child := range v {
				switch strings.ToLower(strings.ReplaceAll(name, "_", "")) {
				case "apikey", "secretkey", "token", "password", "authorization":
					if text, ok := child.(string); ok && text != "" {
						secrets = append(secrets, text)
					}
				}
				visit(child)
			}
		case []any:
			for _, child := range v {
				visit(child)
			}
		}
	}
	visit(root)
	return secrets
}
