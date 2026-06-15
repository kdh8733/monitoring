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

// AlertManager (v4) webhook: cluster comes from commonLabels (external_labels),
// and there is no panelURL/silenceURL. The same parser must handle it.
const sampleAlertManagerPayload = `{
  "version": "4",
  "status": "firing",
  "commonLabels": { "cluster": "prod-seoul" },
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "HighErrorRate",
        "namespace": "payments",
        "app": "checkout-api"
      },
      "annotations": { "description": "5xx ratio high" },
      "startsAt": "2026-06-13T10:00:00Z",
      "generatorURL": "https://prometheus.local/graph?g0.expr=...",
      "fingerprint": "fp-am-001"
    }
  ]
}`

func TestParseWebhook_AlertManager(t *testing.T) {
	alerts, err := ParseWebhook([]byte(sampleAlertManagerPayload))
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 {
		t.Fatalf("want 1 alert, got %d", len(alerts))
	}
	a := alerts[0]
	if a.Cluster != "prod-seoul" {
		t.Errorf("Cluster=%q, want prod-seoul (from commonLabels)", a.Cluster)
	}
	if a.App != "checkout-api" {
		t.Errorf("App=%q, want checkout-api", a.App)
	}
	if a.Summary != "5xx ratio high" {
		t.Errorf("Summary=%q, want description fallback", a.Summary)
	}
	if a.GeneratorURL == "" || a.Fingerprint != "fp-am-001" {
		t.Errorf("generatorURL/fingerprint not parsed: %+v", a)
	}
}

func TestParseWebhook_ImageURL(t *testing.T) {
	body := `{"alerts":[{"status":"firing","labels":{"alertname":"X"},"imageURL":"https://grafana/render/abc.png"}]}`
	alerts, err := ParseWebhook([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if alerts[0].ImageURL != "https://grafana/render/abc.png" {
		t.Errorf("ImageURL=%q", alerts[0].ImageURL)
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
