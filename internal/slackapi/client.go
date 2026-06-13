// Package slackapi is a thin Slack Web API client (bot token) covering only
// what the orchestrator needs: identity lookup, threaded messages, and modal
// views. Slack returns HTTP 200 with {"ok":false} on logical errors, so each
// call checks the ok field.
package slackapi

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/kdh8733/monitoring/internal/httpx"
)

type Client struct {
	baseURL string
	token   string
	http    *httpx.Client
}

func New(baseURL, botToken string) *Client {
	if baseURL == "" {
		baseURL = "https://slack.com/api"
	}
	return &Client{baseURL: trimSlash(baseURL), token: botToken, http: httpx.New(10*time.Second, false)}
}

type apiResp struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	User  struct {
		ID string `json:"id"`
	} `json:"user"`
	TS string `json:"ts"`
}

// UserIDByEmail returns the Slack user ID for an email, or "" with an error
// when no match exists (callers treat that as a graceful fallback).
func (c *Client) UserIDByEmail(ctx context.Context, email string) (string, error) {
	if c.token == "" {
		return "", fmt.Errorf("slack: not configured")
	}
	u := fmt.Sprintf("%s/users.lookupByEmail?email=%s", c.baseURL, url.QueryEscape(email))
	var r apiResp
	if err := c.http.DoJSON(ctx, "GET", u, httpx.Auth{Bearer: c.token}, nil, &r); err != nil {
		return "", err
	}
	if !r.OK {
		return "", fmt.Errorf("slack lookupByEmail: %s", r.Error)
	}
	return r.User.ID, nil
}

// PostMessage posts blocks to a channel. threadTS empty posts a top-level
// message; otherwise it replies in that thread. Returns the new message ts.
func (c *Client) PostMessage(ctx context.Context, channel, threadTS string, blocks any, fallback string) (string, error) {
	if c.token == "" {
		return "", fmt.Errorf("slack: not configured")
	}
	body := map[string]any{"channel": channel, "text": fallback, "blocks": blocks}
	if threadTS != "" {
		body["thread_ts"] = threadTS
	}
	var r apiResp
	if err := c.http.DoJSON(ctx, "POST", c.baseURL+"/chat.postMessage", httpx.Auth{Bearer: c.token}, body, &r); err != nil {
		return "", err
	}
	if !r.OK {
		return "", fmt.Errorf("slack chat.postMessage: %s", r.Error)
	}
	return r.TS, nil
}

// OpenView opens a modal (e.g. the silence form) for a trigger_id.
func (c *Client) OpenView(ctx context.Context, triggerID string, view any) error {
	if c.token == "" {
		return fmt.Errorf("slack: not configured")
	}
	body := map[string]any{"trigger_id": triggerID, "view": view}
	var r apiResp
	if err := c.http.DoJSON(ctx, "POST", c.baseURL+"/views.open", httpx.Auth{Bearer: c.token}, body, &r); err != nil {
		return err
	}
	if !r.OK {
		return fmt.Errorf("slack views.open: %s", r.Error)
	}
	return nil
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
