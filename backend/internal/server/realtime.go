package server

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/subscriptions"

	"sentinel2/internal/intel"
	"sentinel2/internal/logging"
	"sentinel2/internal/realtime"
	"sentinel2/internal/store"
)

const uploaderCountRecomputeInterval = 30 * time.Second

func registerRealtime(app *pocketbase.PocketBase, intelService *intel.IntelService, publisher *realtime.Publisher) {
	registerUploaderConfigTopicAuth(app, intelService)
	registerIntelUploadersTopic(app, intelService)
	registerUploaderConfigBroadcasts(app, intelService, publisher)
	startIntelUploaderHeartbeatCountBroadcasts(app, intelService, publisher)
}

func registerUploaderConfigTopicAuth(app *pocketbase.PocketBase, intelService *intel.IntelService) {
	app.OnRealtimeSubscribeRequest().BindFunc(func(e *core.RealtimeSubscribeRequestEvent) error {
		requiresUploaderSession := false
		for _, subscription := range e.Subscriptions {
			name := strings.TrimSpace(subscription)
			if idx := strings.Index(name, "?"); idx >= 0 {
				name = name[:idx]
			}
			if name == realtime.TopicUploaderConfig {
				requiresUploaderSession = true
				break
			}
		}
		if !requiresUploaderSession {
			return e.Next()
		}

		authRecord := e.Auth
		if authRecord == nil || authRecord.Collection() == nil || authRecord.Collection().Name != store.CollectionUploaderSessions {
			return e.ForbiddenError("Uploader realtime subscription requires uploader session auth.", nil)
		}
		if authRecord.GetString("scope") != intel.UploaderSessionScopeConfig {
			return e.ForbiddenError("Invalid uploader session scope.", nil)
		}
		expiresAt := authRecord.GetDateTime("expires_at")
		if expiresAt.IsZero() || expiresAt.Time().Before(time.Now()) {
			return e.ForbiddenError("Uploader realtime session expired.", nil)
		}
		if intelService != nil {
			uploaderTokenID := strings.TrimSpace(authRecord.GetString("uploader_token"))
			if uploaderTokenID == "" {
				return e.ForbiddenError("Uploader realtime session missing token linkage.", nil)
			}
			if _, tokenErr := intelService.ValidateUploaderTokenID(uploaderTokenID); tokenErr != nil {
				return e.ForbiddenError("Uploader token revoked.", nil)
			}
		}

		return e.Next()
	})
}

func registerUploaderConfigBroadcasts(app *pocketbase.PocketBase, intelService *intel.IntelService, publisher *realtime.Publisher) {
	broadcast := func(e *core.ModelEvent) error {
		record := resolveRecord(e.Model)
		if record == nil || record.Collection() == nil || record.Collection().Name != store.CollectionIntelChannels {
			return e.Next()
		}

		if intelService == nil {
			return e.Next()
		}
		config, configErr := intelService.UploaderConfig()
		if configErr != nil {
			logging.New(app).
				WithErr(configErr).
				Error("uploader config realtime build failed")
			return e.Next()
		}

		if publisher == nil {
			return e.Next()
		}
		if _, publishErr := publisher.PublishJSON(realtime.TopicUploaderConfig, config); publishErr != nil {
			logging.New(app).
				WithErr(publishErr).
				Warn("uploader config realtime publish failed")
		}

		return e.Next()
	}

	app.OnModelAfterCreateSuccess().BindFunc(broadcast)
	app.OnModelAfterUpdateSuccess().BindFunc(broadcast)
	app.OnModelAfterDeleteSuccess().BindFunc(broadcast)
}

func registerIntelUploadersTopic(app *pocketbase.PocketBase, intelService *intel.IntelService) {
	app.OnRealtimeSubscribeRequest().BindFunc(func(e *core.RealtimeSubscribeRequestEvent) error {
		requiresAuth := false
		for _, subscription := range e.Subscriptions {
			name := strings.TrimSpace(subscription)
			if idx := strings.Index(name, "?"); idx >= 0 {
				name = name[:idx]
			}
			if name == realtime.TopicIntelUploaders {
				requiresAuth = true
				break
			}
		}
		if !requiresAuth {
			return e.Next()
		}

		if e.Auth == nil {
			return e.ForbiddenError("Intel uploader realtime subscription requires authentication.", nil)
		}

		if intelService != nil {
			count, countErr := intelService.UploaderCount()
			if countErr == nil {
				payload, marshalErr := json.Marshal(map[string]any{"uploaders": count})
				if marshalErr == nil {
					e.Client.Send(subscriptions.Message{
						Name: realtime.TopicIntelUploaders,
						Data: payload,
					})
				}
			}
		}

		return e.Next()
	})
}

func startIntelUploaderHeartbeatCountBroadcasts(app *pocketbase.PocketBase, intelService *intel.IntelService, publisher *realtime.Publisher) {
	if app == nil || intelService == nil || publisher == nil {
		return
	}

	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		ticker := time.NewTicker(uploaderCountRecomputeInterval)
		go func() {
			defer ticker.Stop()

			lastCount := -1
			for range ticker.C {
				count, countErr := intelService.UploaderCount()
				if countErr != nil {
					logging.New(app).
						WithErr(countErr).
						Warn("intel uploader heartbeat count build failed")
					continue
				}
				if count == lastCount {
					continue
				}
				if publishErr := publishIntelUploaderCount(publisher, count); publishErr != nil {
					logging.New(app).
						WithErr(publishErr).
						Warn("intel uploader heartbeat count publish failed")
					continue
				}
				lastCount = count
			}
		}()

		return nil
	})
}

func publishIntelUploaderCount(publisher *realtime.Publisher, count int) error {
	_, err := publisher.PublishJSON(realtime.TopicIntelUploaders, map[string]any{"uploaders": count})
	return err
}

func resolveRecord(model core.Model) *core.Record {
	if record, ok := model.(*core.Record); ok {
		return record
	}
	if proxy, ok := model.(core.RecordProxy); ok {
		return proxy.ProxyRecord()
	}
	return nil
}
