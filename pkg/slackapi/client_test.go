package slackapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUserIDByEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "users.lookupByEmail") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"ok":true,"user":{"id":"U123"}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "xoxb-test")
	id, err := c.UserIDByEmail(context.Background(), "kim@acme.io")
	if err != nil {
		t.Fatal(err)
	}
	if id != "U123" {
		t.Errorf("id=%q", id)
	}
}

func TestUserIDByEmail_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error":"users_not_found"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "xoxb-test")
	if _, err := c.UserIDByEmail(context.Background(), "ghost@acme.io"); err == nil {
		t.Fatal("expected error for ok:false")
	}
}

func TestPostMessage_ThreadAndBlocks(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		_, _ = w.Write([]byte(`{"ok":true,"ts":"1700.001"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "xoxb-test")
	ts, err := c.PostMessage(context.Background(), "C1", "1699.000", []map[string]any{{"type": "section"}}, "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if ts != "1700.001" {
		t.Errorf("ts=%q", ts)
	}
	if body["channel"] != "C1" || body["thread_ts"] != "1699.000" {
		t.Errorf("body channel/thread wrong: %v", body)
	}
}
