package middleware

import "github.com/pocketbase/pocketbase/core"

func RequireAuth(c *core.RequestEvent) error {
	if c.Auth == nil {
		return ErrUnauthorized
	}
	return c.Next()
}
