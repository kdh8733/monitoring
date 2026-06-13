package notifier

import (
	"context"

	"github.com/kdh8733/monitoring/pkg/model"
)

// Poster is the slice of the Slack client the notifier needs (interface for
// testability).
type Poster interface {
	PostMessage(ctx context.Context, channel, threadTS string, blocks any, fallback string) (string, error)
}

// Notifier posts the main alert message and its enrichment thread.
type Notifier struct {
	Slack           Poster
	Channel         string
	RollbackEnabled bool
}

// Notify posts the main message, then context and action replies in its
// thread. Returns the main message timestamp (thread key).
func (n *Notifier) Notify(ctx context.Context, a model.EnrichedAlert) (string, error) {
	ts, err := n.Slack.PostMessage(ctx, n.Channel, "", BuildMainBlocks(a), fallbackText(a))
	if err != nil {
		return "", err
	}
	if _, err := n.Slack.PostMessage(ctx, n.Channel, ts, BuildContextBlocks(a), "context"); err != nil {
		return ts, err
	}
	if _, err := n.Slack.PostMessage(ctx, n.Channel, ts, BuildActionBlocks(a, n.RollbackEnabled), "actions"); err != nil {
		return ts, err
	}
	return ts, nil
}

func fallbackText(a model.EnrichedAlert) string {
	return orDash(a.RuleName) + " - " + orDash(a.Cluster) + "/" + orDash(a.App)
}

// SilenceModalView builds the Block Kit modal opened by the Silence button.
// privateMeta carries the alert identity (a JSON ButtonValue) back to the
// view_submission handler. The callback_id routes the submission.
func SilenceModalView(privateMeta string) map[string]any {
	return map[string]any{
		"type":             "modal",
		"callback_id":      "silence_submit",
		"private_metadata": privateMeta,
		"title":            plain("Silence 생성"),
		"submit":           plain("적용"),
		"close":            plain("취소"),
		"blocks": []map[string]any{
			{
				"type":     "input",
				"block_id": "duration",
				"label":    plain("기간"),
				"element": map[string]any{
					"type":           "static_select",
					"action_id":      "value",
					"initial_option": durationOption("1h"),
					"options": []map[string]any{
						durationOption("1h"), durationOption("4h"),
						durationOption("12h"), durationOption("24h"),
					},
				},
			},
			{
				"type":     "input",
				"block_id": "comment",
				"label":    plain("사유"),
				"element": map[string]any{
					"type":        "plain_text_input",
					"action_id":   "value",
					"multiline":   true,
					"placeholder": plain("왜 silence 하나요?"),
				},
			},
		},
	}
}

func plain(s string) map[string]any {
	return map[string]any{"type": "plain_text", "text": s, "emoji": true}
}

func durationOption(d string) map[string]any {
	return map[string]any{"text": plain(d), "value": d}
}
