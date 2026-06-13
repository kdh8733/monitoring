// Package notifier formats and sends the Slack message and its thread.
//
// Main thread (one per issue): cluster name, app/service name (k8s), and
// deploy repo name + a short issue summary.
//
// Thread replies:
//   - Silence button -> opens a Block Kit modal (duration, comment) that
//     submits to /slack/interactivity and calls the Grafana Silences API.
//   - Panel + log-source links (Kibana or Prometheus) and the alert rule.
//   - @mention of the deployer (committer of the ArgoCD-synced revision),
//     matched to Slack via users.lookupByEmail.
//   - Optional (feature-flagged) ArgoCD rollback button.
package notifier
