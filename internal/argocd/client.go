// Package argocd reads app deploy provenance and (when enabled) triggers a
// rollback sync, via the ArgoCD REST API using {ArgoCDBaseURL, ArgoCDToken}.
package argocd

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/kdh8733/monitoring/internal/httpx"
	"github.com/kdh8733/monitoring/internal/model"
)

type Client struct {
	baseURL string
	token   string
	http    *httpx.Client
}

func New(baseURL, token string) *Client {
	return &Client{baseURL: trimSlash(baseURL), token: token, http: httpx.New(10*time.Second, false)}
}

type application struct {
	Spec struct {
		Source struct {
			RepoURL string `json:"repoURL"`
		} `json:"source"`
	} `json:"spec"`
	Status struct {
		Sync struct {
			Revision string `json:"revision"`
		} `json:"sync"`
		History []struct {
			ID       int    `json:"id"`
			Revision string `json:"revision"`
		} `json:"history"`
	} `json:"status"`
}

// AppInfo returns the repo URL, current synced revision, and the previous
// revision (for rollback) of an ArgoCD application.
func (c *Client) AppInfo(ctx context.Context, app string) (model.ArgoInfo, error) {
	if c.baseURL == "" {
		return model.ArgoInfo{}, fmt.Errorf("argocd: not configured")
	}
	u := fmt.Sprintf("%s/api/v1/applications/%s", c.baseURL, url.PathEscape(app))
	var a application
	if err := c.http.DoJSON(ctx, "GET", u, httpx.Auth{Bearer: c.token}, nil, &a); err != nil {
		return model.ArgoInfo{}, err
	}

	info := model.ArgoInfo{
		RepoURL:  a.Spec.Source.RepoURL,
		Revision: a.Status.Sync.Revision,
	}
	if h := a.Status.History; len(h) >= 2 {
		sort.Slice(h, func(i, j int) bool { return h[i].ID < h[j].ID })
		// last is current, the one before it is the rollback target.
		info.PrevRevision = h[len(h)-2].Revision
	}
	return info, nil
}

// Sync triggers a sync of app to a specific revision (used for rollback).
func (c *Client) Sync(ctx context.Context, app, revision string) error {
	if c.baseURL == "" {
		return fmt.Errorf("argocd: not configured")
	}
	u := fmt.Sprintf("%s/api/v1/applications/%s/sync", c.baseURL, url.PathEscape(app))
	body := map[string]any{"revision": revision, "prune": false}
	return c.http.DoJSON(ctx, "POST", u, httpx.Auth{Bearer: c.token}, body, nil)
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
