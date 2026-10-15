package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestSafeAPIError(t *testing.T) {
	const secret = "FAKE_CREDENTIAL_SHOULD_NEVER_ESCAPE"
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"cancel", fmt.Errorf("%s: %w", secret, context.Canceled), "cancelled"},
		{"deadline", fmt.Errorf("%s: %w", secret, context.DeadlineExceeded), "timed out"},
		{"HTTP401", fmt.Errorf("outer %s: %w", secret, errors.New("failed to create Agent Memory store: 401 - "+secret)), "HTTP 401"},
		{"HTTP403", errors.New("failed to create Agent Memory store: 403 - " + secret), "permission"},
		{"HTTP404", errors.New("failed to get Agent Memory store: 404 - " + secret), "not found"},
		{"HTTP409", errors.New("failed to create Agent Memory store: 409 - " + secret), "already exist"},
		{"HTTP429", errors.New("failed to create Agent Memory store: 429 - " + secret), "rate limit"},
		{"HTTP503", errors.New("failed to get Agent Memory store: 503 - " + secret), "service health"},
		{"task", errors.New("task failed processing-failed - " + secret), "outcome is unknown"},
		{"url", errors.New("Get https://user:" + secret + "@example.test: failed"), "Error details were omitted"},
		{"new format", errors.New("unexpected SDK response " + secret), "Error details were omitted"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := SafeAPIError(tc.err)
			if strings.Contains(actual, secret) {
				t.Fatal("credential leaked")
			}
			if !strings.Contains(actual, tc.want) {
				t.Fatalf("missing expected guidance %q in %q", tc.want, actual)
			}
		})
	}
}

func TestSafeSDKLog(t *testing.T) {
	const secret = "FAKE_CREDENTIAL_SHOULD_NEVER_ESCAPE"
	cases := []struct{ name, message, want string }{
		{"request", "DEBUG: Request /memory-stores/" + secret + ":\n---[ REQUEST ]---\nPOST /v1/memory-stores HTTP/1.1\r\nAuthorization: " + secret + "\r\n\r\n{\"apiKey\":\"" + secret + "\"}", "HTTP POST request"},
		{"response", "DEBUG: Response /" + secret + ":\n---[ RESPONSE ]---\nHTTP/1.1 400 " + secret + "\r\nSet-Cookie: " + secret + "\r\n\r\n{\"message\":\"" + secret + "\"}", "400 Bad Request"},
		{"wait", "Waiting for Agent Memory store " + secret + " to finish being updated", "Waiting for Redis Cloud operation"},
		{"task", "task " + secret + " not processed yet: " + secret, "task is still processing"},
		{"rate", "status code 429 received, request will be retried " + secret, "rate limit"},
		{"unknown", secret, "Error details were omitted"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := SafeSDKLog(tc.message)
			if strings.Contains(actual, secret) {
				t.Fatal("credential leaked")
			}
			if !strings.Contains(actual, tc.want) {
				t.Fatalf("missing expected metadata %q in %q", tc.want, actual)
			}
		})
	}
}

func FuzzSafeSDKLogDoesNotEchoPayload(f *testing.F) {
	f.Add("{\"message\":\"secret\"}")
	f.Add("\r\nHTTP/1.1 500 Secret\r\n")
	f.Fuzz(func(t *testing.T, payload string) {
		const secret = "FAKE_SENTINEL_82f7e5d2d9e34f3ba0a82d4accc36cc2"
		message := "DEBUG: Response /" + secret + ":\nHTTP/1.1 500 Internal Server Error\n" + payload + secret
		if strings.Contains(SafeSDKLog(message), secret) {
			t.Fatal("payload escaped redaction")
		}
		if strings.Contains(SafeAPIError(errors.New(payload+secret)), secret) {
			t.Fatal("error escaped redaction")
		}
	})
}
