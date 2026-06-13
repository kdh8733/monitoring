package webhook

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kdh8733/monitoring/pkg/model"
)

// grafanaPayload is the subset of a Grafana Alerting contact-point webhook
// that the orchestrator consumes.
type grafanaPayload struct {
	Alerts []struct {
		Status       string            `json:"status"`
		Labels       map[string]string `json:"labels"`
		Annotations  map[string]string `json:"annotations"`
		StartsAt     string            `json:"startsAt"`
		GeneratorURL string            `json:"generatorURL"`
		Fingerprint  string            `json:"fingerprint"`
		SilenceURL   string            `json:"silenceURL"`
		DashboardURL string            `json:"dashboardURL"`
		PanelURL     string            `json:"panelURL"`
	} `json:"alerts"`
}

// label keys checked, in priority order, for the app/service name.
var appLabelKeys = []string{"app", "service", "app_kubernetes_io_name", "deployment", "job"}

// ParseGrafanaWebhook decodes a Grafana Alerting webhook body into normalized
// alerts. It does not fail on missing optional labels; those map to "".
func ParseGrafanaWebhook(body []byte) ([]model.Alert, error) {
	var p grafanaPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("webhook: decode grafana payload: %w", err)
	}

	out := make([]model.Alert, 0, len(p.Alerts))
	for _, a := range p.Alerts {
		panel := a.PanelURL
		if panel == "" {
			panel = a.DashboardURL
		}
		out = append(out, model.Alert{
			Fingerprint:  a.Fingerprint,
			Status:       a.Status,
			RuleName:     a.Labels["alertname"],
			RuleUID:      a.Labels["__alert_rule_uid__"],
			Cluster:      a.Labels["cluster"],
			Namespace:    a.Labels["namespace"],
			App:          pick(a.Labels, appLabelKeys),
			Summary:      annotation(a.Annotations),
			PanelURL:     panel,
			GeneratorURL: a.GeneratorURL,
			SilenceURL:   a.SilenceURL,
			StartsAt:     parseTime(a.StartsAt),
		})
	}
	return out, nil
}

func pick(m map[string]string, keys []string) string {
	for _, k := range keys {
		if v := m[k]; v != "" {
			return v
		}
	}
	return ""
}

func annotation(m map[string]string) string {
	if v := m["summary"]; v != "" {
		return v
	}
	return m["description"]
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
