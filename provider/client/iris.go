package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// IrisServiceConfig supports service bearer tokens or Cloud gateway API credentials.
type IrisServiceConfig struct{ URL, Token, APIKey, SecretKey string }

// IrisServiceClient sends synchronous requests to one direct-service control plane.
// It never retries mutations or logs request/response bodies containing credentials.
type IrisServiceClient struct {
	baseURL   *url.URL
	token     string
	apiKey    string
	secretKey string
	http      *http.Client
}

type IrisHTTPError struct{ StatusCode int }

func (e *IrisHTTPError) Error() string { return fmt.Sprintf("Iris API returned HTTP %d", e.StatusCode) }
func IrisNotFound(err error) bool {
	var e *IrisHTTPError
	return errors.As(err, &e) && e.StatusCode == http.StatusNotFound
}
func NewIrisServiceClient(config IrisServiceConfig) (*IrisServiceClient, error) {
	u, err := url.Parse(config.URL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("iris URL must be an HTTP(S) origin or path prefix without credentials, query or fragment")
	}
	if config.Token == "" && (config.APIKey == "" || config.SecretKey == "") {
		return nil, errors.New("an Iris bearer token or Redis Cloud API key and secret is required when its service URL is set")
	}
	return &IrisServiceClient{baseURL: u, token: config.Token, apiKey: config.APIKey, secretKey: config.SecretKey, http: &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *IrisServiceClient) Call(ctx context.Context, method, path string, query url.Values, body map[string]any) (map[string]any, error) {
	var input io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode Iris request: %w", err)
		}
		input = bytes.NewReader(b)
	}
	endpoint := strings.TrimRight(c.baseURL.String(), "/") + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, input)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	} else {
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("x-api-secret-key", c.secretKey)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "terraform-provider-rediscloud/iris")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("iris request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &IrisHTTPError{StatusCode: resp.StatusCode}
	}
	if resp.StatusCode == http.StatusNoContent {
		return map[string]any{}, nil
	}
	const maxResponse = 16 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil {
		return nil, errors.New("read Iris response failed")
	}
	if len(data) > maxResponse {
		return nil, errors.New("iris response exceeds 16 MiB")
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]any{}, nil
	}
	var result map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err = decoder.Decode(&result); err != nil || result == nil {
		return nil, errors.New("iris returned an invalid JSON object")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil, errors.New("iris response contains trailing data")
	}
	return result, nil
}
