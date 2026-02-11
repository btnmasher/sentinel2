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

	DefaultOIDCIssuer    = "https://sso.pleaseignore.com/auth/realms/auth-ng"
	DefaultAuthEndpoint  = "https://sso.pleaseignore.com/auth/realms/auth-ng/protocol/openid-connect/auth"
	DefaultTokenEndpoint = "https://sso.pleaseignore.com/auth/realms/auth-ng/protocol/openid-connect/token"

	DefaultAuthBackend = "eve"

	DefaultEVEAuthURL  = "https://login.eveonline.com/v2/oauth/authorize"
	DefaultEVETokenURL = "https://login.eveonline.com/v2/oauth/token"
)

type Config struct {
	SentinelVersion string

	AuthBackend string `long:"auth-backend" env:"AUTH_BACKEND" default:"eve"`

	OIDCIssuer        string `long:"oidc-issuer" env:"OIDC_ISSUER" default:"https://sso.pleaseignore.com/auth/realms/auth-ng"`
	OIDCAuthURL       string `long:"oidc-auth-url" env:"OIDC_AUTH_URL" default:"https://sso.pleaseignore.com/auth/realms/auth-ng/protocol/openid-connect/auth"`
	OIDCTokenURL      string `long:"oidc-token-url" env:"OIDC_TOKEN_URL" default:"https://sso.pleaseignore.com/auth/realms/auth-ng/protocol/openid-connect/token"`
	OIDCUserInfoURL   string `long:"oidc-userinfo-url" env:"OIDC_USERINFO_URL" default:"https://sso.pleaseignore.com/auth/realms/auth-ng/protocol/openid-connect/userinfo"`
	OIDCClientID      string `long:"oidc-client-id" env:"OIDC_CLIENT_ID"`
	OIDCClientSecret  string `long:"oidc-client-secret" env:"OIDC_CLIENT_SECRET"`
	OIDCScopes        string `long:"oidc-scopes" env:"OIDC_SCOPES" default:"openid"`
	OIDCRequiredRoles string `long:"oidc-required-roles" env:"OIDC_REQUIRED_ROLES" default:"urn:sso:alliance:test-alliance,urn:sso:allies"`
	OIDCStaffRoles    string `long:"oidc-staff-roles" env:"OIDC_STAFF_ROLES" default:"urn:sso:staff_user"`
	OIDCPortalURL     string `long:"oidc-portal-url" env:"OIDC_PORTAL_URL" default:"https://auth.pleaseignore.com"`

	EVEClientID     string `long:"eve-client-id" env:"EVE_CLIENT_ID"`
	EVEClientSecret string `long:"eve-client-secret" env:"EVE_CLIENT_SECRET"`
	EVEAuthURL      string `long:"eve-auth-url" env:"EVE_AUTH_URL" default:"https://login.eveonline.com/v2/oauth/authorize"`
	EVETokenURL     string `long:"eve-token-url" env:"EVE_TOKEN_URL" default:"https://login.eveonline.com/v2/oauth/token"`
	EVEScopes       string `long:"eve-scopes" env:"EVE_SCOPES" default:"esi-location.read_location.v1 esi-ui.write_waypoint.v1"`

	ESIDirectBaseURL string `long:"esi-direct-base-url" env:"ESI_DIRECT_BASE_URL" default:"https://esi.evetech.net/latest/"`
	ESIProxyBaseURL  string `long:"esi-proxy-base-url" env:"ESI_PROXY_BASE_URL" default:"https://auth.pleaseignore.com/esi/"`
	ESIUserAgent     string

	DefaultMapRegions string `long:"default-map-regions" env:"DEFAULT_MAP_REGIONS" default:"10000029"`

	FrontendDevProxy string `long:"dev-proxy" env:"DEV_PROXY"`
	DebugEnabled     bool   `long:"dev" env:"DEBUG_ENABLED"`
	LogLevel         string `long:"log-level" env:"LOG_LEVEL" default:"info" choice:"debug" choice:"info" choice:"warn" choice:"error"`
	LogPretty        bool   `long:"log-pretty" env:"LOG_PRETTY"`
	LogPrettyPB      bool   `long:"log-pretty-pb" env:"LOG_PRETTY_PB"`
	LogJSON          bool   `long:"log-json" env:"LOG_JSON"`
	LogJSONPath      string `long:"log-json-path" env:"LOG_JSON_PATH"`
	LogJSONPB        bool   `long:"log-json-pb" env:"LOG_JSON_PB"`
}

func Load() Config {
	cfg := Config{}
	parser := flags.NewParser(&cfg, flags.Default)
	_, parseErr := parser.Parse()
	if parseErr == nil {
		cfg.AuthBackend = normalizeAuthBackend(cfg.AuthBackend)
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

func (c Config) RequiredRoles() []string {
	return splitCSV(c.OIDCRequiredRoles)
}

func (c Config) StaffRoles() []string {
	return splitCSV(c.OIDCStaffRoles)
}

func (c Config) EVEScopeList() []string {
	return splitScopes(c.EVEScopes)
}

func (c Config) DefaultRegions() []string {
	return splitRegions(c.DefaultMapRegions)
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	out := []string{}
	current := ""
	for _, r := range value {
		if r == ',' {
			if current != "" {
				out = append(out, current)
				current = ""
				continue
			}
			current = ""
			continue
		}
		if r != ' ' {
			current += string(r)
		}
	}
	if current != "" {
		out = append(out, current)
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
