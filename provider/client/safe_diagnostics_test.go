package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"strings"
	"testing"

	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
)

func TestSafeAPIError(t *testing.T) {
	const secret = "FAKE_CREDENTIAL_SHOULD_NEVER_ESCAPE"
	var parsed any
	syntaxErr := json.Unmarshal([]byte(`{"incomplete":`), &parsed)
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"cancel", fmt.Errorf("%s: %w", secret, context.Canceled), "cancelled"},
		{"deadline", fmt.Errorf("%s: %w", secret, context.DeadlineExceeded), "timed out"},
		{"network-timeout", &url.Error{Op: "GET", URL: "https://user:" + secret + "@example.test", Err: &net.DNSError{Err: secret, IsTimeout: true}}, "network request timed out"},
		{"syntax-error", fmt.Errorf("%s: %w", secret, syntaxErr), "invalid response"},
		{"type-error", &json.UnmarshalTypeError{Value: secret, Type: reflect.TypeOf(0), Struct: secret, Field: secret}, "invalid response"},
		{"HTTP400", errors.New("failed to create Agent Memory store: 400 - " + secret), "request configuration"},
		{"HTTP401-wrapped", fmt.Errorf("outer %s: %w", secret, errors.New("failed to create Agent Memory store: 401 - "+secret)), "HTTP 401"},
		{"HTTP403", errors.New("failed to create Agent Memory store: 403 - " + secret), "permission"},
		{"HTTP404", errors.New("failed to get Agent Memory store: 404 - " + secret), "not found"},
		{"HTTP409", errors.New("failed to create Agent Memory store: 409 - " + secret), "already exist"},
		{"HTTP429", errors.New("failed to create Agent Memory store: 429 - " + secret), "rate limit"},
		{"HTTP500", errors.New("failed to get Agent Memory store: 500 - " + secret), "HTTP 500"},
		{"HTTP503", errors.New("failed to get Agent Memory store: 503 - " + secret), "service health"},
		{"HTTP418", errors.New("failed to get Agent Memory store: 418 - " + secret), "HTTP 418"},
		{"task", errors.New("task task-1 failed processing-failed - " + secret), "task failed"},
		{"url", errors.New("Get https://user:" + secret + "@example.test: failed"), "Error details were omitted"},
		{"unknown-format", errors.New("unexpected SDK response " + secret), "Error details were omitted"},
		{"joined", errors.Join(errors.New(secret), context.Canceled), "cancelled"},
		{"untrusted-operation-name", errors.New("failed to " + secret + ": 403 - " + secret), "HTTP 403"},
		{"multiline-name", errors.New("failed to " + secret + "\noperation: 400 - " + secret), "Error details were omitted"},
		{"body-with-status", errors.New("failed to get Agent Memory store: 500 - " + secret + "\nfailed to get: 401 - data"), "HTTP 500"},
		{"status-outside-prefix", errors.New(secret + " failed to get Agent Memory store: 401 - data"), "Error details were omitted"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual := SafeAPIError(tc.err)
			if strings.Contains(actual, secret) {
				t.Error("credential content escaped into the safe diagnostic")
			}
			if !strings.Contains(actual, tc.want) {
				t.Errorf("missing safe guidance %q", tc.want)
			}
			if tc.err == nil && actual != "" {
				t.Error("nil error produced a diagnostic")
			}
		})
	}
}

func TestSafeAPIErrorPreservesRecoveryContext(t *testing.T) {
	const secret = "FAKE_RECOVERY_SECRET_NEVER_ECHO"
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"create-task-id", errors.New("create Agent Memory store response did not contain a task ID"), "did not return a task ID"},
		{"update-task-id", errors.New("update Agent Memory store response did not contain a task ID"), "did not return a task ID"},
		{"delete-task-id", errors.New("delete Agent Memory store response did not contain a task ID"), "did not return a task ID"},
		{"store-id", errors.New("create Agent Memory store task response did not contain a store ID"), "without returning a store ID"},
		{"key-secret", errors.New("create Agent Memory API key response did not contain an API key"), "secret cannot be recovered by import"},
		{"key-list-id", errors.New("agent memory API key listing contained metadata without an ID"), "metadata without an ID"},
		{"task-resource", errors.New("completed task response did not contain a resource"), "omitted its resource"},
		{"key-unresolved", fmt.Errorf("created Agent Memory API key %q for store %q but could not resolve its ID", secret, secret), "could not uniquely resolve"},
		{"key-ambiguous", fmt.Errorf("created Agent Memory API key %q for store %q but found 2 possible key IDs", secret, secret), "could not uniquely resolve"},
		{"task-failure", fmt.Errorf("task %s failed: backend rejected %s", secret, secret), "task failed"},
		{"store-missing", &agentmemoryapi.NotFound{StoreID: secret}, "not found"},
		{"key-missing", &agentmemoryapi.APIKeyNotFound{StoreID: secret, APIKeyID: secret}, "not found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := SafeAPIError(fmt.Errorf("outer %s: %w", secret, tc.err))
			if strings.Contains(got, secret) {
				t.Error("recovery context included credentials")
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("missing recovery context %q", tc.want)
			}
		})
	}
}

func FuzzSafeAPIErrorDoesNotEchoPayload(f *testing.F) {
	f.Add("failed to get Agent Memory store: 400 - ")
	f.Add("task failed ")
	f.Add("https://user:password@example.test/")
	f.Fuzz(func(t *testing.T, payload string) {
		const secret = "FAKE_SENTINEL_f6f58a5621924c65b581517bde00706e"
		if strings.Contains(SafeAPIError(errors.New(payload+secret)), secret) {
			t.Fatal("payload escaped into the safe diagnostic")
		}
	})
}
