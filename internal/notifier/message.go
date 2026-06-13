package notifier

import (
	"encoding/json"
	"fmt"

	"github.com/kdh8733/monitoring/pkg/model"
)

// Action IDs used on buttons and matched by the interactivity handler.
const (
	ActionSilenceOpen     = "silence_open"
	ActionRollbackTrigger = "rollback_trigger"
)

// ButtonValue is the JSON payload carried on action buttons so the
// interactivity handler knows what alert to act on.
type ButtonValue struct {
	App       string `json:"app"`
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
	RuleName  string `json:"rule_name"`
	RuleUID   string `json:"rule_uid"`
}

type block = map[string]any

// BuildMainBlocks renders the top-of-thread message: cluster, app/service,
// and repo (completion criterion 1), plus a short summary.
func BuildMainBlocks(a model.EnrichedAlert) []block {
	title := "🚨 " + orDash(a.RuleName)
	if a.Status == "resolved" {
		title = "✅ Resolved: " + orDash(a.RuleName)
	}

	fields := []block{
		mrkdwn("*Cluster:*\n" + orDash(a.Cluster)),
		mrkdwn("*App/Service:*\n" + orDash(a.App)),
		mrkdwn("*Namespace:*\n" + orDash(a.Namespace)),
		mrkdwn("*Repo:*\n" + orDash(a.RepoName)),
	}
	blocks := []block{
		{"type": "header", "text": map[string]any{"type": "plain_text", "text": truncate(title, 150), "emoji": true}},
		{"type": "section", "fields": fields},
	}
	if a.Summary != "" {
		blocks = append(blocks, section("*Summary:*\n"+a.Summary))
	}
	return blocks
}

// BuildContextBlocks renders the thread reply with panel link, log-source
// link, and alert rule (completion criterion 3), plus the deploy owner
// mention (completion criterion 4).
func BuildContextBlocks(a model.EnrichedAlert) []block {
	var blocks []block

	links := joinNonEmpty("  |  ",
		link("Panel", a.PanelURL),
		link("Logs", a.LogURL),
		link("Rule", a.GeneratorURL, orDash(a.RuleName)),
	)
	if links == "" {
		links = "*Rule:* " + orDash(a.RuleName)
	}
	blocks = append(blocks, section(links))

	if owner := ownerMention(a); owner != "" {
		blocks = append(blocks, section(owner))
	}
	return blocks
}

// BuildActionBlocks renders the action buttons. The rollback button is only
// present when rollbackEnabled is true (completion criterion 5).
func BuildActionBlocks(a model.EnrichedAlert, rollbackEnabled bool) []block {
	val, _ := json.Marshal(ButtonValue{
		App: a.App, Cluster: a.Cluster, Namespace: a.Namespace,
		RuleName: a.RuleName, RuleUID: a.RuleUID,
	})

	elements := []block{
		button("🔕 Silence", ActionSilenceOpen, string(val), ""),
	}
	if rollbackEnabled {
		elements = append(elements, button("↩️ Rollback", ActionRollbackTrigger, string(val), "danger"))
	}
	return []block{{"type": "actions", "elements": elements}}
}

// ownerMention returns the deploy-owner line: a Slack @mention when the
// committer email resolved, otherwise the committer name (graceful fallback).
func ownerMention(a model.EnrichedAlert) string {
	rev := ""
	if a.Revision != "" {
		rev = fmt.Sprintf("  (rev `%s`)", short(a.Revision))
	}
	switch {
	case a.SlackUserID != "":
		return fmt.Sprintf("*배포 책임자:* <@%s>%s", a.SlackUserID, rev)
	case a.CommitterName != "":
		return fmt.Sprintf("*배포 책임자:* %s (Slack 매칭 실패)%s", a.CommitterName, rev)
	default:
		return ""
	}
}

// --- block kit helpers ---

func section(text string) block {
	return block{"type": "section", "text": map[string]any{"type": "mrkdwn", "text": text}}
}

func mrkdwn(text string) block {
	return block{"type": "mrkdwn", "text": text}
}

func button(text, actionID, value, style string) block {
	b := block{
		"type":      "button",
		"text":      map[string]any{"type": "plain_text", "text": text, "emoji": true},
		"action_id": actionID,
		"value":     value,
	}
	if style != "" {
		b["style"] = style
	}
	return b
}

// link renders "*Label:* <url|text>" or "" when url is empty. An optional
// display text overrides the default "열기".
func link(label, url string, text ...string) string {
	if url == "" {
		return ""
	}
	disp := "열기"
	if len(text) > 0 && text[0] != "" {
		disp = text[0]
	}
	return fmt.Sprintf("*%s:* <%s|%s>", label, url, disp)
}

func joinNonEmpty(sep string, parts ...string) string {
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if out != "" {
			out += sep
		}
		out += p
	}
	return out
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
