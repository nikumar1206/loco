package api

import (
	"fmt"
	"io"
	"net/http"
	"time"

	json "github.com/goccy/go-json"
	"resty.dev/v3"
)

var Resty = resty.New()

type Client struct {
	BaseURL    string
	HTTPClient http.Client
}

func NewClient(baseURL string) *Client {
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
	// No status code → probably request construction or transport error
	if e.StatusCode == 0 {
		return e.Body
	}

	var msg string
	var payload map[string]string
	if err := json.Unmarshal([]byte(e.Body), &payload); err == nil {
		msg = payload["message"]
	}
	if msg == "" {
		msg = e.Body
	}

	switch {
	case e.StatusCode >= 400 && e.StatusCode < 500:
		return fmt.Sprintf("client error: %s", msg)
	case e.StatusCode >= 500:
		return fmt.Sprintf("server error: %s", msg)
	default:
		return fmt.Sprintf("unexpected error: %s", msg)
	}
}

func (c *Client) doRequest(method, path string, body io.Reader, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return nil, &APIError{
			StatusCode: 0,
			Body:       fmt.Sprintf("failed to create request: %v", err),
		}
	}

	if headers == nil {
		headers = make(map[string]string)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, &APIError{
			StatusCode: 0,
			Body:       err.Error(),
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       fmt.Sprintf("failed to read response body: %v", err),
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	return respBody, nil
}

func (c *Client) Get(path string, headers map[string]string) ([]byte, error) {
	return c.doRequest(http.MethodGet, path, nil, headers)
}

func (c *Client) Post(path string, body any, headers map[string]string) ([]byte, error) {
	buf, err := structToBuffer(body)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request body to buffer: %v", err)
	}
	return c.doRequest(http.MethodPost, path, buf, headers)
}
