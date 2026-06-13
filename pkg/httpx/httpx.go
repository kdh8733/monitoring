// Package httpx is a thin JSON-over-HTTP helper shared by the integration
// clients (grafana, argocd, github, kube, slackapi). Every external system in
// this service follows the same {BaseURL, Token} shape, so the request glue
// lives here once.
package httpx

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps an *http.Client with sane defaults.
type Client struct {
	HTTP *http.Client
}

// New returns a Client with a request timeout. insecure skips TLS verification
// (used only for in-cluster Kubernetes API access with a self-signed CA).
func New(timeout time.Duration, insecure bool) *Client {
	tr := &http.Transport{}
	if insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &Client{HTTP: &http.Client{Timeout: timeout, Transport: tr}}
}

// Auth describes how to attach credentials to a request.
type Auth struct {
	// Bearer is sent as "Authorization: Bearer <token>" when non-empty.
	Bearer string
	// Header/HeaderValue sets an arbitrary header (e.g. Grafana, GitHub).
	Header      string
	HeaderValue string
}

// DoJSON performs an HTTP request and decodes a JSON response into out
// (out may be nil to ignore the body). body, when non-nil, is JSON-encoded.
// A non-2xx status is returned as an error including the response snippet.
func (c *Client) DoJSON(ctx context.Context, method, url string, auth Auth, body, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if auth.Bearer != "" {
		req.Header.Set("Authorization", "Bearer "+auth.Bearer)
	}
	if auth.Header != "" {
		req.Header.Set(auth.Header, auth.HeaderValue)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: status %d: %s", method, url, resp.StatusCode, snippet(data))
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode %s: %w", url, err)
		}
	}
	return nil
}

func snippet(b []byte) string {
	const max = 300
	if len(b) > max {
		return string(b[:max]) + "..."
	}
	return string(b)
}
