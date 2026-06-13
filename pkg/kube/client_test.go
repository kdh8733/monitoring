package kube

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPodInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/v1/namespaces/payments/pods") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer ktok" {
			t.Errorf("missing bearer, got %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"items":[{"spec":{"containers":[{"image":"reg/checkout:abc123"}]},"status":{"startTime":"2026-06-13T09:00:00Z"}}]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "ktok", false)
	pi, err := c.PodInfo(context.Background(), "prod", "payments", "checkout-api")
	if err != nil {
		t.Fatal(err)
	}
	if pi.Image != "reg/checkout:abc123" {
		t.Errorf("image=%q", pi.Image)
	}
	if pi.StartedAt.IsZero() {
		t.Error("startedAt not parsed")
	}
}

func TestPodInfo_NotConfigured(t *testing.T) {
	c := New("", "", false)
	if _, err := c.PodInfo(context.Background(), "", "ns", "app"); err == nil {
		t.Fatal("expected not-configured error")
	}
}

func TestPodInfo_NoPods(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer srv.Close()
	c := New(srv.URL, "ktok", false)
	if _, err := c.PodInfo(context.Background(), "", "ns", "app"); err == nil {
		t.Fatal("expected error for no pods")
	}
}
