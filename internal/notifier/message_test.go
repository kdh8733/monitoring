package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kdh8733/monitoring/pkg/model"
)

// dump serializes blocks without HTML-escaping so assertions can match the
// logical mrkdwn (e.g. "<@U123>"). Slack receives the escaped JSON and parses
// it back to the same characters, so this matches what Slack sees.
func dump(t *testing.T, v any) string {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func enriched() model.EnrichedAlert {
	return model.EnrichedAlert{
		Alert: model.Alert{
			Status: "firing", RuleName: "HighErrorRate", Cluster: "prod-seoul",
			Namespace: "payments", App: "checkout-api", Summary: "5xx > 5%",
			PanelURL: "https://grafana/d/p", GeneratorURL: "https://grafana/rule",
		},
		RepoName: "acme/checkout", Revision: "deadbeefcafe",
		CommitterName: "Kim", SlackUserID: "U123", LogURL: "https://kibana/x",
	}
}

func TestBuildMainBlocks_HasIdentity(t *testing.T) {
	s := dump(t, BuildMainBlocks(enriched()))
	for _, want := range []string{"prod-seoul", "checkout-api", "acme/checkout", "HighErrorRate"} {
		if !strings.Contains(s, want) {
			t.Errorf("main blocks missing %q in %s", want, s)
		}
	}
}

func TestBuildContextBlocks_LinksAndMention(t *testing.T) {
	s := dump(t, BuildContextBlocks(enriched()))
	for _, want := range []string{"https://grafana/d/p", "https://kibana/x", "HighErrorRate", "<@U123>"} {
		if !strings.Contains(s, want) {
			t.Errorf("context blocks missing %q in %s", want, s)
		}
	}
}

func TestOwnerMention_FallbackWhenNoSlackID(t *testing.T) {
	a := enriched()
	a.SlackUserID = ""
	s := dump(t, BuildContextBlocks(a))
	if !strings.Contains(s, "Kim") || !strings.Contains(s, "매칭 실패") {
		t.Errorf("expected committer-name fallback, got %s", s)
	}
}

func TestBuildActionBlocks_RollbackGatedByFlag(t *testing.T) {
	off := dump(t, BuildActionBlocks(enriched(), false))
	if !strings.Contains(off, ActionSilenceOpen) {
		t.Error("silence button must always be present")
	}
	if strings.Contains(off, ActionRollbackTrigger) {
		t.Error("rollback button must be absent when flag is off")
	}

	on := dump(t, BuildActionBlocks(enriched(), true))
	if !strings.Contains(on, ActionRollbackTrigger) {
		t.Error("rollback button must be present when flag is on")
	}
}

type fakePoster struct {
	calls []struct{ thread string }
}

func (f *fakePoster) PostMessage(_ context.Context, _, threadTS string, _ any, _ string) (string, error) {
	f.calls = append(f.calls, struct{ thread string }{threadTS})
	return "1700.000", nil
}

func TestNotify_PostsMainThenThreadReplies(t *testing.T) {
	fp := &fakePoster{}
	n := &Notifier{Slack: fp, Channel: "C1", RollbackEnabled: true}
	ts, err := n.Notify(context.Background(), enriched())
	if err != nil {
		t.Fatal(err)
	}
	if ts != "1700.000" {
		t.Errorf("ts=%q", ts)
	}
	if len(fp.calls) != 3 {
		t.Fatalf("expected 3 posts (main + 2 replies), got %d", len(fp.calls))
	}
	if fp.calls[0].thread != "" {
		t.Error("first post must be top-level (no thread_ts)")
	}
	if fp.calls[1].thread != "1700.000" || fp.calls[2].thread != "1700.000" {
		t.Error("replies must target the main message thread")
	}
}
