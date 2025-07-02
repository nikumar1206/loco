package api

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"time"

	json "github.com/goccy/go-json"
)

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	PodName   string    `json:"podId"`
	Log       string    `json:"log"`
}

func (c *Client) GetSSE(path string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	return client.Do(req)
}

// StreamLogs connects to the SSE endpoint and yields logs through the provided channel.
func (c *Client) StreamLogs(ctx context.Context, locoToken, appName string, logsChan chan<- LogEntry, errChan chan<- error) {
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", locoToken),
	}

	path := fmt.Sprintf("/api/v1/app/%s/logs", appName)
	resp, err := c.GetSSE(path, headers)
	if err != nil {
		errChan <- fmt.Errorf("failed to get logs: %w", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errChan <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			line := scanner.Text()
			if len(line) >= 6 && line[:6] == "data: " {
				line = line[6:]
			}

			var logEntry LogEntry
			if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
				continue
			}
			// fmt.Println("foo")
			logsChan <- logEntry
		}
	}

	if err := scanner.Err(); err != nil {
		errChan <- fmt.Errorf("error reading log stream: %w", err)
	}
}
