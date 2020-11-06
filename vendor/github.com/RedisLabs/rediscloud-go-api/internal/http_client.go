package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

type HttpClient struct {
	client  *http.Client
	baseUrl *url.URL
}

func NewHttpClient(client *http.Client, baseUrl string) (*HttpClient, error) {
	parsed, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}
	return &HttpClient{client: client, baseUrl: parsed}, nil
}

func (c *HttpClient) Get(ctx context.Context, name, path string, responseBody interface{}) error {
	return c.connection(ctx, http.MethodGet, name, path, nil, nil, responseBody)
}

func (c *HttpClient) GetWithQuery(ctx context.Context, name, path string, query url.Values, responseBody interface{}) error {
	return c.connection(ctx, http.MethodGet, name, path, query, nil, responseBody)
}

func (c *HttpClient) Put(ctx context.Context, name, path string, requestBody interface{}, responseBody interface{}) error {
	return c.connection(ctx, http.MethodPut, name, path, nil, requestBody, responseBody)
}

func (c *HttpClient) Post(ctx context.Context, name, path string, requestBody interface{}, responseBody interface{}) error {
	return c.connection(ctx, http.MethodPost, name, path, nil, requestBody, responseBody)
}

func (c *HttpClient) Delete(ctx context.Context, name, path string, responseBody interface{}) error {
	return c.connection(ctx, http.MethodDelete, name, path, nil, nil, responseBody)
}

func (c *HttpClient) connection(ctx context.Context, method, name, path string, query url.Values, requestBody interface{}, responseBody interface{}) error {
	parsed := new(url.URL)
	*parsed = *c.baseUrl

	parsed.Path += path
	if query != nil {
		parsed.RawQuery = query.Encode()
	}

	u := parsed.String()

	var body io.Reader
	if requestBody != nil {
		buf := bytes.NewBuffer(nil)
		if err := json.NewEncoder(buf).Encode(requestBody); err != nil {
			return fmt.Errorf("failed to encode request for %s: %w", name, err)
		}
		body = buf
	}

	request, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return fmt.Errorf("failed to create request to %s: %w", name, err)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to %s: %w", name, err)
	}

	defer response.Body.Close()

	if response.StatusCode > 299 {
		body, _ := ioutil.ReadAll(response.Body)
		return &HTTPError{
			Name:       name,
			StatusCode: response.StatusCode,
			Body:       body,
		}
	}

	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		return fmt.Errorf("failed to decode response to %s: %w", name, err)
	}

	return nil
}

type HTTPError struct {
	Name       string
	StatusCode int
	Body       []byte
}

func (h *HTTPError) Error() string {
	return fmt.Sprintf("failed to %s: %d - %s", h.Name, h.StatusCode, h.Body)
}

var _ error = &HTTPError{}
