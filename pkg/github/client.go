// Package github resolves the author of a deployed git revision via the
// GitHub REST API, using {GitHubBaseURL, GitHubToken}.
package github

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kdh8733/monitoring/pkg/httpx"
	"github.com/kdh8733/monitoring/pkg/model"
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

// ParseImage extracts a repo path ("owner/repo") and a commit SHA from a
// running pod image reference, for committer attribution WITHOUT ArgoCD/GitOps.
// It mirrors the original design's "image tag commit hash -> /commits/{sha}".
//
// It works when the image path encodes the GitHub owner/repo (common with
// ghcr.io and org-mirrored registries) and the tag embeds a git SHA, e.g.:
//
//	ghcr.io/acme/checkout:main-1a2b3c4        -> acme/checkout, 1a2b3c4
//	registry.io/acme/checkout:v1.4.0-g9f8e7d6 -> acme/checkout, 9f8e7d6
//	acme/checkout:0badc0ffee...               -> acme/checkout, 0badc0ffee...
//
// ok is false when no SHA can be recovered or the path lacks owner/repo.
func ParseImage(image string) (repo, sha string, ok bool) {
	s := strings.TrimSpace(image)
	if s == "" {
		return "", "", false
	}
	// Drop a digest pin (@sha256:...): it is not a git revision.
	if at := strings.IndexByte(s, '@'); at >= 0 {
		s = s[:at]
	}

	// Split path from tag. The tag is the part after the LAST ':' that follows
	// the last '/', so a registry host:port (reg.io:5000/...) is not mistaken
	// for a tag.
	path, tag := s, ""
	lastSlash := strings.LastIndexByte(s, '/')
	if c := strings.LastIndexByte(s, ':'); c > lastSlash {
		path, tag = s[:c], s[c+1:]
	}
	if tag == "" {
		return "", "", false
	}

	sha = extractSHA(tag)
	if sha == "" {
		return "", "", false
	}

	// owner/repo = the last two path segments (drops the registry host).
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", false
	}
	owner, name := parts[len(parts)-2], parts[len(parts)-1]
	if owner == "" || name == "" {
		return "", "", false
	}
	return owner + "/" + name, sha, true
}

// extractSHA returns the longest hex run (len 7..40) in a tag, which is the
// git commit hash convention. Returns "" when the tag carries no such token.
func extractSHA(tag string) string {
	best := ""
	i := 0
	for i < len(tag) {
		if !isHex(tag[i]) {
			i++
			continue
		}
		j := i
		for j < len(tag) && isHex(tag[j]) {
			j++
		}
		run := tag[i:j]
		if n := len(run); n >= 7 && n <= 40 && n >= len(best) {
			best = run
		}
		i = j
	}
	return best
}

func isHex(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
