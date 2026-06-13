// Package github resolves the author of a deployed git revision via the
// GitHub REST API, using {GitHubBaseURL, GitHubToken}.
package github

import (
	"context"
	"fmt"
	"strings"
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

type commitResp struct {
	Commit struct {
		Author    gitIdent `json:"author"`
		Committer gitIdent `json:"committer"`
	} `json:"commit"`
}

type gitIdent struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Commit returns the author identity of revision sha in the given repo.
// repoURL accepts https or git@ forms; the owner/repo is parsed out.
func (c *Client) Commit(ctx context.Context, repoURL, sha string) (model.CommitInfo, error) {
	if c.token == "" {
		return model.CommitInfo{}, fmt.Errorf("github: not configured")
	}
	owner, repo, err := ParseRepo(repoURL)
	if err != nil {
		return model.CommitInfo{}, err
	}
	u := fmt.Sprintf("%s/repos/%s/%s/commits/%s", c.baseURL, owner, repo, sha)

	var cr commitResp
	if err := c.http.DoJSON(ctx, "GET", u, httpx.Auth{Bearer: c.token}, nil, &cr); err != nil {
		return model.CommitInfo{}, err
	}
	id := cr.Commit.Author
	if id.Email == "" { // fall back to committer when author is empty
		id = cr.Commit.Committer
	}
	return model.CommitInfo{Name: id.Name, Email: id.Email}, nil
}

// ParseRepo extracts owner and repo from common GitHub remote URL forms:
//
//	https://github.com/owner/repo(.git)
//	git@github.com:owner/repo(.git)
//	ssh://git@github.com/owner/repo(.git)
func ParseRepo(repoURL string) (owner, repo string, err error) {
	s := strings.TrimSpace(repoURL)
	s = strings.TrimSuffix(s, ".git")
	if i := strings.Index(s, "github.com"); i >= 0 {
		s = s[i+len("github.com"):]
	}
	s = strings.TrimLeft(s, ":/")
	parts := strings.Split(s, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("github: cannot parse repo from %q", repoURL)
	}
	return parts[0], parts[1], nil
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
