// Command server is the entry point for the Monitoring alert orchestrator.
//
// It exposes two ingress endpoints that the system depends on:
//
//	POST /webhook/grafana    - Grafana Alerting contact point delivers firing/resolved alerts here.
//	POST /slack/interactivity - Slack posts Block Kit interaction payloads (silence modal, rollback button) here.
//
// The handlers below are intentionally stubs. Real behavior is built per the
// ROADMAP milestones (webhook parse -> enrich -> Slack thread).
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Grafana Alerting -> orchestrator. M1 fills this in.
	mux.HandleFunc("POST /webhook/grafana", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	})

	// Slack interactivity (silence modal submit, rollback button). M3/M5 fill this in.
	mux.HandleFunc("POST /slack/interactivity", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	})

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("monitoring orchestrator listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
