package grafana

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeGrafana stores a created silence so the test can verify it exists,
// mirroring completion criterion 2 (create then GET-confirm).
func TestCreateAndGetSilence(t *testing.T) {
	store := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/silences"):
			var payload map[string]any
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &payload)
			if _, ok := payload["matchers"]; !ok {
				t.Error("payload missing matchers")
			}
			store["sil-1"] = true
			_, _ = w.Write([]byte(`{"silenceID":"sil-1"}`))
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/silence/"):
			id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
			if store[id] {
				_, _ = w.Write([]byte(`{"id":"` + id + `"}`))
			} else {
				http.Error(w, "not found", http.StatusNotFound)
			}
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "gtok")
	c.now = func() time.Time { return time.Unix(1_700_000_000, 0).UTC() }

	matchers := []Matcher{{Name: "alertname", Value: "HighErrorRate", IsEqual: true}}
	id, err := c.CreateSilence(context.Background(), matchers, time.Hour, "slack:U1", "deploy issue")
	if err != nil {
		t.Fatal(err)
	}
	if id != "sil-1" {
		t.Fatalf("id=%q", id)
	}

	ok, err := c.GetSilence(context.Background(), id)
	if err != nil || !ok {
		t.Fatalf("silence should exist: ok=%v err=%v", ok, err)
	}
}
