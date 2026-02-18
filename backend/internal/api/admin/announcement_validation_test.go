package admin

import (
	"errors"
	"testing"
)

func TestNormalizeAnnouncementPayload_Valid(t *testing.T) {
	variant, message, err := normalizeAnnouncementPayload(siteAnnouncementPayload{
		Variant: "  MODAL ",
		Message: "  ## Release notes  ",
	})
	if err != nil {
		t.Fatalf("normalizeAnnouncementPayload() err = %v, want nil", err)
	}
	if variant != "modal" {
		t.Fatalf("variant = %q, want %q", variant, "modal")
	}
	if message != "## Release notes" {
		t.Fatalf("message = %q, want %q", message, "## Release notes")
	}
}

func TestNormalizeAnnouncementPayload_InvalidVariant(t *testing.T) {
	_, _, err := normalizeAnnouncementPayload(siteAnnouncementPayload{
		Variant: "toast",
		Message: "hello",
	})
	if !errors.Is(err, errInvalidAnnouncementVariant) {
		t.Fatalf("err = %v, want %v", err, errInvalidAnnouncementVariant)
	}
}

func TestNormalizeAnnouncementPayload_EmptyMessage(t *testing.T) {
	_, _, err := normalizeAnnouncementPayload(siteAnnouncementPayload{
		Variant: "banner",
		Message: "   ",
	})
	if !errors.Is(err, errAnnouncementMessageRequired) {
		t.Fatalf("err = %v, want %v", err, errAnnouncementMessageRequired)
	}
}

func TestNormalizeAnnouncementPayload_BannerVariantAccepted(t *testing.T) {
	variant, message, err := normalizeAnnouncementPayload(siteAnnouncementPayload{
		Variant: " banner ",
		Message: " uptime tonight ",
	})
	if err != nil {
		t.Fatalf("normalizeAnnouncementPayload() err = %v, want nil", err)
	}
	if variant != "banner" {
		t.Fatalf("variant = %q, want %q", variant, "banner")
	}
	if message != "uptime tonight" {
		t.Fatalf("message = %q, want %q", message, "uptime tonight")
	}
}

func TestNormalizeAnnouncementPayload_InvalidVariantCheckedFirst(t *testing.T) {
	_, _, err := normalizeAnnouncementPayload(siteAnnouncementPayload{
		Variant: "invalid",
		Message: "   ",
	})
	if !errors.Is(err, errInvalidAnnouncementVariant) {
		t.Fatalf("err = %v, want %v", err, errInvalidAnnouncementVariant)
	}
}
