package esi

import "errors"

var (
	ErrAffiliationUnsupported   = errors.New("character affiliation not supported by ESI proxy")
	ErrNotModified              = errors.New("esi resource not modified")
	ErrOrganizationInactive     = errors.New("organization inactive or closed")
	ErrMissingCharacter         = errors.New("missing character")
	ErrMissingUser              = errors.New("missing user")
	ErrMissingUserSub           = errors.New("missing user sub")
	ErrScopeRequired            = errors.New("scope required")
	ErrNotificationsUnsupported = errors.New("character notifications not supported by ESI proxy")
	ErrRateLimited              = errors.New("esi rate limited")
)
