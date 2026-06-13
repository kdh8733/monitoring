// Package kube reads pod provenance from a Kubernetes API server over REST,
// using the central {KubeAPIURL, KubeToken} from config. The cluster argument
// is accepted for forward-compatibility (central multi-cluster), but in the
// central topology a single API endpoint serves all clusters distinguished by
// the alert's `cluster` label.
package kube

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/kdh8733/monitoring/internal/httpx"
	"github.com/kdh8733/monitoring/internal/model"
)

type Client struct {
	baseURL string
	token   string
	http    *httpx.Client
}

func New(baseURL, token string, insecure bool) *Client {
	return &Client{
		baseURL: trimSlash(baseURL),
		token:   token,
		http:    httpx.New(10*time.Second, insecure),
	}
}

type podList struct {
	Items []struct {
		Spec struct {
			Containers []struct {
				Image string `json:"image"`
			} `json:"containers"`
		} `json:"spec"`
		Status struct {
			StartTime time.Time `json:"startTime"`
		} `json:"status"`
	} `json:"items"`
}

// PodInfo returns the image and start time of the first pod matching
// app=<app> in the namespace. The cluster argument is currently unused in the
// central topology (see package doc).
func (c *Client) PodInfo(ctx context.Context, cluster, namespace, app string) (model.PodInfo, error) {
	if c.baseURL == "" {
		return model.PodInfo{}, fmt.Errorf("kube: not configured")
	}
	sel := url.QueryEscape("app=" + app)
	u := fmt.Sprintf("%s/api/v1/namespaces/%s/pods?labelSelector=%s&limit=1", c.baseURL, url.PathEscape(namespace), sel)

	var pl podList
	if err := c.http.DoJSON(ctx, "GET", u, httpx.Auth{Bearer: c.token}, nil, &pl); err != nil {
		return model.PodInfo{}, err
	}
	if len(pl.Items) == 0 || len(pl.Items[0].Spec.Containers) == 0 {
		return model.PodInfo{}, fmt.Errorf("kube: no pod for app=%s in %s", app, namespace)
	}
	p := pl.Items[0]
	return model.PodInfo{Image: p.Spec.Containers[0].Image, StartedAt: p.Status.StartTime}, nil
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
