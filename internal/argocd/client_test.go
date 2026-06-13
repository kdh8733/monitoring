package argocd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAppInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"spec":{"source":{"repoURL":"https://github.com/acme/checkout.git"}},
			"status":{
				"sync":{"revision":"newsha"},
				"history":[{"id":1,"revision":"oldsha"},{"id":2,"revision":"newsha"}]
			}
		}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "atok")
	info, err := c.AppInfo(context.Background(), "checkout-api")
	if err != nil {
		t.Fatal(err)
	}
	if info.RepoURL != "https://github.com/acme/checkout.git" {
		t.Errorf("repoURL=%q", info.RepoURL)
	}
	if info.Revision != "newsha" {
		t.Errorf("revision=%q", info.Revision)
	}
	if info.PrevRevision != "oldsha" {
		t.Errorf("prevRevision=%q", info.PrevRevision)
	}
}

func TestSync(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || !strings.HasSuffix(r.URL.Path, "/sync") {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "atok")
	if err := c.Sync(context.Background(), "checkout-api", "oldsha"); err != nil {
		t.Fatal(err)
	}
	if gotBody["revision"] != "oldsha" {
		t.Errorf("sync body revision=%v", gotBody["revision"])
	}
}
