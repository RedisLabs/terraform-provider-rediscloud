package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// IrisServiceConfig supports service bearer tokens or Cloud gateway API credentials.
type IrisServiceConfig struct{ URL, Token, APIKey, SecretKey, Version string }

// IrisServiceClient sends synchronous requests to one direct-service control plane.
// It never retries mutations or logs request/response bodies containing credentials.
type IrisServiceClient struct {
	baseURL   *url.URL
	token     string
	apiKey    string
	secretKey string
	http      *http.Client
	userAgent string
}

func IrisNotFound(err error) bool {
	var e *IrisHTTPError
	return errors.As(err, &e) && e.StatusCode == http.StatusNotFound
}
func irisServiceURL(config IrisServiceConfig) (*url.URL, error) {
	u, err := url.Parse(config.URL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return nil, errors.New("iris URL must be an HTTP(S) origin or path prefix without credentials, query or fragment")
	}
	if u.Scheme == "http" {
		host := u.Hostname()
		ip := net.ParseIP(host)
		if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
			return nil, errors.New("iris URL must use HTTPS except for loopback development endpoints")
		}
	}
	if config.Token == "" && (config.APIKey == "" || config.SecretKey == "") {
		return nil, errors.New("an Iris bearer token or Redis Cloud API key and secret is required when its service URL is set")
	}
	return u, nil
}

// ValidateIrisServiceConfig checks settings without constructing an unused transport.
func ValidateIrisServiceConfig(config IrisServiceConfig) error {
	_, err := irisServiceURL(config)
	return err
}
func NewIrisServiceClient(config IrisServiceConfig) (*IrisServiceClient, error) {
	u, err := irisServiceURL(config)
	if err != nil {
		return nil, err
	}
	version := config.Version
	if version == "" {
		version = "dev"
	}
	return &IrisServiceClient{userAgent: "terraform-provider-rediscloud/" + version + " (iris)", baseURL: u, token: config.Token, apiKey: config.APIKey, secretKey: config.SecretKey, http: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *IrisServiceClient) Do(ctx context.Context, method, path string, query url.Values, body, result any) error {
	if _, bounded := ctx.Deadline(); !bounded && (method == http.MethodGet || method == http.MethodHead) {
		// Resource callers own their deadlines; bound standalone reads as well.
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Minute)
		defer cancel()
	}
	var input io.Reader
	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode Iris request: %w", err)
		}
		payload = b
		input = bytes.NewReader(b)
	}
	endpoint := strings.TrimRight(c.baseURL.String(), "/") + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, input)
	if err != nil {
		return err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	} else {
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("x-api-secret-key", c.secretKey)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("iris request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return irisResponseError(resp, append([]string{c.token, c.apiKey, c.secretKey}, irisRequestSecrets(payload)...))
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	const maxResponse = 16 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil {
		return fmt.Errorf("read Iris response failed: %w", err)
	}
	if len(data) > maxResponse {
		return errors.New("iris response exceeds 16 MiB")
	}
	if len(bytes.TrimSpace(data)) == 0 || result == nil {
		return nil
	}
	if bytes.TrimSpace(data)[0] != '{' {
		return errors.New("iris returned an invalid JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err = decoder.Decode(result); err != nil {
		return errors.New("iris returned an invalid JSON object")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return errors.New("iris response contains trailing data")
	}
	return nil
}
