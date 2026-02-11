package auth

import (
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/tools/security"
)

const (
	authFlowTTL     = 10 * time.Minute
	exchangeCodeTTL = 5 * time.Minute
	pbTokenTTL      = 15 * time.Minute
)

func PBTokenTTL() time.Duration {
	return pbTokenTTL
}

type AuthFlowType string

const (
	FlowLogin AuthFlowType = "login"
	FlowLink  AuthFlowType = "link"
)

type AuthFlow struct {
	Type       AuthFlowType
	LinkUserID string
	CreatedAt  time.Time
}

type authExchange struct {
	UserID    string
	CreatedAt time.Time
}

func newExchangeCode() string {
	return security.RandomString(48)
}

func flowKey(state string) string {
	return "auth_flow:" + state
}

func exchangeKey(code string) string {
	return "auth_exchange:" + code
}

func saveAuthFlow(app *pocketbase.PocketBase, state string, flow AuthFlow) {
	if app == nil || state == "" {
		return
	}
	flow.CreatedAt = time.Now()
	app.Store().Set(flowKey(state), flow)
}

func loadAuthFlow(app *pocketbase.PocketBase, state string) (AuthFlow, bool) {
	if app == nil || state == "" {
		return AuthFlow{}, false
	}
	raw, ok := app.Store().GetOk(flowKey(state))
	if !ok {
		return AuthFlow{}, false
	}
	flow, ok := raw.(AuthFlow)
	if !ok {
		app.Store().Remove(flowKey(state))
		return AuthFlow{}, false
	}
	if time.Since(flow.CreatedAt) > authFlowTTL {
		app.Store().Remove(flowKey(state))
		return AuthFlow{}, false
	}
	return flow, true
}

func deleteAuthFlow(app *pocketbase.PocketBase, state string) {
	if app == nil || state == "" {
		return
	}
	app.Store().Remove(flowKey(state))
}

func saveAuthExchange(app *pocketbase.PocketBase, userID string) string {
	if app == nil || userID == "" {
		return ""
	}
	code := newExchangeCode()
	app.Store().Set(exchangeKey(code), authExchange{
		UserID:    userID,
		CreatedAt: time.Now(),
	})
	return code
}

func loadAuthExchange(app *pocketbase.PocketBase, code string) (authExchange, bool) {
	if app == nil || code == "" {
		return authExchange{}, false
	}
	raw, ok := app.Store().GetOk(exchangeKey(code))
	if !ok {
		return authExchange{}, false
	}
	ex, ok := raw.(authExchange)
	if !ok {
		app.Store().Remove(exchangeKey(code))
		return authExchange{}, false
	}
	if time.Since(ex.CreatedAt) > exchangeCodeTTL {
		app.Store().Remove(exchangeKey(code))
		return authExchange{}, false
	}
	return ex, true
}

func deleteAuthExchange(app *pocketbase.PocketBase, code string) {
	if app == nil || code == "" {
		return
	}
	app.Store().Remove(exchangeKey(code))
}
