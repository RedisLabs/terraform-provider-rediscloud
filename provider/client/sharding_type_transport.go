package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
)

// ShardingTypeCustomRegexRules is the Cloud API shardingType used when a
// database is created with a caller-supplied hashing policy.
//
// rediscloud-go-api v0.53.0 CreateDatabase has no shardingType field, so the
// provider cannot set it on the typed request. DatabaseCreateRequest in the
// Cloud API does accept it, and a custom hashing policy is rejected later
// unless the database was created as custom-regex-rules.
const ShardingTypeCustomRegexRules = "custom-regex-rules"

type shardingTypeContextKey struct{}

// WithShardingType marks ctx so ShardingTypeTransport adds shardingType to the
// database create request made with that context. Other requests are unchanged.
func WithShardingType(ctx context.Context, shardingType string) context.Context {
	return context.WithValue(ctx, shardingTypeContextKey{}, shardingType)
}

func shardingTypeFromContext(ctx context.Context) (string, bool) {
	shardingType, ok := ctx.Value(shardingTypeContextKey{}).(string)
	if !ok || shardingType == "" {
		return "", false
	}
	return shardingType, true
}

// databaseCreatePath matches POST /subscriptions/{subscriptionId}/databases and
// nothing nested under it, such as a database update or upgrade.
var databaseCreatePath = regexp.MustCompile(`^/v1/subscriptions/[^/]+/databases/?$`)

// ShardingTypeTransport injects a context-supplied shardingType into a database
// create body. base may be nil, in which case http.DefaultTransport is used.
func NewShardingTypeTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &shardingTypeTransport{base: base}
}

type shardingTypeTransport struct {
	base http.RoundTripper
}

func (t *shardingTypeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	shardingType, ok := shardingTypeFromContext(req.Context())
	if !ok || req.Method != http.MethodPost || req.Body == nil || req.URL == nil || !databaseCreatePath.MatchString(req.URL.Path) {
		return t.base.RoundTrip(req)
	}

	original, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return nil, err
	}

	updated, err := injectShardingType(original, shardingType)
	if err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(bytes.NewReader(updated))
	req.ContentLength = int64(len(updated))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(updated)), nil
	}
	return t.base.RoundTrip(req)
}

// injectShardingType adds shardingType when the create body does not already
// carry one. A future client that sets the field itself keeps its value.
func injectShardingType(body []byte, shardingType string) ([]byte, error) {
	payload := map[string]json.RawMessage{}
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}
	}
	if _, exists := payload["shardingType"]; exists {
		return body, nil
	}
	encoded, err := json.Marshal(shardingType)
	if err != nil {
		return nil, err
	}
	payload["shardingType"] = encoded
	return json.Marshal(payload)
}
