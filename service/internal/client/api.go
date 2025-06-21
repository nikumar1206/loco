package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient http.Client
}

func NewAPIClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error: status %d, body: %s", e.StatusCode, e.Body)
}

func (c *Client) doRequest(method, path string, body []byte, headers map[string]string) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if headers == nil {
		headers = make(map[string]string)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	return respBody, nil
}

// Example usage methods
func (c *Client) Get(path string, headers map[string]string) ([]byte, error) {
	return c.doRequest(http.MethodGet, path, nil, headers)
}

func (c *Client) Post(path string, body []byte, headers map[string]string) ([]byte, error) {
	return c.doRequest(http.MethodPost, path, body, headers)
}
