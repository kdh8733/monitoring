// Package model holds the types shared across the alert pipeline so the
// integration packages (webhook, enricher, notifier) avoid import cycles.
package model

import "time"

// Alert is a normalized alert extracted from a Grafana Alerting webhook.
// Only fields the orchestrator acts on are kept.
type Alert struct {
	Fingerprint  string // stable id from Grafana (dedupe / thread key)
	Status       string // "firing" | "resolved"
	RuleName     string // alertname
	RuleUID      string // grafana rule uid, when present
	Cluster      string // from the `cluster` label (multi-cluster identity)
	Namespace    string // k8s namespace
	App          string // k8s app/service name
	Summary      string // annotations.summary or description
	PanelURL     string // dashboard/panel deep link
	GeneratorURL string // grafana rule/explore link
	SilenceURL   string // grafana-provided silence link (fallback)
	StartsAt     time.Time
}

// EnrichedAlert is an Alert plus everything gathered live at alert time.
// Any field may be empty when a source is unconfigured or lookup fails;
// the pipeline degrades gracefully rather than dropping the alert.
type EnrichedAlert struct {
	Alert

	// kubernetes
	PodImage   string
	DeployedAt time.Time

	// argocd
	RepoURL  string
	RepoName string
	Revision string // synced git SHA actually running

	// github (committer of the synced revision)
	CommitterName  string
	CommitterEmail string

	// slack identity
	SlackUserID string // "" when no match -> notifier falls back to name text

	// links resolved for the thread
	LogURL string // kibana or prometheus
}

// PodInfo is the subset of a Kubernetes pod the enricher needs.
type PodInfo struct {
	Image     string
	StartedAt time.Time
}

// ArgoInfo is the deploy provenance for an app from ArgoCD.
type ArgoInfo struct {
	RepoURL  string
	Revision string // synced git SHA actually running
	// PrevRevision is the revision before the current one (for rollback).
	PrevRevision string
}

// CommitInfo identifies the author of a git revision.
type CommitInfo struct {
	Name  string
	Email string
}
