package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
)

var sdkHTTPErrorStatus = regexp.MustCompile(`^failed to [^:\r\n]+: ([1-5][0-9][0-9]) - `)

// SafeAPIError returns an actionable classification without forwarding API
// response bodies, task descriptions, request URLs or credential-bearing errors.
func SafeAPIError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return "The Redis Cloud request was cancelled. Refresh the resource before retrying an operation whose outcome is unknown."
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "The Redis Cloud request timed out. Refresh the resource before retrying an operation whose outcome is unknown."
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return "The Redis Cloud network request timed out. Check connectivity and refresh the resource before retrying."
	}
	var syntaxError *json.SyntaxError
	var typeError *json.UnmarshalTypeError
	if errors.As(err, &syntaxError) || errors.As(err, &typeError) {
		return "The Redis Cloud API returned an invalid response. Response details were omitted because they may contain credentials."
	}
	// The pinned SDK's HTTPError type is internal and exposes no public status
	// interface. Read only its numeric status prefix; unknown formats fail closed.
	for cause := err; cause != nil; cause = errors.Unwrap(cause) {
		if match := sdkHTTPErrorStatus.FindStringSubmatch(cause.Error()); len(match) == 2 {
			status, _ := strconv.Atoi(match[1])
			guidance := "Check the request configuration, API permissions and service health."
			switch status {
			case http.StatusUnauthorized:
				guidance = "Check the Redis Cloud API credentials."
			case http.StatusForbidden:
				guidance = "Check that the API credentials have permission for this operation."
			case http.StatusNotFound:
				guidance = "The requested resource was not found. Refresh state and verify its identifier."
			case http.StatusConflict:
				guidance = "The resource may be busy or already exist. Refresh state before retrying."
			case http.StatusTooManyRequests:
				guidance = "The API rate limit was reached. Wait before retrying."
			default:
				if status >= 500 {
					guidance = "The Redis Cloud service could not complete the request. Check service health and refresh state before retrying."
				}
			}
			return fmt.Sprintf("The Redis Cloud API returned HTTP %d (%s). %s Response details were omitted because they may contain credentials.", status, http.StatusText(status), guidance)
		}
	}
	return "The Redis Cloud request failed. Check connectivity, API permissions and service health. Error details were omitted because they may contain credentials; refresh state before retrying an operation whose outcome is unknown."
}
