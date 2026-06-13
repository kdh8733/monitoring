package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Config is the single source of truth for every external endpoint and
// credential. Every value can be left blank in the central config file and
// filled in later; the server still starts (unconfigured integrations just
// no-op and are reported by Missing()).
type Config struct {
	ListenAddr string

	// Slack
	SlackBotToken      string
	SlackSigningSecret string
	SlackAlertChannel  string
	SlackBaseURL       string // default https://slack.com/api

	// Grafana (central, multi-cluster via labels)
	GrafanaBaseURL  string
	GrafanaAPIToken string

	// Kubernetes (central API access; same {url, token} pattern)
	KubeAPIURL             string
	KubeToken              string
	KubeInsecureSkipVerify bool

	// GitHub
	GitHubBaseURL string // default https://api.github.com
	GitHubToken   string

	// ArgoCD
	ArgoCDBaseURL         string
	ArgoCDToken           string
	ArgoCDRollbackEnabled bool

	// Log source for thread links
	LogSource         string // "kibana" | "prometheus"
	KibanaBaseURL     string
	PrometheusBaseURL string

	// missingResolver re-checks resolved (file+env) values for Missing().
	missingResolver configResolver
}

// configResolver resolves a config key to its final (file+env) value.
type configResolver func(string) string

// requiredKeys are the keys that must be set for the full flow to work.
// They are reported (not enforced) so the service boots with blanks.
var requiredKeys = []string{
	"SLACK_BOT_TOKEN", "SLACK_SIGNING_SECRET", "SLACK_ALERT_CHANNEL",
	"GRAFANA_BASE_URL", "GRAFANA_API_TOKEN",
	"KUBE_API_URL", "KUBE_TOKEN",
	"GITHUB_TOKEN",
	"ARGOCD_BASE_URL", "ARGOCD_TOKEN",
}

// Load reads the central config file at path (dotenv style: KEY=VALUE, with
// '#' comments and blank lines ignored), then lets real OS environment
// variables override file values. A missing file is not an error - env-only
// operation is supported.
func Load(path string) (*Config, error) {
	fileVals, err := parseDotenv(path)
	if err != nil {
		return nil, err
	}

	get := func(key string) string {
		if v, ok := os.LookupEnv(key); ok && v != "" {
			return v
		}
		return strings.TrimSpace(fileVals[key])
	}

	c := &Config{
		ListenAddr: orDefault(get("LISTEN_ADDR"), ":8080"),

		SlackBotToken:      get("SLACK_BOT_TOKEN"),
		SlackSigningSecret: get("SLACK_SIGNING_SECRET"),
		SlackAlertChannel:  get("SLACK_ALERT_CHANNEL"),
		SlackBaseURL:       orDefault(get("SLACK_BASE_URL"), "https://slack.com/api"),

		GrafanaBaseURL:  get("GRAFANA_BASE_URL"),
		GrafanaAPIToken: get("GRAFANA_API_TOKEN"),

		KubeAPIURL:             get("KUBE_API_URL"),
		KubeToken:              get("KUBE_TOKEN"),
		KubeInsecureSkipVerify: asBool(get("KUBE_INSECURE_SKIP_VERIFY")),

		GitHubBaseURL: orDefault(get("GITHUB_BASE_URL"), "https://api.github.com"),
		GitHubToken:   get("GITHUB_TOKEN"),

		ArgoCDBaseURL:         get("ARGOCD_BASE_URL"),
		ArgoCDToken:           get("ARGOCD_TOKEN"),
		ArgoCDRollbackEnabled: asBool(get("ARGOCD_ROLLBACK_ENABLED")),

		LogSource:         orDefault(get("LOG_SOURCE"), "kibana"),
		KibanaBaseURL:     get("KIBANA_BASE_URL"),
		PrometheusBaseURL: get("PROMETHEUS_BASE_URL"),
	}

	// keep file values for Missing() reporting
	c.missingResolver = get
	return c, nil
}

// Missing returns the required keys that resolved to an empty value.
// Use it to warn operators which integrations are not yet configured.
func (c *Config) Missing() []string {
	var out []string
	for _, k := range requiredKeys {
		if c.missingResolver == nil || strings.TrimSpace(c.missingResolver(k)) == "" {
			out = append(out, k)
		}
	}
	return out
}

func parseDotenv(path string) (map[string]string, error) {
	vals := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return vals, nil
		}
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		val = strings.Trim(val, `"'`)
		vals[key] = val
	}
	return vals, sc.Err()
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func asBool(v string) bool {
	b, _ := strconv.ParseBool(strings.TrimSpace(v))
	return b
}
