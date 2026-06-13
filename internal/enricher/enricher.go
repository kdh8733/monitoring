package enricher

import (
	"context"

	"github.com/kdh8733/monitoring/pkg/github"
	"github.com/kdh8733/monitoring/pkg/model"
)

// Source interfaces decouple the enricher from concrete clients so it can be
// unit-tested with fakes. Any source may be nil (integration unconfigured).
type (
	KubeSource interface {
		PodInfo(ctx context.Context, cluster, namespace, app string) (model.PodInfo, error)
	}
	ArgoSource interface {
		AppInfo(ctx context.Context, app string) (model.ArgoInfo, error)
	}
	GitHubSource interface {
		Commit(ctx context.Context, repoURL, sha string) (model.CommitInfo, error)
	}
	SlackIdentity interface {
		UserIDByEmail(ctx context.Context, email string) (string, error)
	}
)

// Enricher gathers live context onto an alert. Each source failure degrades
// the result gracefully (field left empty) rather than dropping the alert.
type Enricher struct {
	Kube   KubeSource
	Argo   ArgoSource
	GitHub GitHubSource
	Slack  SlackIdentity

	// LogURL builds the log-source deep link (kibana/prometheus) for an alert.
	LogURL func(model.Alert) string
	// Logf, when set, records non-fatal enrichment failures.
	Logf func(format string, args ...any)
}

func (e *Enricher) logf(format string, args ...any) {
	if e.Logf != nil {
		e.Logf(format, args...)
	}
}

// Enrich returns the alert augmented with everything available right now.
func (e *Enricher) Enrich(ctx context.Context, a model.Alert) model.EnrichedAlert {
	out := model.EnrichedAlert{Alert: a}

	if e.Kube != nil && a.Namespace != "" && a.App != "" {
		if pi, err := e.Kube.PodInfo(ctx, a.Cluster, a.Namespace, a.App); err != nil {
			e.logf("enrich kube %s/%s: %v", a.Namespace, a.App, err)
		} else {
			out.PodImage, out.DeployedAt = pi.Image, pi.StartedAt
		}
	}

	if e.Argo != nil && a.App != "" {
		if ai, err := e.Argo.AppInfo(ctx, a.App); err != nil {
			e.logf("enrich argocd %s: %v", a.App, err)
		} else {
			out.RepoURL, out.Revision = ai.RepoURL, ai.Revision
			if owner, repo, perr := github.ParseRepo(ai.RepoURL); perr == nil {
				out.RepoName = owner + "/" + repo
			}
		}
	}

	if e.GitHub != nil && out.RepoURL != "" && out.Revision != "" {
		if ci, err := e.GitHub.Commit(ctx, out.RepoURL, out.Revision); err != nil {
			e.logf("enrich github %s@%s: %v", out.RepoURL, out.Revision, err)
		} else {
			out.CommitterName, out.CommitterEmail = ci.Name, ci.Email
		}
	}

	if e.Slack != nil && out.CommitterEmail != "" {
		if uid, err := e.Slack.UserIDByEmail(ctx, out.CommitterEmail); err != nil {
			e.logf("enrich slack id %s: %v", out.CommitterEmail, err)
		} else {
			out.SlackUserID = uid
		}
	}

	if e.LogURL != nil {
		out.LogURL = e.LogURL(a)
	}
	return out
}
