package server

import (
	"reflect"
	"testing"
)

func TestPreferredTrustedProxyHeaders_CloudflareDefaults(t *testing.T) {
	got := preferredTrustedProxyHeaders
	want := []string{"CF-Connecting-IP", "True-Client-IP"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("headers = %v, want %v", got, want)
	}
}
