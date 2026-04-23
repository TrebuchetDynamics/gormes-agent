package hermes

import (
	"net/http"
	"time"
)

func newStreamingHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 5 * time.Second
	return &http.Client{Timeout: 0, Transport: transport}
}
