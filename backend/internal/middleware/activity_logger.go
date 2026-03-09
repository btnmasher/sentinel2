package middleware

import (
	"errors"
	"log/slog"
	"maps"
	"net/url"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/routine"
	"github.com/spf13/cast"
)

const (
	requestEventKeyExecStart              = "__execStart"
	requestEventKeySkipSuccessActivityLog = "__skipSuccessActivityLogger"
	requestAttrCapacity                   = 15
	maxMethodLen                          = 50
	maxRequestURILen                      = 3000
	maxHeaderValueLen                     = 2000
)

// ActivityLoggerWithMeta wraps the default PocketBase activity logger
// so we can enrich request logs with auth context.
func ActivityLoggerWithMeta() *hook.Handler[*core.RequestEvent] {
	return &hook.Handler[*core.RequestEvent]{
		Id:       apis.DefaultActivityLoggerMiddlewareId,
		Priority: apis.DefaultLoadAuthTokenMiddlewarePriority + 1,
		Func: func(e *core.RequestEvent) error {
			e.Set(requestEventKeyExecStart, time.Now())

			err := e.Next()

			logRequestWithMeta(e, err)

			return err
		},
	}
}

func logRequestWithMeta(event *core.RequestEvent, err error) {
	if shouldSkipActivityLog(event, err) {
		return
	}

	meta := buildRequestMeta(event)
	if len(meta) > 0 {
		event.Set(apis.RequestEventKeyLogMeta, meta)
	}

	attrs := make([]any, 0, requestAttrCapacity)
	attrs = append(attrs, slog.String("type", "request"))
	attrs = appendExecAndMetaAttrs(attrs, event)
	status := event.Status()
	method := cutStr(strings.ToUpper(event.Request.Method), maxMethodLen)
	requestURI := cutStr(event.Request.URL.RequestURI(), maxRequestURILen)
	status, attrs = appendRequestErrorAttrs(status, attrs, err)
	attrs = appendCommonRequestAttrs(attrs, event, method, requestURI, status)
	attrs = appendAuthAndIPAttrs(attrs, event)
	logRequestAsync(event, err, method, requestURI, attrs)
}

func shouldSkipActivityLog(event *core.RequestEvent, err error) bool {
	if event.App.Settings().Logs.MaxDays == 0 {
		return true
	}
	return err == nil && event.Get(requestEventKeySkipSuccessActivityLog) != nil
}

func buildRequestMeta(event *core.RequestEvent) map[string]any {
	meta := map[string]any{}

	if existing := event.Get(apis.RequestEventKeyLogMeta); existing != nil {
		if data, ok := existing.(map[string]any); ok {
			maps.Copy(meta, data)
		}
	}

	if event.Auth != nil {
		meta["auth_provider"] = event.Auth.GetString("auth_provider")
		meta["character_id"] = event.Auth.GetInt("eve_character_id")
		meta["character_name"] = event.Auth.GetString("eve_character_name")
		meta["access_level"] = event.Auth.GetString("access_level")
	}
	return meta
}

func appendExecAndMetaAttrs(attrs []any, event *core.RequestEvent) []any {
	started := cast.ToTime(event.Get(requestEventKeyExecStart))
	if !started.IsZero() {
		attrs = append(attrs, slog.Float64("execTime", float64(time.Since(started))/float64(time.Millisecond)))
	}

	if metaValue := event.Get(apis.RequestEventKeyLogMeta); metaValue != nil {
		attrs = append(attrs, slog.Any("meta", metaValue))
	}
	return attrs
}

func appendRequestErrorAttrs(status int, attrs []any, err error) (nextStatus int, nextAttrs []any) {
	if err == nil {
		return status, attrs
	}
	var apiErr *router.ApiError
	if !errors.As(err, &apiErr) {
		return status, append(attrs, slog.String("error", err.Error()))
	}

	if status == 0 {
		status = apiErr.Status
	}
	errMsg := apiErr.Message
	if errMsg == "" {
		errMsg = err.Error()
	}
	attrs = append(attrs, slog.String("error", errMsg), slog.Any("details", apiErr.RawData()))
	return status, attrs
}

func appendCommonRequestAttrs(attrs []any, event *core.RequestEvent, method, requestURI string, status int) []any {
	return append(
		attrs,
		slog.String("url", requestURI),
		slog.String("method", method),
		slog.Int("status", status),
		slog.String("referer", cutStr(event.Request.Referer(), maxHeaderValueLen)),
		slog.String("userAgent", cutStr(event.Request.UserAgent(), maxHeaderValueLen)),
	)
}

func appendAuthAndIPAttrs(attrs []any, event *core.RequestEvent) []any {
	if event.Auth != nil {
		attrs = append(attrs, slog.String("auth", event.Auth.Collection().Name))
		if event.App.Settings().Logs.LogAuthId {
			attrs = append(attrs, slog.String("authId", event.Auth.Id))
		}
	} else {
		attrs = append(attrs, slog.String("auth", ""))
	}

	if event.App.Settings().Logs.LogIP {
		attrs = append(
			attrs,
			slog.String("userIP", event.RealIP()),
			slog.String("remoteIP", event.RemoteIP()),
		)
	}
	return attrs
}

func logRequestAsync(event *core.RequestEvent, err error, method, requestURI string, attrs []any) {
	routine.FireAndForget(func() {
		message := method + " "
		if escaped, unescapeErr := url.PathUnescape(requestURI); unescapeErr == nil {
			message += escaped
		} else {
			message += requestURI
		}
		if err != nil {
			event.App.Logger().Error(message, attrs...)
			return
		}
		event.App.Logger().Info(message, attrs...)
	})
}

func cutStr(value string, maxLen int) string {
	if len(value) > maxLen {
		return value[:maxLen] + "..."
	}
	return value
}
