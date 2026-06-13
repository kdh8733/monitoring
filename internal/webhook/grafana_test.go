package webhook

import "testing"

const sampleGrafanaPayload = `{
  "receiver": "monitoring-orchestrator",
  "status": "firing",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "HighErrorRate",
        "__alert_rule_uid__": "abc123",
        "cluster": "prod-seoul",
        "namespace": "payments",
        "app": "checkout-api",
        "service": "checkout"
      },
      "annotations": { "summary": "5xx ratio > 5%" },
      "startsAt": "2026-06-13T10:00:00Z",
      "generatorURL": "https://grafana.local/alerting/grafana/abc123/view",
      "panelURL": "https://grafana.local/d/dash/panel?viewPanel=2",
      "silenceURL": "https://grafana.local/alerting/silence/new",
      "fingerprint": "fp-001"
    }
  ]
}`

func TestParseGrafanaWebhook(t *testing.T) {
	alerts, err := ParseGrafanaWebhook([]byte(sampleGrafanaPayload))
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 {
		t.Fatalf("want 1 alert, got %d", len(alerts))
	}
	a := alerts[0]
	checks := map[string][2]string{
		"RuleName":  {a.RuleName, "HighErrorRate"},
		"RuleUID":   {a.RuleUID, "abc123"},
		"Cluster":   {a.Cluster, "prod-seoul"},
		"Namespace": {a.Namespace, "payments"},
		"App":       {a.App, "checkout-api"}, // "app" wins over "service"
		"Summary":   {a.Summary, "5xx ratio > 5%"},
		"PanelURL":  {a.PanelURL, "https://grafana.local/d/dash/panel?viewPanel=2"},
		"Status":    {a.Status, "firing"},
		"Finger":    {a.Fingerprint, "fp-001"},
	}
	for name, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s = %q, want %q", name, c[0], c[1])
		}
	}
	if a.StartsAt.IsZero() {
		t.Error("StartsAt not parsed")
	}
}

func TestParseGrafanaWebhook_DashboardURLFallback(t *testing.T) {
	body := `{"alerts":[{"status":"firing","labels":{"alertname":"X"},"dashboardURL":"https://g/d/abc"}]}`
	alerts, err := ParseGrafanaWebhook([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if alerts[0].PanelURL != "https://g/d/abc" {
		t.Errorf("expected dashboardURL fallback, got %q", alerts[0].PanelURL)
	}
}
