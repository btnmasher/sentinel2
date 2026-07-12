package auth

import (
	"errors"
	"net/http"

	"github.com/pocketbase/pocketbase/tools/router"
)

var (
	ErrAccessDenied             = router.NewApiError(http.StatusForbidden, "Access denied", nil)
	ErrCharacterAlreadyLinked   = router.NewApiError(http.StatusConflict, "Character already linked", nil)
	ErrFailedCheckAccess        = router.NewApiError(http.StatusInternalServerError, "Failed to check access", nil)
	ErrFailedCreateNonce        = router.NewApiError(http.StatusInternalServerError, "Failed to create nonce", nil)
	ErrFailedCreateState        = router.NewApiError(http.StatusInternalServerError, "Failed to create state", nil)
	ErrFailedDecodeToken        = router.NewApiError(http.StatusBadGateway, "Failed to decode token", nil)
	ErrFailedExchangeToken      = router.NewApiError(http.StatusBadRequest, "Failed to exchange token", nil)
	ErrFailedFetchCharacter     = router.NewApiError(http.StatusBadGateway, "Failed to fetch character", nil)
	ErrFailedFetchMainCharacter = router.NewApiError(http.StatusBadGateway, "Failed to fetch main character", nil)
	ErrFailedIssueExchangeCode  = router.NewApiError(http.StatusInternalServerError, "Failed to issue exchange code", nil)
	ErrFailedIssueToken         = router.NewApiError(http.StatusInternalServerError, "Failed to issue token", nil)
	ErrFailedPersistCharacter   = router.NewApiError(http.StatusInternalServerError, "Failed to persist character", nil)
	ErrFailedPersistUser        = router.NewApiError(http.StatusInternalServerError, "Failed to persist user", nil)
	ErrInvalidCharacter         = router.NewApiError(http.StatusBadGateway, "Invalid character", nil)
	ErrCharacterNotLinkable     = router.NewApiError(http.StatusBadRequest, "Character not linkable", nil)
	ErrInvalidCode              = router.NewApiError(http.StatusBadRequest, "Invalid code", nil)
	ErrInvalidState             = router.NewApiError(http.StatusBadRequest, "Invalid state", nil)
	ErrMissingCode              = router.NewApiError(http.StatusBadRequest, "Missing code", nil)
	ErrMissingAccessToken       = router.NewApiError(http.StatusUnauthorized, "Missing access token", nil)
	ErrMissingRequiredRoles     = router.NewApiError(http.StatusForbidden, "Missing required roles", nil)
	ErrMissingSub               = router.NewApiError(http.StatusUnauthorized, "Missing sub", nil)
	ErrUnauthorized             = router.NewApiError(http.StatusUnauthorized, "Unauthorized", nil)
	ErrUserNotFound             = router.NewApiError(http.StatusNotFound, "User not found", nil)

	// testauth errors
	ErrTestAuthDiscovery = errors.New("oauth discovery failed")
	ErrTestAuthToken     = errors.New("token exchange failed")
	ErrTestAuthRefresh   = errors.New("token refresh failed")
	ErrTestAuthUserInfo  = errors.New("user info fetch failed")
	ErrTestAuthEsiProxy  = errors.New("esi proxy failed")
	ErrUserInfoFetch     = errors.New("user info fetch failed")
)
