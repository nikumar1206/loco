package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
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

	c.HTTPClient.Timeout = 0 // disable timeout for streaming requests
	logsClient := appv1connect.NewAppServiceClient(&c.HTTPClient, c.BaseURL)

	req := connect.NewRequest(logsRequest)
	req.Header().Set("Authorization", authHeader)

	stream, err := logsClient.Logs(ctx, req)
	if err != nil {
		errChan <- fmt.Errorf("failed to initiate log stream: %w", err)
	}
	for stream.Receive() {
		msg := stream.Msg()

		logsChan <- LogEntry{
			Timestamp: msg.Timestamp.AsTime(),
			PodName:   msg.PodName,
			Log:       msg.Log,
		}
	}

	if err := stream.Err(); err != nil && !errors.Is(err, context.Canceled) {
		errChan <- err
	}
}
