package intel

import "errors"

var (
	ErrExpiredOrRevoked = errors.New("expired or revoked")
	ErrInvalidLogFormat = errors.New("text did not match standard EVE log format")
)
