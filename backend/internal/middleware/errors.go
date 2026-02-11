package middleware

import "github.com/pocketbase/pocketbase/tools/router"

var (
	ErrForbidden             = router.NewForbiddenError("Forbidden", nil)
	ErrInvalidUploaderToken  = router.NewUnauthorizedError("Invalid uploader token.", nil)
	ErrMainCharacterRequired = router.NewForbiddenError("Main character required", nil)
	ErrUnauthorized          = router.NewUnauthorizedError("Unauthorized", nil)
)
