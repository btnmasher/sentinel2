package middleware

import (
	"errors"
	"log/slog"
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
	if event.App.Settings().Logs.MaxDays == 0 {
		return
	}

	if err == nil && event.Get(requestEventKeySkipSuccessActivityLog) != nil {
		return
	}

	meta := map[string]any{}
	if existing := event.Get(apis.RequestEventKeyLogMeta); existing != nil {
		if data, ok := existing.(map[string]any); ok {
			for key, value := range data {
				meta[key] = value
			}
		}
	}

	if event.Auth != nil {
		meta["auth_provider"] = event.Auth.GetString("auth_provider")
		meta["character_id"] = event.Auth.GetInt("eve_character_id")
		meta["character_name"] = event.Auth.GetString("eve_character_name")
		meta["access_level"] = event.Auth.GetString("access_level")
	}

	if len(meta) > 0 {
		event.Set(apis.RequestEventKeyLogMeta, meta)
	}

	attrs := make([]any, 0, 15)
	attrs = append(attrs, slog.String("type", "request"))

	started := cast.ToTime(event.Get(requestEventKeyExecStart))
	if !started.IsZero() {
		attrs = append(attrs, slog.Float64("execTime", float64(time.Since(started))/float64(time.Millisecond)))
	}

	if metaValue := event.Get(apis.RequestEventKeyLogMeta); metaValue != nil {
		attrs = append(attrs, slog.Any("meta", metaValue))
	}

	status := event.Status()
	method := cutStr(strings.ToUpper(event.Request.Method), 50)
	requestUri := cutStr(event.Request.URL.RequestURI(), 3000)

	if err != nil {
		apiErr, isPlainApiError := err.(*router.ApiError)
		if isPlainApiError || errors.As(err, &apiErr) {
			if status == 0 {
				status = apiErr.Status
			}

			var errMsg string
			if isPlainApiError {
				errMsg = apiErr.Message
			} else {
				errMsg = err.Error()
			}

			attrs = append(
				attrs,
				slog.String("error", errMsg),
				slog.Any("details", apiErr.RawData()),
			)
		} else {
			attrs = append(attrs, slog.String("error", err.Error()))
		}
	}

	attrs = append(
		attrs,
		slog.String("url", requestUri),
		slog.String("method", method),
		slog.Int("status", status),
		slog.String("referer", cutStr(event.Request.Referer(), 2000)),
		slog.String("userAgent", cutStr(event.Request.UserAgent(), 2000)),
	)

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

	routine.FireAndForget(func() {
		message := method + " "

		if escaped, unescapeErr := url.PathUnescape(requestUri); unescapeErr == nil {
			message += escaped
		} else {
			message += requestUri
		}

		if err != nil {
			event.App.Logger().Error(message, attrs...)
		} else {
			event.App.Logger().Info(message, attrs...)
		}
	})
}

func cutStr(value string, max int) string {
	if len(value) > max {
		return value[:max] + "..."
	}
	return value
}
