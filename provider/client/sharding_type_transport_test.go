package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type recordingTransport struct {
	req  *http.Request
	body []byte
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.req = req
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		r.body = body
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func TestShardingTypeTransport_InjectsOnDatabaseCreate(t *testing.T) {
	recorder := &recordingTransport{}
	transport := NewShardingTypeTransport(recorder)
	req, err := http.NewRequestWithContext(
		WithShardingType(context.Background(), ShardingTypeCustomRegexRules),
		http.MethodPost,
		"https://api.redislabs.com/v1/subscriptions/42/databases",
		bytes.NewReader([]byte(`{"name":"example"}`)),
	)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	var payload map[string]any
	require.NoError(t, json.Unmarshal(recorder.body, &payload))
	require.Equal(t, "example", payload["name"])
	require.Equal(t, ShardingTypeCustomRegexRules, payload["shardingType"])
}

func TestShardingTypeTransport_LeavesOtherRequestsAlone(t *testing.T) {
	cases := []struct {
		name   string
		method string
		url    string
		ctx    context.Context
	}{
		{
			name:   "create without sharding context",
			method: http.MethodPost,
			url:    "https://api.redislabs.com/v1/subscriptions/42/databases",
			ctx:    context.Background(),
		},
		{
			name:   "database update",
			method: http.MethodPut,
			url:    "https://api.redislabs.com/v1/subscriptions/42/databases/7",
			ctx:    WithShardingType(context.Background(), ShardingTypeCustomRegexRules),
		},
		{
			name:   "upgrade",
			method: http.MethodPost,
			url:    "https://api.redislabs.com/v1/subscriptions/42/databases/7/upgrade",
			ctx:    WithShardingType(context.Background(), ShardingTypeCustomRegexRules),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &recordingTransport{}
			transport := NewShardingTypeTransport(recorder)
			req, err := http.NewRequestWithContext(tc.ctx, tc.method, tc.url, bytes.NewReader([]byte(`{"name":"example"}`)))
			require.NoError(t, err)

			resp, err := transport.RoundTrip(req)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())

			var payload map[string]any
			require.NoError(t, json.Unmarshal(recorder.body, &payload))
			require.Equal(t, map[string]any{"name": "example"}, payload)
		})
	}
}

func TestInjectShardingType_KeepsExistingValue(t *testing.T) {
	body, err := injectShardingType([]byte(`{"shardingType":"redis-oss-hashing"}`), ShardingTypeCustomRegexRules)
	require.NoError(t, err)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "redis-oss-hashing", payload["shardingType"])
}
