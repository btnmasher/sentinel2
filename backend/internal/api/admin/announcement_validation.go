package admin

import (
	"errors"
	"strings"
)

var (
	errInvalidAnnouncementVariant  = errors.New("invalid announcement variant")
	errAnnouncementMessageRequired = errors.New("announcement message is required")
)

func normalizeAnnouncementPayload(payload siteAnnouncementPayload) (string, string, error) {
	variant := strings.TrimSpace(strings.ToLower(payload.Variant))
	if variant != "banner" && variant != "modal" {
		return "", "", errInvalidAnnouncementVariant
	}
	message := strings.TrimSpace(payload.Message)
	if message == "" {
		return "", "", errAnnouncementMessageRequired
	}
	return variant, message, nil
}
