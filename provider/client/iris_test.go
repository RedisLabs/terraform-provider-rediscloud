package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIrisTransportIsolationAndRedirects(t *testing.T) {
	redirected := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { redirected = true }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer service-secret", r.Header.Get("Authorization"))
		assert.Empty(t, r.Header.Get("x-api-key"))
		assert.Equal(t, "/proxy/v1/resource", r.URL.Path)
		assert.Equal(t, "a+/=&", r.URL.Query().Get("cursor"))
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	connector, err := NewIrisServiceClient(IrisServiceConfig{URL: server.URL + "/proxy/", Token: "service-secret"})
	require.NoError(t, err)
	err = connector.Do(context.Background(), http.MethodPost, "/v1/resource", url.Values{"cursor": {"a+/=&"}}, map[string]any{"token": "body-secret"}, nil)
	require.Error(t, err)
	require.False(t, redirected)
	assert.NotContains(t, err.Error(), "service-secret")
	assert.NotContains(t, err.Error(), "body-secret")
}
func TestIrisTransportRejectsInvalidResponsesWithoutLeakingBodies(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"malformed", `{"secret":"private"`, 200}, {"trailing", `{} {"secret":"private"}`, 200}, {"array", `[{"secret":"private"}]`, 200}, {"null", `null`, 200}, {"api-error", `{"message":"private"}`, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			c, err := NewIrisServiceClient(IrisServiceConfig{URL: server.URL, Token: "test"})
			require.NoError(t, err)
			var response map[string]any
			err = c.Do(context.Background(), http.MethodGet, "/resource", nil, nil, &response)
			require.Error(t, err)
			require.NotContains(t, err.Error(), "private")
		})
	}
}
func TestIrisTransportCancellationAndConfiguration(t *testing.T) {
	for _, endpoint := range []string{"", "://bad", "ftp://example.com", "https://user:pass@example.com", "https://example.com?x=y", "https://example.com?", "https://example.com/prefix?", "https://example.com#fragment"} {
		_, err := NewIrisServiceClient(IrisServiceConfig{URL: endpoint, Token: "secret"})
		require.Error(t, err)
	}
	_, err := NewIrisServiceClient(IrisServiceConfig{URL: "https://example.com"})
	require.Error(t, err)
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer server.Close()
	c, err := NewIrisServiceClient(IrisServiceConfig{URL: server.URL, Token: "secret"})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = c.Do(ctx, http.MethodPost, "/resource", nil, map[string]any{}, nil)
	require.True(t, errors.Is(err, context.Canceled))
	require.False(t, called)
	require.False(t, IrisNotFound(errors.New(strings.Repeat("error", 2))))
	require.True(t, IrisNotFound(&IrisHTTPError{StatusCode: 404}))
}

func TestIrisCloudGatewayAuthentication(t *testing.T) {
	for _, bearer := range []string{"", "explicit-token"} {
		t.Run(bearer, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if bearer != "" {
					assert.Equal(t, "Bearer "+bearer, r.Header.Get("Authorization"))
					assert.Empty(t, r.Header.Get("x-api-key"))
					assert.Empty(t, r.Header.Get("x-api-secret-key"))
				} else {
					assert.Empty(t, r.Header.Get("Authorization"))
					assert.Equal(t, "fake-access", r.Header.Get("x-api-key"))
					assert.Equal(t, "fake-secret", r.Header.Get("x-api-secret-key"))
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			c, err := NewIrisServiceClient(IrisServiceConfig{URL: server.URL, Token: bearer, APIKey: "fake-access", SecretKey: "fake-secret"})
			require.NoError(t, err)
			err = c.Do(context.Background(), http.MethodGet, "/iris/v1/agent-memory/stores", nil, nil, nil)
			require.NoError(t, err)
		})
	}
}

func TestIrisRequiresTLSOutsideLoopback(t *testing.T) {
	for _, config := range []IrisServiceConfig{{Token: "bearer"}, {APIKey: "cloud-key", SecretKey: "cloud-secret"}} {
		for _, endpoint := range []string{"http://example.com", "http://192.168.0.1", "http://0.0.0.0", "http://localhost.example.com", "http://[::]"} {
			config.URL = endpoint
			_, err := NewIrisServiceClient(config)
			require.ErrorContains(t, err, "HTTPS")
		}
		for _, endpoint := range []string{"https://example.com", "http://localhost:8080", "http://127.0.0.1:8080/proxy", "http://[::1]:8080"} {
			config.URL = endpoint
			_, err := NewIrisServiceClient(config)
			require.NoError(t, err)
		}
	}
}

func TestIrisProblemErrorsAreUsefulAndRedacted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"type":"https://errors.example.com/invalid-request","detail":"private internal data","message":"private response","errors":[{"pointer":"/llm/credentials/apiKey","detail":"invalid model-secret bearer-secret cloud-secret\nvalue"}]}`))
	}))
	defer server.Close()
	c, err := NewIrisServiceClient(IrisServiceConfig{URL: server.URL, Token: "bearer-secret", SecretKey: "cloud-secret"})
	require.NoError(t, err)
	err = c.Do(context.Background(), http.MethodPost, "/stores", nil, map[string]any{"llm": map[string]any{"credentials": map[string]string{"apiKey": "model-secret"}}}, nil)
	var response *IrisHTTPError
	require.True(t, errors.As(err, &response))
	require.Equal(t, 3*time.Second, response.RetryAfter)
	require.Contains(t, err.Error(), "Too Many Requests")
	require.Contains(t, err.Error(), "invalid-request")
	require.Contains(t, err.Error(), "/llm/credentials/apiKey")
	require.Contains(t, err.Error(), "[REDACTED]")
	for _, secret := range []string{"model-secret", "bearer-secret", "cloud-secret", "private internal data", "private response", "\n"} {
		require.NotContains(t, err.Error(), secret)
	}
	require.LessOrEqual(t, len(irisErrorText(string(make([]byte, 5000)), nil)), 259)
}

func TestIrisRetryAfterIsBounded(t *testing.T) {
	for header, expected := range map[string]time.Duration{
		"":           0,
		"nonsense":   0,
		"-5":         0,
		"0":          0,
		"3":          3 * time.Second,
		"86400":      maxRetryAfter,
		"2147483647": maxRetryAfter,
		time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat):     0,
		time.Now().Add(24 * time.Hour).UTC().Format(http.TimeFormat): maxRetryAfter,
	} {
		require.Equal(t, expected, retryAfter(header), header)
	}
	soon := retryAfter(time.Now().Add(20 * time.Second).UTC().Format(http.TimeFormat))
	require.Greater(t, soon, 10*time.Second)
	require.LessOrEqual(t, soon, 20*time.Second)
}

func TestIrisTopLevelValidationDetailIsRedactedAndBounded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"type": "/errors/invalid-data", "detail": "llm model must be specified; rejected model-secret bearer-secret cloud-secret\n" + strings.Repeat("x", 1000)})
	}))
	defer server.Close()
	c, err := NewIrisServiceClient(IrisServiceConfig{URL: server.URL, Token: "bearer-secret", SecretKey: "cloud-secret"})
	require.NoError(t, err)
	err = c.Do(context.Background(), http.MethodPost, "/stores", nil, map[string]any{"credentials": map[string]string{"apiKey": "model-secret"}}, nil)
	require.ErrorContains(t, err, "llm model must be specified")
	for _, secret := range []string{"model-secret", "bearer-secret", "cloud-secret", "\n"} {
		require.NotContains(t, err.Error(), secret)
	}
	require.Contains(t, err.Error(), "[REDACTED]")
	require.Less(t, len(err.Error()), 400)
}

type irisRoundTripper func(*http.Request) (*http.Response, error)

func (f irisRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestIrisPreservesOperationDeadline(t *testing.T) {
	c, err := NewIrisServiceClient(IrisServiceConfig{URL: "https://memory.example.com", Token: "secret"})
	require.NoError(t, err)
	c.http.Transport = irisRoundTripper(func(req *http.Request) (*http.Response, error) {
		deadline, ok := req.Context().Deadline()
		require.True(t, ok)
		require.Greater(t, time.Until(deadline), 4*time.Minute)
		return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	require.NoError(t, c.Do(ctx, http.MethodDelete, "/stores/store-1", nil, nil, nil))
}

func TestIrisReadAttemptDeadline(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		for _, callerLimit := range []time.Duration{0, 10 * time.Second, 5 * time.Minute} {
			t.Run(method+"/"+callerLimit.String(), func(t *testing.T) {
				c, err := NewIrisServiceClient(IrisServiceConfig{URL: "https://memory.example.com", Token: "secret"})
				require.NoError(t, err)
				ctx := context.Background()
				if callerLimit > 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, callerLimit)
					defer cancel()
				}
				limit := time.Minute
				if callerLimit > 0 {
					limit = callerLimit
				}
				var attempt context.Context
				c.http.Transport = irisRoundTripper(func(req *http.Request) (*http.Response, error) {
					attempt = req.Context()
					deadline, ok := attempt.Deadline()
					require.True(t, ok)
					require.WithinDuration(t, time.Now().Add(limit), deadline, time.Second)
					return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
				})
				require.NoError(t, c.Do(ctx, method, "/stores/store-1", nil, nil, nil))
				if callerLimit == 0 {
					require.ErrorIs(t, attempt.Err(), context.Canceled, "release the fallback context after reading the response")
				}
				require.NoError(t, ctx.Err(), "the read must not cancel its parent operation")
			})
		}
	}
}

func TestIrisResponseBodyTimeoutRemainsRetryable(t *testing.T) {
	c, err := NewIrisServiceClient(IrisServiceConfig{URL: "https://memory.example.com", Token: "secret"})
	require.NoError(t, err)
	c.http.Transport = irisRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(iotest.ErrReader(context.DeadlineExceeded)), Header: make(http.Header)}, nil
	})
	err = c.Do(context.Background(), http.MethodGet, "/stores/store-1", nil, nil, nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
