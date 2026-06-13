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
