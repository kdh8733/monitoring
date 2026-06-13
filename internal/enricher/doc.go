// Package enricher gathers context at alert time and attaches it to the alert.
// No Prometheus annotations are relied upon; everything is queried live.
//
//	kubernetes.go - client-go: pod image tag, pod.status.startTime, cluster label.
//	argocd.go     - ArgoCD Application API: app -> deploy repo URL and the exact
//	                synced revision (git SHA). Also exposes a (feature-flagged)
//	                rollback/re-sync trigger gated by ARGOCD_ROLLBACK_ENABLED.
//	github.go     - synced revision SHA -> GET /commits/{sha}: committer name + email.
//	slack.go      - committer email -> users.lookupByEmail -> Slack user ID for @mention.
//
// Attribution uses the ArgoCD-synced revision (what is actually deployed),
// not the repo HEAD's last commit, so the tagged person is the deployer of
// the running version.
package enricher
