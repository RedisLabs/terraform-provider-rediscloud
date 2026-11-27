package iris

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"syscall"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func TestReadinessRetryableResponses(t *testing.T) {
	for _, status := range []int{404, 408, 429, 500, 502, 503, 504, 520, 521, 522, 523, 524} {
		require.True(t, retryableStoreRead(&client.IrisHTTPError{StatusCode: status}))
	}
	for _, status := range []int{400, 401, 403, 409, 422, 501} {
		require.False(t, retryableStoreRead(&client.IrisHTTPError{StatusCode: status}))
	}
	require.False(t, retryableStoreRead(context.Canceled))
	require.True(t, retryableStoreRead(context.DeadlineExceeded))
}

// A connection closed or a body cut off mid-response says nothing about the
// store, so the read is repeated. Responses the client rejects as malformed stay
// fatal because a repeat cannot fix them.
func TestReadinessRetriesDroppedResponses(t *testing.T) {
	for name, err := range map[string]error{
		"truncated body":   fmt.Errorf("read Iris response failed: %w", io.ErrUnexpectedEOF),
		"closed before":    &url.Error{Op: "Get", URL: "https://iris.example.com", Err: io.EOF},
		"connection reset": fmt.Errorf("iris request failed: %w", &url.Error{Op: "Get", URL: "https://iris.example.com", Err: syscall.ECONNRESET}),
	} {
		require.True(t, retryableStoreRead(err), name)
	}
	require.False(t, retryableStoreRead(errors.New("iris returned an invalid JSON object")))
}

func TestReadinessSurvivesTruncatedResponse(t *testing.T) {
	calls := 0
	r := testResource(t, "iris_agent_memory", func(w http.ResponseWriter, req *http.Request) {
		calls++
		if calls == 1 {
			// Promise more bytes than are sent; the server then drops the connection.
			w.Header().Set("Content-Length", "100")
			_, _ = w.Write([]byte(`{"storeId":`))
			return
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"storeId": "store-1", "state": "READY"}))
	}).(*agentMemoryResource)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	current, err := r.read(ctx, &AgentMemoryModel{Timeouts: nullTimeouts(), ID: types.StringValue("store-1")}, true)
	require.NoError(t, err)
	require.Equal(t, "READY", *current.State)
	require.Equal(t, 2, calls)
}

func TestReadinessSurvivesInitialNotFound(t *testing.T) {
	calls := 0
	r := testResource(t, "iris_agent_memory", func(w http.ResponseWriter, req *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(404)
			return
		}
		require.Equal(t, http.MethodGet, req.Method)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"storeId": "store-1", "state": "READY"}))
	}).(*agentMemoryResource)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	current, err := r.read(ctx, &AgentMemoryModel{Timeouts: nullTimeouts(), ID: types.StringValue("store-1")}, true)
	require.NoError(t, err)
	require.Equal(t, "READY", *current.State)
	require.Equal(t, 2, calls)
}
