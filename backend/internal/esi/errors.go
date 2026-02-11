package esi

import "errors"

var (
	ErrAffiliationUnsupported = errors.New("character affiliation not supported by ESI proxy")
	ErrMissingCharacter       = errors.New("missing character")
	ErrMissingUser            = errors.New("missing user")
	ErrMissingUserSub         = errors.New("missing user sub")
	ErrScopeRequired          = errors.New("scope required")
)
