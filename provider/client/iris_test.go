package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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
	_, err = connector.Call(context.Background(), http.MethodPost, "/v1/resource", url.Values{"cursor": {"a+/=&"}}, map[string]any{"token": "body-secret"})
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
			_, err = c.Call(context.Background(), http.MethodGet, "/resource", nil, nil)
			require.Error(t, err)
			require.NotContains(t, err.Error(), "private")
		})
	}
}
func TestIrisTransportCancellationAndConfiguration(t *testing.T) {
	for _, endpoint := range []string{"", "://bad", "ftp://example.com", "https://user:pass@example.com", "https://example.com?x=y", "https://example.com#fragment"} {
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
	_, err = c.Call(ctx, http.MethodPost, "/resource", nil, map[string]any{})
	require.True(t, errors.Is(err, context.Canceled))
	require.False(t, called)
	require.False(t, IrisNotFound(errors.New(strings.Repeat("error", 2))))
	require.True(t, IrisNotFound(&IrisHTTPError{StatusCode: 404}))
}
