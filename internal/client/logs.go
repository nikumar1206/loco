package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	json "github.com/goccy/go-json"
	appv1 "github.com/nikumar1206/loco/proto/app/v1"
	appv1connect "github.com/nikumar1206/loco/proto/app/v1/appv1connect"
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
func (c *Client) StreamLogs(ctx context.Context, locoToken string, logsRequest *appv1.LogsRequest, logsChan chan<- LogEntry, errChan chan<- error) {
	authHeader := fmt.Sprintf("Bearer %s", locoToken)

	logsClient := appv1connect.NewAppServiceClient(&c.HTTPClient, c.BaseURL)

	req := connect.NewRequest(logsRequest)
	req.Header().Set("Authorization", authHeader)

	resp, err := logsClient.Logs(ctx, connect.NewRequest(logsRequest))
	if err != nil {
		errChan <- fmt.Errorf("failed to get logs: %w", err)
		return
	}

	for _, log_line := range resp.Msg.LogLine {
		line := log_line.Log
		if len(line) >= 6 && line[:6] == "data: " {
			line = line[6:]
		}
		var logEntry LogEntry
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			continue
		}
		logsChan <- logEntry
	}
}
