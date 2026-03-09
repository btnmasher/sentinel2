package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	flags "github.com/jessevdk/go-flags"
)

const (
	SentinelVersion = "2.0.1"

	DefaultOIDCIssuer   = "https://sso.pleaseignore.com/auth/realms/auth-ng"
	DefaultAuthEndpoint = "https://sso.pleaseignore.com/auth/realms/auth-ng/protocol/openid-connect/auth"
	//nolint:gosec // endpoint URL, not a credential
	DefaultTokenEndpoint = "https://sso.pleaseignore.com/auth/realms/auth-ng/protocol/openid-connect/token"

	DefaultAuthBackend = "eve"

	DefaultEVEAuthURL = "https://login.eveonline.com/v2/oauth/authorize"
	//nolint:gosec // endpoint URL, not a credential
	DefaultEVETokenURL = "https://login.eveonline.com/v2/oauth/token"

	TimerSourceStandalone = "standalone"
	TimerSourceWebhook    = "webhook"
)

type Config struct {
	SentinelVersion string

	AuthBackend string `long:"auth-backend" env:"AUTH_BACKEND" default:"eve"`

	OIDCIssuer        string   `long:"oidc-issuer" env:"OIDC_ISSUER" default:"https://sso.pleaseignore.com/auth/realms/auth-ng"`
	OIDCAuthURL       string   `long:"oidc-auth-url" env:"OIDC_AUTH_URL" default:"https://sso.pleaseignore.com/auth/realms/auth-ng/protocol/openid-connect/auth"`
	OIDCTokenURL      string   `long:"oidc-token-url" env:"OIDC_TOKEN_URL" default:"https://sso.pleaseignore.com/auth/realms/auth-ng/protocol/openid-connect/token"`
	OIDCUserInfoURL   string   `long:"oidc-userinfo-url" env:"OIDC_USERINFO_URL" default:"https://sso.pleaseignore.com/auth/realms/auth-ng/protocol/openid-connect/userinfo"`
	OIDCClientID      string   `long:"oidc-client-id" env:"OIDC_CLIENT_ID"`
	OIDCClientSecret  string   `long:"oidc-client-secret" env:"OIDC_CLIENT_SECRET"`
	OIDCScopes        []string `long:"oidc-scopes" env:"OIDC_SCOPES" env-delim:"," default:"openid"`
	OIDCRequiredRoles []string `long:"oidc-required-roles" env:"OIDC_REQUIRED_ROLES" env-delim:"," default:"urn:sso:alliance:test-alliance" default:"urn:sso:allies"`
	OIDCStaffRoles    []string `long:"oidc-staff-roles" env:"OIDC_STAFF_ROLES" env-delim:"," default:"urn:sso:staff_user"`
	OIDCPortalURL     string   `long:"oidc-portal-url" env:"OIDC_PORTAL_URL" default:"https://auth.pleaseignore.com"`

	EVEClientID     string   `long:"eve-client-id" env:"EVE_CLIENT_ID"`
	EVEClientSecret string   `long:"eve-client-secret" env:"EVE_CLIENT_SECRET"`
	EVEAuthURL      string   `long:"eve-auth-url" env:"EVE_AUTH_URL" default:"https://login.eveonline.com/v2/oauth/authorize"`
	EVETokenURL     string   `long:"eve-token-url" env:"EVE_TOKEN_URL" default:"https://login.eveonline.com/v2/oauth/token"`
	EVEScopes       []string `long:"eve-scopes" env:"EVE_SCOPES" env-delim:"," default:"esi-search.search_structures.v1" default:"esi-universe.read_structures.v1" default:"esi-location.read_online.v1" default:"esi-location.read_location.v1" default:"esi-ui.write_waypoint.v1" default:"esi-characters.read_notifications.v1"`

	ESIDirectBaseURL string `long:"esi-direct-base-url" env:"ESI_DIRECT_BASE_URL" default:"https://esi.evetech.net/latest/"`
	ESIProxyBaseURL  string `long:"esi-proxy-base-url" env:"ESI_PROXY_BASE_URL" default:"https://auth.pleaseignore.com/esi/"`
	ESIUserAgent     string

	DefaultMapRegions    string `long:"default-map-regions" env:"DEFAULT_MAP_REGIONS" default:"10000029"`
	IntelReportHashSlots int    `long:"intel-report-hash-slots" env:"INTEL_REPORT_HASH_SLOTS" default:"20"`
	ZKillFeedEnabled     bool   `long:"zkill-feed-enabled" env:"ZKILL_FEED_ENABLED"`
	ZKillFeedBaseURL     string `long:"zkill-feed-base-url" env:"ZKILL_FEED_BASE_URL" default:"https://r2z2.zkillboard.com/ephemeral"`
	ZKillFeedPollSeconds int    `long:"zkill-feed-poll-seconds" env:"ZKILL_FEED_POLL_SECONDS" default:"10"`
	ZKillMaxEventAgeSec  int    `long:"zkill-max-event-age-seconds" env:"ZKILL_MAX_EVENT_AGE_SECONDS" default:"300"`
	UploaderGitHubRepo   string `long:"uploader-github-repo" env:"UPLOADER_GITHUB_REPO" default:"btnmasher/sentinel2-uploader"`
	TimersEnabled        bool   `long:"timers-enabled" env:"TIMERS_ENABLED"`
	TimerSource          string `long:"timer-source" env:"TIMER_SOURCE" default:"standalone" choice:"standalone" choice:"webhook"`
	TimersWebhookToken   string `long:"timers-webhook-bearer-token" env:"TIMERS_WEBHOOK_BEARER_TOKEN"`

	FrontendDevProxy    string   `long:"dev-proxy" env:"DEV_PROXY"`
	DebugEnabled        bool     `long:"dev" env:"DEBUG_ENABLED"`
	LogLevel            string   `long:"log-level" env:"LOG_LEVEL" default:"info" choice:"debug" choice:"info" choice:"warn" choice:"error"`
	LogPretty           bool     `long:"log-pretty" env:"LOG_PRETTY"`
	LogPrettyPB         bool     `long:"log-pretty-pb" env:"LOG_PRETTY_PB"`
	LogJSON             bool     `long:"log-json" env:"LOG_JSON"`
	LogJSONPath         string   `long:"log-json-path" env:"LOG_JSON_PATH"`
	LogJSONPB           bool     `long:"log-json-pb" env:"LOG_JSON_PB"`
	TrustedProxyHeaders []string `long:"trusted-proxy-headers" env:"TRUSTED_PROXY_HEADERS" env-delim:"," default:"CF-Connecting-IP" default:"True-Client-IP"`
}

func Load() Config {
	cfg := Config{
		TimersEnabled:    true,
		ZKillFeedEnabled: true,
	}
	parser := flags.NewParser(&cfg, flags.Default|flags.IgnoreUnknown)
	_, parseErr := parser.Parse()
	if parseErr == nil {
		cfg.AuthBackend = normalizeAuthBackend(cfg.AuthBackend)
		cfg.TimerSource = normalizeTimerSource(cfg.TimerSource)
		return cfg
	}

	var flagsErr *flags.Error
	if errors.As(parseErr, &flagsErr) && flagsErr.Type == flags.ErrHelp {
		os.Exit(0)
	}
	log.Fatalf("failed to parse config: %v", parseErr)
	return cfg
}

func (c *Config) EnsureESIUserAgent() {
	if strings.TrimSpace(c.ESIUserAgent) != "" {
		return
	}
	version := strings.TrimSpace(c.SentinelVersion)
	if version == "" {
		version = SentinelVersion
	}
	c.ESIUserAgent = fmt.Sprintf("sentinel2/%s", version)
}

func normalizeAuthBackend(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return value
}

func normalizeTimerSource(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case TimerSourceWebhook:
		return TimerSourceWebhook
	default:
		return TimerSourceStandalone
	}
}

func (c *Config) RequiredRoles() []string {
	return normalizeScopes(c.OIDCRequiredRoles)
}

func (c *Config) StaffRoles() []string {
	return normalizeScopes(c.OIDCStaffRoles)
}

func (c *Config) EVEScopeList() []string {
	return normalizeScopes(c.EVEScopes)
}

func (c *Config) DefaultRegions() []string {
	return splitRegions(c.DefaultMapRegions)
}

func (c *Config) TimersReadOnly() bool {
	if c == nil {
		return false
	}
	return c.TimerSource == TimerSourceWebhook
}

func normalizeScopes(value []string) []string {
	if len(value) == 0 {
		return nil
	}
	out := make([]string, 0, len(value))
	for _, entry := range value {
		trimmed := strings.TrimSpace(entry)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func splitScopes(value string) []string {
	if value == "" {
		return nil
	}
	out := []string{}
	current := ""
	for _, r := range value {
		if r == ' ' {
			if current != "" {
				out = append(out, current)
				current = ""
				continue
			}
			continue
		}
		current += string(r)
	}

	if current != "" {
		out = append(out, current)
	}
	return out
}

func splitRegions(value string) []string {
	if value == "" {
		return nil
	}
	cleaned := strings.NewReplacer("+", ",", ";", ",").Replace(value)
	parts := strings.Split(cleaned, ",")
	out := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
