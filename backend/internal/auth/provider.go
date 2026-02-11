package auth

import (
	"context"

	"github.com/pocketbase/pocketbase/core"
)

type Provider interface {
	Name() string
	Authenticate(c *core.RequestEvent, flow AuthFlow) error
	BuildAuthURL(c *core.RequestEvent, flow AuthFlow) (string, error)
	Callback(c *core.RequestEvent) (*AuthResult, AuthFlow, error)
	Refresh(ctx context.Context, user *core.Record) (AuthTokens, error)
	Logout(c *core.RequestEvent) error
}
