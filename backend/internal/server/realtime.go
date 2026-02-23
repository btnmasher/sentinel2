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
const realtimeKeepaliveInterval = 30 * time.Second

func registerRealtime(app *pocketbase.PocketBase, intelService *intel.IntelService, publisher *realtime.Publisher) {
	registerUploaderConfigTopicAuth(app, intelService)
	registerIntelUploadersTopic(app, intelService)
	registerKeepaliveTopicAuth(app)
	registerUploaderConfigBroadcasts(app, intelService, publisher)
	startIntelUploaderHeartbeatCountBroadcasts(app, intelService, publisher)
	startRealtimeKeepaliveBroadcasts(app, publisher)
}

func registerUploaderConfigTopicAuth(app *pocketbase.PocketBase, intelService *intel.IntelService) {
	app.OnRealtimeSubscribeRequest().BindFunc(func(e *core.RealtimeSubscribeRequestEvent) error {
		if !subscriptionContainsTopic(e.Subscriptions, realtime.TopicUploaderConfig) {
			return e.Next()
		}

		if forbiddenMessage := uploaderConfigForbiddenReason(e.Auth, intelService); forbiddenMessage != "" {
			return e.ForbiddenError(forbiddenMessage, nil)
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
		if !subscriptionContainsTopic(e.Subscriptions, realtime.TopicIntelUploaders) {
			return e.Next()
		}

		if e.Auth == nil {
			return e.ForbiddenError("Intel uploader realtime subscription requires authentication.", nil)
		}

		if intelService != nil {
			sendIntelUploaderCountSnapshot(e, intelService)
		}

		return e.Next()
	})
}

func registerKeepaliveTopicAuth(app *pocketbase.PocketBase) {
	app.OnRealtimeSubscribeRequest().BindFunc(func(e *core.RealtimeSubscribeRequestEvent) error {
		if !subscriptionContainsTopic(e.Subscriptions, realtime.TopicKeepalive) {
			return e.Next()
		}
		if e.Auth == nil {
			return e.ForbiddenError("Realtime keepalive subscription requires authentication.", nil)
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
			runIntelUploaderHeartbeatLoop(app, intelService, publisher, ticker.C)
		}()

		return nil
	})
}

func startRealtimeKeepaliveBroadcasts(app *pocketbase.PocketBase, publisher *realtime.Publisher) {
	if app == nil || publisher == nil {
		return
	}

	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		ticker := time.NewTicker(realtimeKeepaliveInterval)
		go func() {
			defer ticker.Stop()
			runRealtimeKeepaliveLoop(app, publisher, ticker.C)
		}()

		return nil
	})
}

func subscriptionContainsTopic(topicSubs []string, topic string) bool {
	for _, subscription := range topicSubs {
		if normalizeSubscriptionName(subscription) == topic {
			return true
		}
	}
	return false
}

func normalizeSubscriptionName(subscription string) string {
	name := strings.TrimSpace(subscription)
	if idx := strings.Index(name, "?"); idx >= 0 {
		name = name[:idx]
	}
	return name
}

func uploaderConfigForbiddenReason(authRecord *core.Record, intelService *intel.IntelService) string {
	if authRecord == nil || authRecord.Collection() == nil || authRecord.Collection().Name != store.CollectionUploaderSessions {
		return "Uploader realtime subscription requires uploader session auth."
	}
	if authRecord.GetString("scope") != intel.UploaderSessionScopeConfig {
		return "Invalid uploader session scope."
	}
	expiresAt := authRecord.GetDateTime("expires_at")
	if expiresAt.IsZero() || expiresAt.Time().Before(time.Now()) {
		return "Uploader realtime session expired."
	}
	if intelService == nil {
		return ""
	}
	uploaderTokenID := strings.TrimSpace(authRecord.GetString("uploader_token"))
	if uploaderTokenID == "" {
		return "Uploader realtime session missing token linkage."
	}
	if _, tokenErr := intelService.ValidateUploaderTokenID(uploaderTokenID); tokenErr != nil {
		return "Uploader token revoked."
	}
	return ""
}

func sendIntelUploaderCountSnapshot(e *core.RealtimeSubscribeRequestEvent, intelService *intel.IntelService) {
	count, countErr := intelService.UploaderCount()
	if countErr != nil {
		return
	}
	payload, marshalErr := json.Marshal(map[string]any{"uploaders": count})
	if marshalErr != nil {
		return
	}
	e.Client.Send(subscriptions.Message{
		Name: realtime.TopicIntelUploaders,
		Data: payload,
	})
}

func runIntelUploaderHeartbeatLoop(app *pocketbase.PocketBase, intelService *intel.IntelService, publisher *realtime.Publisher, ticks <-chan time.Time) {
	lastCount := -1
	for range ticks {
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
}

func publishIntelUploaderCount(publisher *realtime.Publisher, count int) error {
	_, err := publisher.PublishJSON(realtime.TopicIntelUploaders, map[string]any{"uploaders": count})
	return err
}

func runRealtimeKeepaliveLoop(app *pocketbase.PocketBase, publisher *realtime.Publisher, ticks <-chan time.Time) {
	for tick := range ticks {
		if _, err := publisher.PublishJSON(realtime.TopicKeepalive, map[string]any{
			"ts": tick.UTC().Unix(),
		}); err != nil {
			logging.New(app).
				WithErr(err).
				Warn("realtime keepalive publish failed")
		}
	}
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
