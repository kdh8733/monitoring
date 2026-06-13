// Package webhook receives and validates inbound HTTP payloads.
//
//   - Grafana Alerting contact-point webhook: parse the alert (labels:
//     cluster, namespace, app/service; annotations; alert rule UID/name;
//     panel/dashboard URLs) into a normalized internal Alert.
//   - Slack interactivity: verify the Slack signing secret, then route
//     Block Kit actions (silence modal submit, rollback button) to handlers.
package webhook
