// Package grafana talks to the central Grafana instance.
//
//   - Build panel/dashboard deep links and log-source links (Kibana or
//     Prometheus/Grafana Explore) for the alert's rule and time range.
//   - Resolve the alert rule definition (UID -> rule expr/name).
//   - Create/expire silences via the Grafana-managed Alertmanager Silences
//     API (POST /api/alertmanager/grafana/api/v2/silences), driven by the
//     Slack silence modal.
package grafana
