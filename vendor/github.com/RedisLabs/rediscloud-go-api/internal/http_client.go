package internal

import (
	"context"
	"encoding/json"
	"fmt"
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
	parsed := new(url.URL)
	*parsed = *c.baseUrl

	parsed.Path += path

	u := parsed.String()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("failed to create request to %s: %w", name, err)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to %s: %w", name, err)
	}

	if response.StatusCode > 299 {
		return fmt.Errorf("failed to %s: %d", name, response.StatusCode)
	}

	defer response.Body.Close()

	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		return fmt.Errorf("failed to decode response for %s: %w", name, err)
	}

	return nil
}
