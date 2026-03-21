package collector

import (
	"crypto/tls"
	"io"
	"net/http"
	"time"
)

// httpClient is a shared HTTP client with sensible timeouts for all collectors.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	},
}

// maxResponseBytes is the maximum response body size (10 MB).
const maxResponseBytes = 10 * 1024 * 1024

// limitedReadAll reads up to maxResponseBytes from r.
func limitedReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, maxResponseBytes))
}
