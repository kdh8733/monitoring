package enricher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kdh8733/monitoring/pkg/model"
)

type fakeKube struct {
	pi  model.PodInfo
	err error
}

func (f fakeKube) PodInfo(_ context.Context, _, _, _ string) (model.PodInfo, error) {
	return f.pi, f.err
}

type fakeArgo struct {
	ai  model.ArgoInfo
	err error
}

func (f fakeArgo) AppInfo(_ context.Context, _ string) (model.ArgoInfo, error) {
	return f.ai, f.err
}

type fakeGitHub struct {
	ci  model.CommitInfo
	err error
}

func (f fakeGitHub) Commit(_ context.Context, _, _ string) (model.CommitInfo, error) {
	return f.ci, f.err
}

// fakeGitHubFn lets a test observe the (repoURL, sha) the enricher queries.
type fakeGitHubFn func(repoURL, sha string) (model.CommitInfo, error)

func (f fakeGitHubFn) Commit(_ context.Context, repoURL, sha string) (model.CommitInfo, error) {
	return f(repoURL, sha)
}

type fakeSlack struct {
	id  string
	err error
}

func (f fakeSlack) UserIDByEmail(_ context.Context, _ string) (string, error) {
	return f.id, f.err
}

func baseAlert() model.Alert {
	return model.Alert{RuleName: "HighErrorRate", Cluster: "prod", Namespace: "payments", App: "checkout-api"}
}

func TestEnrich_FullPipeline(t *testing.T) {
	e := &Enricher{
		Kube:   fakeKube{pi: model.PodInfo{Image: "reg/checkout:abc", StartedAt: time.Unix(1, 0)}},
		Argo:   fakeArgo{ai: model.ArgoInfo{RepoURL: "https://github.com/acme/checkout.git", Revision: "deadbeef"}},
		GitHub: fakeGitHub{ci: model.CommitInfo{Name: "Kim", Email: "kim@acme.io"}},
		Slack:  fakeSlack{id: "U123"},
		LogURL: func(model.Alert) string { return "https://kibana/x" },
	}
	out := e.Enrich(context.Background(), baseAlert())

	if out.PodImage != "reg/checkout:abc" {
		t.Errorf("PodImage=%q", out.PodImage)
	}
	if out.RepoName != "acme/checkout" {
		t.Errorf("RepoName=%q", out.RepoName)
	}
	if out.Revision != "deadbeef" {
		t.Errorf("Revision=%q", out.Revision)
	}
	if out.CommitterEmail != "kim@acme.io" {
		t.Errorf("CommitterEmail=%q", out.CommitterEmail)
	}
	if out.SlackUserID != "U123" {
		t.Errorf("SlackUserID=%q", out.SlackUserID)
	}
	if out.LogURL != "https://kibana/x" {
		t.Errorf("LogURL=%q", out.LogURL)
	}
}

func TestEnrich_GracefulDegrade(t *testing.T) {
	// Slack lookup fails -> SlackUserID empty but committer still attributed,
	// and the alert is never dropped.
	e := &Enricher{
		Argo:   fakeArgo{ai: model.ArgoInfo{RepoURL: "https://github.com/acme/checkout", Revision: "sha"}},
		GitHub: fakeGitHub{ci: model.CommitInfo{Name: "Kim", Email: "kim@acme.io"}},
		Slack:  fakeSlack{err: errors.New("users_not_found")},
	}
	out := e.Enrich(context.Background(), baseAlert())
	if out.SlackUserID != "" {
		t.Errorf("expected empty SlackUserID on lookup failure, got %q", out.SlackUserID)
	}
	if out.CommitterName != "Kim" {
		t.Errorf("committer should still be attributed, got %q", out.CommitterName)
	}
	if out.RuleName != "HighErrorRate" {
		t.Error("base alert fields must survive")
	}
}

func TestEnrich_ImageTagFallbackWithoutArgo(t *testing.T) {
	// No ArgoCD source: committer must still be attributed from the commit
	// hash embedded in the running pod image tag (original-design behavior).
	var gotRepo, gotSha string
	e := &Enricher{
		Kube: fakeKube{pi: model.PodInfo{Image: "ghcr.io/acme/checkout:main-1a2b3c4"}},
		GitHub: fakeGitHubFn(func(repoURL, sha string) (model.CommitInfo, error) {
			gotRepo, gotSha = repoURL, sha
			return model.CommitInfo{Name: "Kim", Email: "kim@acme.io"}, nil
		}),
		Slack: fakeSlack{id: "U123"},
	}
	out := e.Enrich(context.Background(), baseAlert())

	if gotRepo != "acme/checkout" || gotSha != "1a2b3c4" {
		t.Errorf("github queried with %q@%q, want acme/checkout@1a2b3c4", gotRepo, gotSha)
	}
	if out.RepoName != "acme/checkout" {
		t.Errorf("RepoName=%q, want acme/checkout", out.RepoName)
	}
	if out.Revision != "1a2b3c4" {
		t.Errorf("Revision=%q, want 1a2b3c4", out.Revision)
	}
	if out.SlackUserID != "U123" {
		t.Errorf("SlackUserID=%q, want U123", out.SlackUserID)
	}
}

func TestEnrich_ArgoTakesPrecedenceOverImage(t *testing.T) {
	// When ArgoCD provides a revision, the image tag fallback is not used.
	var gotSha string
	e := &Enricher{
		Kube: fakeKube{pi: model.PodInfo{Image: "ghcr.io/acme/checkout:main-1a2b3c4"}},
		Argo: fakeArgo{ai: model.ArgoInfo{RepoURL: "https://github.com/acme/checkout", Revision: "deadbeef"}},
		GitHub: fakeGitHubFn(func(_, sha string) (model.CommitInfo, error) {
			gotSha = sha
			return model.CommitInfo{Name: "Kim", Email: "kim@acme.io"}, nil
		}),
	}
	out := e.Enrich(context.Background(), baseAlert())
	if gotSha != "deadbeef" {
		t.Errorf("github queried with sha %q, want argocd revision deadbeef", gotSha)
	}
	if out.Revision != "deadbeef" {
		t.Errorf("Revision=%q, want deadbeef", out.Revision)
	}
}

func TestEnrich_NilSourcesNoPanic(t *testing.T) {
	e := &Enricher{} // everything nil
	out := e.Enrich(context.Background(), baseAlert())
	if out.App != "checkout-api" {
		t.Error("should pass through with no sources")
	}
}
