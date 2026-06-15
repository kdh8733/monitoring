package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseRepo(t *testing.T) {
	cases := map[string][2]string{
		"https://github.com/acme/checkout.git": {"acme", "checkout"},
		"https://github.com/acme/checkout":     {"acme", "checkout"},
		"git@github.com:acme/checkout.git":     {"acme", "checkout"},
		"ssh://git@github.com/acme/checkout":   {"acme", "checkout"},
	}
	for in, want := range cases {
		o, r, err := ParseRepo(in)
		if err != nil {
			t.Errorf("%s: %v", in, err)
			continue
		}
		if o != want[0] || r != want[1] {
			t.Errorf("%s => %s/%s, want %s/%s", in, o, r, want[0], want[1])
		}
	}
	if _, _, err := ParseRepo("not-a-url"); err == nil {
		t.Error("expected parse error")
	}
}

func TestParseImage(t *testing.T) {
	type want struct {
		repo, sha string
		ok        bool
	}
	cases := map[string]want{
		"ghcr.io/acme/checkout:main-1a2b3c4":        {"acme/checkout", "1a2b3c4", true},
		"registry.io/acme/checkout:v1.4.0-g9f8e7d6": {"acme/checkout", "9f8e7d6", true},
		"reg.io:5000/acme/checkout:0badc0ffee":      {"acme/checkout", "0badc0ffee", true},
		"acme/checkout:1a2b3c4d5e6f":                {"acme/checkout", "1a2b3c4d5e6f", true},
		"ghcr.io/acme/checkout@sha256:deadbeef":     {"", "", false}, // digest pin, no tag
		"ghcr.io/acme/checkout:latest":              {"", "", false}, // no sha in tag
		"ghcr.io/acme/checkout:v1.4.0":              {"", "", false}, // semver only
		"checkout:1a2b3c4":                          {"", "", false}, // no owner
		"":                                          {"", "", false},
	}
	for in, w := range cases {
		repo, sha, ok := ParseImage(in)
		if ok != w.ok || repo != w.repo || sha != w.sha {
			t.Errorf("ParseImage(%q) = (%q,%q,%v), want (%q,%q,%v)",
				in, repo, sha, ok, w.repo, w.sha, w.ok)
		}
	}
}

func TestCommit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/repos/acme/checkout/commits/newsha") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"commit":{"author":{"name":"Kim","email":"kim@acme.io"},"committer":{"name":"GH","email":"noreply@github.com"}}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "ghtok")
	ci, err := c.Commit(context.Background(), "https://github.com/acme/checkout.git", "newsha")
	if err != nil {
		t.Fatal(err)
	}
	if ci.Name != "Kim" || ci.Email != "kim@acme.io" {
		t.Errorf("got %+v, want author Kim", ci)
	}
}

func TestCommit_NotConfigured(t *testing.T) {
	c := New("https://api.github.com", "")
	if _, err := c.Commit(context.Background(), "https://github.com/a/b", "sha"); err == nil {
		t.Fatal("expected not-configured error")
	}
}
