package shared

import (
	"net/http"

	"golang.org/x/net/http2"
)

// NewHTTPClient creates an HTTP client with HTTP/2 support enabled.
func NewHTTPClient() *http.Client {
	transport := &http.Transport{}
	http2.ConfigureTransport(transport)
	return &http.Client{Transport: transport}
}
