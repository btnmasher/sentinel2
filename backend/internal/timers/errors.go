package timers

import (
	"errors"
	"sentinel2/internal/format"
)

var (
	ErrMissingDate                      = format.ErrMissingDateTime
	ErrInvalidDate                      = format.ErrInvalidDateTime
	ErrMissingCreateInput               = errors.New("missing create input")
	ErrMissingUpdateInput               = errors.New("missing update input")
	ErrMissingInput                     = errors.New("missing input")
	ErrMissingSystem                    = errors.New("missing system")
	ErrSystemNotFound                   = errors.New("system not found")
	ErrMissingExpiresAt                 = errors.New("missing expires_at")
	ErrMoonRequired                     = errors.New("moon is required for selected structure")
	ErrPlanetRequired                   = errors.New("planet is required for selected structure")
	ErrInvalidSkyhookFullnessPercentage = errors.New("invalid skyhook fullness percentage")
	ErrEmptyTimerText                   = errors.New("empty timer text")
	ErrNoTimerDateFound                 = errors.New("no timer date found")
	ErrInvalidTimerDate                 = errors.New("invalid timer date")
	ErrESIPublicClientNotConfigured     = errors.New("esi public client not configured")
)
