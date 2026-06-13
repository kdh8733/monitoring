package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad_FileValuesAndDefaults(t *testing.T) {
	p := writeFile(t, `
# central config
GRAFANA_BASE_URL=https://grafana.local
GRAFANA_API_TOKEN="tok123"
ARGOCD_ROLLBACK_ENABLED=true
LOG_SOURCE=prometheus
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GrafanaBaseURL != "https://grafana.local" {
		t.Errorf("GrafanaBaseURL=%q", cfg.GrafanaBaseURL)
	}
	if cfg.GrafanaAPIToken != "tok123" { // quotes stripped
		t.Errorf("token=%q", cfg.GrafanaAPIToken)
	}
	if !cfg.ArgoCDRollbackEnabled {
		t.Error("rollback flag should be true")
	}
	if cfg.LogSource != "prometheus" {
		t.Errorf("LogSource=%q", cfg.LogSource)
	}
	if cfg.ListenAddr != ":8080" { // default applied
		t.Errorf("ListenAddr default=%q", cfg.ListenAddr)
	}
	if cfg.SlackBaseURL != "https://slack.com/api" {
		t.Errorf("SlackBaseURL default=%q", cfg.SlackBaseURL)
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	p := writeFile(t, "GRAFANA_BASE_URL=https://from-file\n")
	t.Setenv("GRAFANA_BASE_URL", "https://from-env")
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GrafanaBaseURL != "https://from-env" {
		t.Errorf("env should override file, got %q", cfg.GrafanaBaseURL)
	}
}

func TestLoad_MissingFileIsOK(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.env"))
	if err != nil {
		t.Fatalf("missing file must not error: %v", err)
	}
	if len(cfg.Missing()) == 0 {
		t.Error("expected required keys reported missing for blank config")
	}
}

func TestMissing_ReportsBlankRequired(t *testing.T) {
	p := writeFile(t, `
SLACK_BOT_TOKEN=x
SLACK_SIGNING_SECRET=x
SLACK_ALERT_CHANNEL=x
GRAFANA_BASE_URL=x
GRAFANA_API_TOKEN=x
KUBE_API_URL=x
KUBE_TOKEN=x
GITHUB_TOKEN=x
ARGOCD_BASE_URL=x
ARGOCD_TOKEN=x
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if m := cfg.Missing(); len(m) != 0 {
		t.Errorf("expected nothing missing, got %v", m)
	}
}
