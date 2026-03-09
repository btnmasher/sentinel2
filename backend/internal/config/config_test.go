package config

import (
	"reflect"
	"testing"

	flags "github.com/jessevdk/go-flags"
)

func TestTrustedProxyHeaders_DefaultCloudflare(t *testing.T) {
	cfg := Config{}
	parser := flags.NewParser(&cfg, flags.Default)
	if _, err := parser.ParseArgs(nil); err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	got := cfg.TrustedProxyHeaders
	want := []string{"CF-Connecting-IP", "True-Client-IP"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("headers = %v, want %v", got, want)
	}
}

func TestTrustedProxyHeaders_ParsesEnvDelim(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_HEADERS", "CF-Connecting-IP,True-Client-IP")
	cfg := Config{}
	parser := flags.NewParser(&cfg, flags.Default)
	if _, err := parser.ParseArgs(nil); err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	got := cfg.TrustedProxyHeaders
	want := []string{"CF-Connecting-IP", "True-Client-IP"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("headers = %v, want %v", got, want)
	}
}

func TestTimersEnabled_DefaultTrue(t *testing.T) {
	cfg := Config{TimersEnabled: true}
	parser := flags.NewParser(&cfg, flags.Default)
	if _, err := parser.ParseArgs(nil); err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if !cfg.TimersEnabled {
		t.Fatal("TimersEnabled = false, want true")
	}
}

func TestTimersEnabled_ParsesEnvFalse(t *testing.T) {
	t.Setenv("TIMERS_ENABLED", "false")
	cfg := Config{TimersEnabled: true}
	parser := flags.NewParser(&cfg, flags.Default)
	if _, err := parser.ParseArgs(nil); err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if cfg.TimersEnabled {
		t.Fatal("TimersEnabled = true, want false")
	}
}

func TestTimerSource_DefaultStandalone(t *testing.T) {
	cfg := Config{}
	parser := flags.NewParser(&cfg, flags.Default)
	if _, err := parser.ParseArgs(nil); err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if cfg.TimerSource != TimerSourceStandalone {
		t.Fatalf("TimerSource = %q, want %q", cfg.TimerSource, TimerSourceStandalone)
	}
}

func TestTimerSource_ParsesEnvWebhook(t *testing.T) {
	t.Setenv("TIMER_SOURCE", TimerSourceWebhook)
	cfg := Config{}
	parser := flags.NewParser(&cfg, flags.Default)
	if _, err := parser.ParseArgs(nil); err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if cfg.TimerSource != TimerSourceWebhook {
		t.Fatalf("TimerSource = %q, want %q", cfg.TimerSource, TimerSourceWebhook)
	}
}
