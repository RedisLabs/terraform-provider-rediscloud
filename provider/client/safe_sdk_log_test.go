package client

import (
	"strings"
	"testing"
)

func TestSafeSDKLogPreservesContext(t *testing.T) {
	const secret = "FAKE_LOG_CONTENT_NEVER_ECHO"
	for _, tc := range []struct{ name, message, want string }{
		{"empty", "", "details were omitted"},
		{"unknown", "unexpected event " + secret, "details were omitted"},
		{"request-unknown-format", "DEBUG: Request /" + secret, "HTTP request (payload omitted)"},
		{"response-unknown-format", "DEBUG: Response /" + secret, "HTTP response (payload omitted)"},
		{"request", "DEBUG: Request /" + secret + ":\nPOST /memory-stores HTTP/1.1\r\n\r\n" + secret, "HTTP POST request"},
		{"response", "DEBUG: Response /" + secret + ":\nHTTP/1.1 403 " + secret + "\r\n\r\n" + secret, "403 Forbidden"},
		{"new-status", "DEBUG: Response /" + secret + ":\nHTTP/1.1 599 " + secret, "HTTP response: 599"},
		{"invalid-status", "DEBUG: Response /" + secret + ":\nHTTP/1.1 XYZ " + secret, "HTTP response (payload omitted)"},
		{"task-progress", "task " + secret + " not processed yet: received", "task is still processing"},
		{"task-failure", "task " + secret + " failed: " + secret, "task failed"},
		{"waiting", "Waiting for Agent Memory store " + secret + " to finish being deleted", "operation to complete"},
		{"rate-limit", "status code 429 received, request will be retried " + secret, "retry pending"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := SafeSDKLog(tc.message)
			if strings.Contains(result, secret) {
				t.Error("SDK log exposed sensitive event contents")
			}
			if !strings.Contains(result, tc.want) {
				t.Errorf("missing safe event context %q", tc.want)
			}
		})
	}
}

func FuzzSafeSDKLogUnknownFormats(f *testing.F) {
	f.Add("DEBUG: Request /store:\nPOST /memory-stores HTTP/1.1\n")
	f.Add("DEBUG: Response /store:\nHTTP/1.1 500 Internal Server Error\n")
	f.Add("task id failed: ")
	f.Fuzz(func(t *testing.T, payload string) {
		const secret = "FAKE_LOG_SENTINEL_211f140f2d9840fea232aedf7e1e831c"
		if strings.Contains(SafeSDKLog(payload+secret), secret) {
			t.Fatal("unknown SDK log format exposed payload contents")
		}
	})
}
