package webhook

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kdh8733/monitoring/pkg/model"
)

// alertPayload is the common subset of a Grafana Alerting contact-point
// webhook AND a Prometheus AlertManager (v4) webhook. Both share alerts[]
// with labels/annotations/startsAt/generatorURL/fingerprint; the Grafana-only
// fields (silenceURL/dashboardURL/panelURL/imageURL) are simply absent for
// AlertManager and decode to "".
type alertPayload struct {
	// CommonLabels carries cluster identity that AlertManager puts at the top
	// level (external_labels) rather than only on each alert.
	CommonLabels map[string]string `json:"commonLabels"`
	Alerts       []struct {
		Status       string            `json:"status"`
		Labels       map[string]string `json:"labels"`
		Annotations  map[string]string `json:"annotations"`
		StartsAt     string            `json:"startsAt"`
		GeneratorURL string            `json:"generatorURL"`
		Fingerprint  string            `json:"fingerprint"`
		SilenceURL   string            `json:"silenceURL"`
		DashboardURL string            `json:"dashboardURL"`
		PanelURL     string            `json:"panelURL"`
		ImageURL     string            `json:"imageURL"`
	} `json:"alerts"`
}

// label keys checked, in priority order, for the app/service name.
var appLabelKeys = []string{"app", "service", "app_kubernetes_io_name", "deployment", "job"}

// ParseWebhook decodes either a Grafana Alerting or a Prometheus AlertManager
// webhook body into normalized alerts. The payload shapes overlap, so one
// parser serves both; missing optional fields map to "". Per-alert labels win,
// falling back to commonLabels (where AlertManager external_labels land).
func ParseWebhook(body []byte) ([]model.Alert, error) {
	var p alertPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, fmt.Errorf("webhook: decode payload: %w", err)
	}

	out := make([]model.Alert, 0, len(p.Alerts))
	for _, a := range p.Alerts {
		panel := a.PanelURL
		if panel == "" {
			panel = a.DashboardURL
		}
		label := func(key string) string {
			if v := a.Labels[key]; v != "" {
				return v
			}
			return p.CommonLabels[key]
		}
		out = append(out, model.Alert{
			Fingerprint:  a.Fingerprint,
			Status:       a.Status,
			RuleName:     label("alertname"),
			RuleUID:      a.Labels["__alert_rule_uid__"],
			Cluster:      label("cluster"),
			Namespace:    label("namespace"),
			App:          pickWithFallback(a.Labels, p.CommonLabels, appLabelKeys),
			Summary:      annotation(a.Annotations),
			PanelURL:     panel,
			GeneratorURL: a.GeneratorURL,
			SilenceURL:   a.SilenceURL,
			ImageURL:     a.ImageURL,
			StartsAt:     parseTime(a.StartsAt),
		})
	}
	return out, nil
}

// ParseGrafanaWebhook is retained as an alias for ParseWebhook (back-compat).
func ParseGrafanaWebhook(body []byte) ([]model.Alert, error) {
	return ParseWebhook(body)
}

// pickWithFallback returns the first non-empty value among keys, checking the
// per-alert labels first, then the common labels.
func pickWithFallback(labels, common map[string]string, keys []string) string {
	if v := pick(labels, keys); v != "" {
		return v
	}
	return pick(common, keys)
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
