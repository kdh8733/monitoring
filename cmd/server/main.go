// Command server is the entry point for the Monitoring alert orchestrator.
//
// Ingress endpoints:
//
//	GET  /healthz             - liveness.
//	POST /webhook/grafana     - Grafana Alerting contact point: enrich + post to Slack.
//	POST /webhook/alertmanager - Prometheus AlertManager webhook (same handler).
//	POST /slack/interactivity - Slack Block Kit actions: silence modal, rollback.
//
// All endpoints/credentials come from the central config file (default .env);
// blank values disable the corresponding integration without crashing.
package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kdh8733/monitoring/internal/config"
	"github.com/kdh8733/monitoring/internal/enricher"
	"github.com/kdh8733/monitoring/internal/notifier"
	"github.com/kdh8733/monitoring/internal/webhook"
	"github.com/kdh8733/monitoring/pkg/argocd"
	"github.com/kdh8733/monitoring/pkg/github"
	"github.com/kdh8733/monitoring/pkg/grafana"
	"github.com/kdh8733/monitoring/pkg/kube"
	"github.com/kdh8733/monitoring/pkg/model"
	"github.com/kdh8733/monitoring/pkg/slackapi"
)

type app struct {
	cfg     *config.Config
	enr     *enricher.Enricher
	ntf     *notifier.Notifier
	slack   *slackapi.Client
	grafana *grafana.Client
	argocd  *argocd.Client
}

func main() {
	cfgPath := getenvDefault("CONFIG_FILE", ".env")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if miss := cfg.Missing(); len(miss) > 0 {
		log.Printf("config: unconfigured (blank) keys, related integrations disabled: %s", strings.Join(miss, ", "))
	}

	slack := slackapi.New(cfg.SlackBaseURL, cfg.SlackBotToken)
	graf := grafana.New(cfg.GrafanaBaseURL, cfg.GrafanaAPIToken)
	argo := argocd.New(cfg.ArgoCDBaseURL, cfg.ArgoCDToken)

	a := &app{
		cfg:     cfg,
		slack:   slack,
		grafana: graf,
		argocd:  argo,
		enr: &enricher.Enricher{
			Kube:   kube.New(cfg.KubeAPIURL, cfg.KubeToken, cfg.KubeInsecureSkipVerify),
			Argo:   argo,
			GitHub: github.New(cfg.GitHubBaseURL, cfg.GitHubToken),
			Slack:  slack,
			LogURL: logLinker(cfg),
			Logf:   log.Printf,
		},
		ntf: &notifier.Notifier{
			Slack:           slack,
			Channel:         cfg.SlackAlertChannel,
			RollbackEnabled: cfg.ArgoCDRollbackEnabled,
			ImageEnabled:    cfg.GrafanaImageEnabled,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	// Both engines post a compatible payload; one handler serves both routes.
	mux.HandleFunc("POST /webhook/grafana", a.handleWebhook)
	mux.HandleFunc("POST /webhook/alertmanager", a.handleWebhook)
	mux.HandleFunc("POST /slack/interactivity", a.handleInteractivity)

	log.Printf("monitoring orchestrator listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatal(err)
	}
}

func (a *app) handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	alerts, err := webhook.ParseWebhook(body)
	if err != nil {
		http.Error(w, "parse", http.StatusBadRequest)
		return
	}
	// Acknowledge fast; do the network-heavy enrich+post in the background so
	// Grafana does not retry on slow downstreams.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for _, al := range alerts {
			ea := a.enr.Enrich(ctx, al)
			if _, err := a.ntf.Notify(ctx, ea); err != nil {
				log.Printf("notify %s: %v", al.RuleName, err)
			}
		}
	}()
	w.WriteHeader(http.StatusOK)
}

// slackInteraction is the subset of a Slack interactivity payload we read.
type slackInteraction struct {
	Type      string `json:"type"`
	TriggerID string `json:"trigger_id"`
	User      struct {
		ID string `json:"id"`
	} `json:"user"`
	Actions []struct {
		ActionID string `json:"action_id"`
		Value    string `json:"value"`
	} `json:"actions"`
	View struct {
		CallbackID      string `json:"callback_id"`
		PrivateMetadata string `json:"private_metadata"`
		State           struct {
			Values map[string]map[string]struct {
				Value          string `json:"value"`
				SelectedOption struct {
					Value string `json:"value"`
				} `json:"selected_option"`
			} `json:"values"`
		} `json:"state"`
	} `json:"view"`
}

func (a *app) handleInteractivity(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	if err := webhook.VerifySlackSignature(
		a.cfg.SlackSigningSecret,
		r.Header.Get("X-Slack-Request-Timestamp"),
		body,
		r.Header.Get("X-Slack-Signature"),
		time.Now(),
	); err != nil {
		log.Printf("slack signature: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// payload is form-encoded: payload=<json>
	form, err := url.ParseQuery(string(body))
	if err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	var p slackInteraction
	if err := json.Unmarshal([]byte(form.Get("payload")), &p); err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}

	switch p.Type {
	case "block_actions":
		a.handleBlockAction(p, w)
	case "view_submission":
		a.handleViewSubmission(p, w)
	default:
		w.WriteHeader(http.StatusOK)
	}
}

func (a *app) handleBlockAction(p slackInteraction, w http.ResponseWriter) {
	if len(p.Actions) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}
	act := p.Actions[0]
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	switch act.ActionID {
	case notifier.ActionSilenceOpen:
		if err := a.slack.OpenView(ctx, p.TriggerID, notifier.SilenceModalView(act.Value)); err != nil {
			log.Printf("open silence modal: %v", err)
		}
	case notifier.ActionRollbackTrigger:
		a.doRollback(ctx, act.Value)
	}
	w.WriteHeader(http.StatusOK)
}

func (a *app) handleViewSubmission(p slackInteraction, w http.ResponseWriter) {
	if p.View.CallbackID != "silence_submit" {
		w.WriteHeader(http.StatusOK)
		return
	}
	var bv notifier.ButtonValue
	_ = json.Unmarshal([]byte(p.View.PrivateMetadata), &bv)

	dur := time.Hour
	comment := ""
	if vals := p.View.State.Values; vals != nil {
		if d := vals["duration"]["value"].SelectedOption.Value; d != "" {
			if pd, err := time.ParseDuration(d); err == nil {
				dur = pd
			}
		}
		comment = vals["comment"]["value"].Value
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	matchers := silenceMatchers(bv)
	createdBy := "slack:" + p.User.ID
	if id, err := a.grafana.CreateSilence(ctx, matchers, dur, createdBy, comment); err != nil {
		log.Printf("create silence: %v", err)
	} else {
		log.Printf("silence created id=%s app=%s", id, bv.App)
	}
	// Empty 200 closes the modal.
	w.WriteHeader(http.StatusOK)
}

func (a *app) doRollback(ctx context.Context, value string) {
	if !a.cfg.ArgoCDRollbackEnabled {
		return
	}
	var bv notifier.ButtonValue
	if err := json.Unmarshal([]byte(value), &bv); err != nil || bv.App == "" {
		return
	}
	info, err := a.argocd.AppInfo(ctx, bv.App)
	if err != nil {
		log.Printf("rollback appinfo %s: %v", bv.App, err)
		return
	}
	if info.PrevRevision == "" {
		log.Printf("rollback %s: no previous revision", bv.App)
		return
	}
	if err := a.argocd.Sync(ctx, bv.App, info.PrevRevision); err != nil {
		log.Printf("rollback sync %s: %v", bv.App, err)
		return
	}
	log.Printf("rollback %s -> %s requested", bv.App, info.PrevRevision)
}

func silenceMatchers(bv notifier.ButtonValue) []grafana.Matcher {
	var m []grafana.Matcher
	add := func(name, val string) {
		if val != "" {
			m = append(m, grafana.Matcher{Name: name, Value: val, IsEqual: true})
		}
	}
	add("alertname", bv.RuleName)
	add("cluster", bv.Cluster)
	add("namespace", bv.Namespace)
	add("app", bv.App)
	return m
}

// logLinker builds the log-source deep link based on the configured source.
func logLinker(cfg *config.Config) func(model.Alert) string {
	return func(a model.Alert) string {
		switch cfg.LogSource {
		case "prometheus":
			if cfg.PrometheusBaseURL == "" {
				return ""
			}
			return cfg.PrometheusBaseURL + "/graph"
		default: // kibana
			if cfg.KibanaBaseURL == "" {
				return ""
			}
			q := url.QueryEscape("kubernetes.namespace_name:\"" + a.Namespace + "\" and kubernetes.labels.app:\"" + a.App + "\"")
			return cfg.KibanaBaseURL + "/app/discover#/?_a=(query:(language:kuery,query:'" + q + "'))"
		}
	}
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
