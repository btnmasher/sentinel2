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
