package auth

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/store"
)

type Manager struct {
	App      *pocketbase.PocketBase
	Provider Provider
}

func NewManager(app *pocketbase.PocketBase, provider Provider) *Manager {
	return &Manager{App: app, Provider: provider}
}

func (m *Manager) Authenticate(c *core.RequestEvent, flow AuthFlow) error {
	return m.Provider.Authenticate(c, flow)
}

func (m *Manager) BuildAuthURL(c *core.RequestEvent, flow AuthFlow) (string, error) {
	return m.Provider.BuildAuthURL(c, flow)
}

type AuthCallbackResult struct {
	ExchangeCode string
	IsLink       bool
}

func (m *Manager) Callback(c *core.RequestEvent) (*AuthCallbackResult, error) {
	result, flow, callbackErr := m.Provider.Callback(c)
	if callbackErr == nil {
		if flow.Type == FlowLink {
			return &AuthCallbackResult{IsLink: true}, nil
		}

		code := saveAuthExchange(m.App, result.UserID)
		if code == "" {
			return nil, ErrFailedIssueExchangeCode
		}
		return &AuthCallbackResult{ExchangeCode: code}, nil
	}

	return nil, callbackErr
}

func (m *Manager) Logout(c *core.RequestEvent) error {
	return m.Provider.Logout(c)
}

func (m *Manager) IssueToken(user *core.Record) (string, error) {
	if user == nil {
		return "", ErrUnauthorized
	}
	token, tokenErr := user.NewStaticAuthToken(pbTokenTTL)
	if tokenErr != nil {
		return "", ErrFailedIssueToken
	}
	return token, nil
}

func (m *Manager) Exchange(code string) (*core.Record, string, error) {
	ex, ok := loadAuthExchange(m.App, code)
	if !ok {
		return nil, "", ErrInvalidCode
	}
	deleteAuthExchange(m.App, code)

	user, userErr := m.App.FindRecordById(store.CollectionUsers, ex.UserID)
	if userErr != nil {
		return nil, "", ErrUserNotFound
	}

	token, tokenErr := m.IssueToken(user)
	if tokenErr != nil {
		return nil, "", tokenErr
	}
	return user, token, nil
}

func CurrentUser(c *core.RequestEvent) (*core.Record, error) {
	if c.Auth == nil {
		return nil, ErrUnauthorized
	}
	return c.Auth, nil
}
