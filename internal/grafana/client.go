package grafana

import (
	"context"
	"fmt"
	"time"

	"github.com/kdh8733/monitoring/internal/httpx"
)

// Client talks to the Grafana-managed Alertmanager Silences API using
// {GrafanaBaseURL, GrafanaAPIToken}.
type Client struct {
	baseURL string
	token   string
	now     func() time.Time
	http    *httpx.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		baseURL: trimSlash(baseURL),
		token:   token,
		now:     time.Now,
		http:    httpx.New(10*time.Second, false),
	}
}

// Matcher is an Alertmanager silence matcher.
type Matcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsRegex bool   `json:"isRegex"`
	IsEqual bool   `json:"isEqual"`
}

type silencePayload struct {
	Matchers  []Matcher `json:"matchers"`
	StartsAt  string    `json:"startsAt"`
	EndsAt    string    `json:"endsAt"`
	CreatedBy string    `json:"createdBy"`
	Comment   string    `json:"comment"`
}

const silencesPath = "/api/alertmanager/grafana/api/v2/silences"

// CreateSilence creates a silence for the given matchers lasting `dur`,
// returning the new silence ID.
func (c *Client) CreateSilence(ctx context.Context, matchers []Matcher, dur time.Duration, createdBy, comment string) (string, error) {
	if c.baseURL == "" {
		return "", fmt.Errorf("grafana: not configured")
	}
	start := c.now().UTC()
	payload := silencePayload{
		Matchers:  matchers,
		StartsAt:  start.Format(time.RFC3339),
		EndsAt:    start.Add(dur).Format(time.RFC3339),
		CreatedBy: createdBy,
		Comment:   comment,
	}
	var resp struct {
		SilenceID string `json:"silenceID"`
	}
	if err := c.http.DoJSON(ctx, "POST", c.baseURL+silencesPath, httpx.Auth{Bearer: c.token}, payload, &resp); err != nil {
		return "", err
	}
	return resp.SilenceID, nil
}

// GetSilence reports whether a silence with id exists (used to verify creation).
func (c *Client) GetSilence(ctx context.Context, id string) (bool, error) {
	if c.baseURL == "" {
		return false, fmt.Errorf("grafana: not configured")
	}
	u := fmt.Sprintf("%s/api/alertmanager/grafana/api/v2/silence/%s", c.baseURL, id)
	var resp struct {
		ID string `json:"id"`
	}
	if err := c.http.DoJSON(ctx, "GET", u, httpx.Auth{Bearer: c.token}, nil, &resp); err != nil {
		return false, err
	}
	return resp.ID != "", nil
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
