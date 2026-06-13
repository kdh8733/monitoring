// Package config loads runtime configuration and credentials from the
// environment (.env / env vars): central Grafana base URL + API token,
// kubeconfig/in-cluster access, GitHub token, ArgoCD server + token,
// Slack bot/signing tokens, and feature flags (e.g. ARGOCD_ROLLBACK_ENABLED).
package config
