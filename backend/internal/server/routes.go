package server

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"sentinel2/internal/api/appconfig"
	"sentinel2/internal/config"
	"sentinel2/internal/middleware"
	"sentinel2/internal/web"
)

const (
	intelRetrievePerHour  = 180
	mapRoutePerMinute     = 30
	mapCharactersPerHour  = 150
	mapLocationPerHour    = 180
	mapSearchPerMinute    = 100
	mapTopRoutesPerMinute = 10
	defaultLimiterBurst   = 5
	mapCharactersBurst    = 25
	intelRetrieveBurst    = 20
	locationLimiterBurst  = 10
	searchLimiterBurst    = 20
)

//nolint:gocognit // route registration is intentionally centralized.
func registerRoutes(app *pocketbase.PocketBase, cfg *config.Config, deps *dependencies) {
	userKey := func(c *core.RequestEvent) string {
		if c.Auth != nil {
			return c.Auth.Id
		}
		return ""
	}

	intelRetrieveLimiter := middleware.NewRateLimiter(middleware.LimitPerHour(intelRetrievePerHour), intelRetrieveBurst)
	mapRouteLimiter := middleware.NewRateLimiter(middleware.LimitPerMinute(mapRoutePerMinute), defaultLimiterBurst)
	mapCharactersLimiter := middleware.NewRateLimiter(middleware.LimitPerHour(mapCharactersPerHour), mapCharactersBurst)
	mapLocationLimiter := middleware.NewRateLimiter(middleware.LimitPerHour(mapLocationPerHour), locationLimiterBurst)
	mapSearchLimiter := middleware.NewRateLimiter(middleware.LimitPerMinute(mapSearchPerMinute), searchLimiterBurst)
	mapTopRoutesLimiter := middleware.NewRateLimiter(middleware.LimitPerMinute(mapTopRoutesPerMinute), defaultLimiterBurst)

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.Unbind(apis.DefaultActivityLoggerMiddlewareId)
		e.Router.Bind(middleware.ActivityLoggerWithMeta())

		e.Router.BindFunc(middleware.ContentSecurityPolicy(cfg.FrontendDevProxy != ""))

		e.Router.BindFunc(
			func(c *core.RequestEvent) error {
				c.Set("sentinel_version", cfg.SentinelVersion)
				return c.Next()
			},
		)

		apiGroup := e.Router.Group("/api")
		apiGroup.GET("/app-config", appconfig.AppConfig(cfg))

		authGroup := apiGroup.Group("/auth")
		authGroup.GET("/login", deps.authHandler.Authenticate)
		authGroup.GET("/callback", deps.authHandler.Callback)
		authGroup.GET("/logout", deps.authHandler.Logout)
		authGroup.GET("/me", deps.authHandler.Me).BindFunc(middleware.RequireAuth)
		authGroup.GET("/exchange", deps.authHandler.Exchange)
		authGroup.GET("/characters/linkable", deps.authHandler.LinkableCharacters).BindFunc(middleware.RequireAuth)
		authGroup.POST("/characters", deps.authHandler.LinkCharacters).BindFunc(middleware.RequireAuth)
		authGroup.POST("/refresh", deps.authHandler.Refresh).BindFunc(middleware.RequireAuth)
		authGroup.DELETE("/characters/{id}", deps.authHandler.RemoveCharacter).BindFunc(middleware.RequireAuth)
		authGroup.GET("/uploader-token", deps.intelHandler.GetUploaderToken).BindFunc(middleware.RequireAuth, middleware.RequireMainCharacter(app))
		authGroup.POST("/uploader-token/rotate", deps.intelHandler.RotateUploaderToken).BindFunc(middleware.RequireAuth, middleware.RequireMainCharacter(app))
		authGroup.GET("/profile", deps.authHandler.Profile).BindFunc(middleware.RequireAuth)
		if cfg.AuthBackend == "eve" {
			authGroup.GET("/link", deps.authHandler.Link).BindFunc(middleware.RequireAuth)
		}

		testAuthGroup := apiGroup.Group("/testauth")
		testAuthGroup.GET("/authenticate", deps.authHandler.Authenticate)
		testAuthGroup.GET("/callback", deps.authHandler.Callback)
		testAuthGroup.GET("/logout", deps.authHandler.Logout)

		intelGroup := apiGroup.Group("/intel")
		intelGroup.GET("/reports", deps.intelHandler.ListReports).BindFunc(middleware.RequireAuth, middleware.RequireMainCharacter(app), middleware.RateLimit(intelRetrieveLimiter, userKey))

		organizationGroup := apiGroup.Group("/organizations")
		organizationGroup.BindFunc(middleware.RequireAuth, middleware.RequireMainCharacter(app))
		organizationGroup.GET("/search", deps.timers.SearchEntities)

		if cfg.TimersEnabled {
			timerGroup := apiGroup.Group("/timers")
			timerGroup.BindFunc(middleware.RequireAuth, middleware.RequireMainCharacter(app))
			timerGroup.GET("", deps.timers.List)
			if cfg.TimerSource == config.TimerSourceStandalone {
				timerGroup.GET("/systems", deps.timers.SearchSystems)
				timerGroup.GET("/planets", deps.timers.SearchPlanets)
				timerGroup.GET("/moons", deps.timers.SearchMoons)
				timerGroup.POST("", deps.timers.Create)
				timerGroup.PATCH("/{id}", deps.timers.Update).BindFunc(middleware.RequireStaff(app))
				timerGroup.POST("/{id}/cancel", deps.timers.Cancel).BindFunc(middleware.RequireStaff(app))
				timerGroup.POST("/{id}/uncancel", deps.timers.Uncancel).BindFunc(middleware.RequireStaff(app))
				timerGroup.DELETE("/{id}", deps.timers.Delete).BindFunc(middleware.RequireStaff(app))
				timerGroup.POST("/parse", deps.timers.Parse)
			}
		}

		if cfg.TimersEnabled && cfg.TimerSource == config.TimerSourceWebhook {
			webhookTimerGroup := apiGroup.Group("/webhooks/timers")
			webhookTimerGroup.BindFunc(middleware.RequireTimersWebhookToken(cfg))
			webhookTimerGroup.POST("", deps.timerWebhook.Create)
			webhookTimerGroup.PATCH("/{id}", deps.timerWebhook.Patch)
			webhookTimerGroup.DELETE("/{id}", deps.timerWebhook.Delete)
		}

		uploaderGroup := apiGroup.Group("/uploader")
		uploaderGroup.GET("/download-links", deps.uploaderHandler.DownloadLinks).BindFunc(middleware.RequireAuth, middleware.RequireMainCharacter(app))

		uploaderAgentGroup := apiGroup.Group("/uploader")
		uploaderAgentGroup.BindFunc(middleware.RequireUploaderToken(deps.intelService))
		uploaderAgentGroup.POST("/realtime/token", deps.intelHandler.UploaderRealtimeToken)

		uploaderSessionGroup := apiGroup.Group("/uploader")
		uploaderSessionGroup.BindFunc(
			middleware.RequireAuth,
			middleware.RequireUploaderRealtimeSession(deps.intelService),
		)
		uploaderSessionGroup.PUT("/submit", deps.intelHandler.Submit)
		uploaderSessionGroup.GET("/config", deps.intelHandler.UploaderConfig)
		uploaderSessionGroup.POST("/heartbeat", deps.intelHandler.Heartbeat)
		uploaderSessionGroup.POST("/session/refresh", deps.intelHandler.UploaderSessionRefresh)

		mapGroup := apiGroup.Group("/map")
		mapGroup.BindFunc(middleware.RequireAuth, middleware.RequireMainCharacter(app))
		mapGroup.GET("/regions/{regions}/dotlan", deps.mapHandler.RegionsDotlan)
		mapGroup.GET("/regions/{regions}/metro", deps.mapHandler.RegionsMetro)
		mapGroup.GET("/regions/{regions}/real", deps.mapHandler.RegionsReal)
		mapGroup.GET("/regions/{regions}/eve2d", deps.mapHandler.RegionsEve2D)
		mapGroup.GET("/regions/{regions}/overlays", deps.mapHandler.RegionOverlays)
		mapGroup.GET("/characters", deps.mapHandler.Characters).BindFunc(middleware.RateLimit(mapCharactersLimiter, userKey))
		mapGroup.POST("/locations", deps.mapHandler.CharacterLocations).BindFunc(middleware.RateLimit(mapLocationLimiter, userKey))
		mapGroup.POST("/route/{character}", deps.mapHandler.Route).BindFunc(middleware.RateLimit(mapRouteLimiter, userKey))
		mapGroup.DELETE("/route/{character}", deps.mapHandler.ClearRoute).BindFunc(middleware.RateLimit(mapRouteLimiter, userKey))
		mapGroup.GET("/search", deps.mapHandler.Search).BindFunc(middleware.RateLimit(mapSearchLimiter, userKey))
		mapGroup.GET("/top-routes", deps.mapHandler.TopRoutes).BindFunc(middleware.RateLimit(mapTopRoutesLimiter, userKey))

		staffGroup := apiGroup.Group("/staff")
		staffGroup.BindFunc(middleware.RequireAuth, middleware.RequireMainCharacter(app), middleware.RequireStaff(app))
		staffGroup.GET("/channels", deps.staffChannels.List)
		staffGroup.POST("/channels", deps.staffChannels.Create)
		staffGroup.DELETE("/channels/{id}", deps.staffChannels.Delete)
		staffGroup.GET("/organization-standings", deps.staffOrgStandings.List)
		staffGroup.POST("/organization-standings", deps.staffOrgStandings.Create)
		staffGroup.PATCH("/organization-standings/{id}", deps.staffOrgStandings.Update)
		staffGroup.DELETE("/organization-standings/{id}", deps.staffOrgStandings.Delete)
		staffGroup.POST("/jumpbridges/import", deps.staffJumpbridges.Import)
		staffGroup.POST("/jumpbridges/clear", deps.staffJumpbridges.Clear)
		staffGroup.POST("/jumpbridges/add", deps.staffJumpbridges.Add)
		staffGroup.POST("/jumpbridges/remove", deps.staffJumpbridges.Remove)
		staffGroup.POST("/jumpbridges/update", deps.staffJumpbridges.Update)

		adminGroup := apiGroup.Group("/admin")
		adminGroup.BindFunc(middleware.RequireAdmin(app), middleware.RequireMainCharacter(app))
		adminGroup.GET("/search", deps.admin.Search)
		adminGroup.GET("/users/{id}", deps.admin.UserDetails)
		adminGroup.GET("/audit", deps.admin.AuditLogs)
		adminGroup.GET("/job-runs", deps.admin.JobRuns)
		adminGroup.POST("/users/{id}/main", deps.admin.SetMainCharacter)
		adminGroup.POST("/users/{id}/resync", deps.admin.ResyncAccount)
		adminGroup.POST("/users/{id}/access-level", deps.admin.SetAccessLevel)
		adminGroup.POST("/users/{id}/revoke-sessions", deps.admin.RevokeSessions)
		adminGroup.POST("/users/{id}/revoke-upload-tokens", deps.admin.RevokeUploaderTokens)
		adminGroup.POST("/users/{id}/regenerate-upload-token", deps.admin.RegenerateUploaderToken)
		adminGroup.POST("/users/{id}/merge", deps.admin.MergeUsers)
		adminGroup.POST("/map-data/run", deps.adminMapDataUpdate.RunAll)
		adminGroup.POST("/map-data/sde", deps.adminMapDataUpdate.RunSDEImport)
		adminGroup.POST("/map-data/real-positions", deps.adminMapDataUpdate.RunRealPositions)
		adminGroup.POST("/map-data/eve2d-positions", deps.adminMapDataUpdate.RunEve2DPositions)
		adminGroup.POST("/map-data/dotlan", deps.adminMapDataUpdate.RunDotlan)
		adminGroup.POST("/map-data/planets", deps.adminMapDataUpdate.RunPlanets)
		adminGroup.POST("/map-data/moons", deps.adminMapDataUpdate.RunMoons)
		adminGroup.POST("/map-data/types", deps.adminMapDataUpdate.RunTypes)
		adminGroup.POST("/map-data/metro-positions", deps.adminMapDataUpdate.RunMetroPositions)
		adminGroup.POST("/map-data/system-graphs", deps.adminMapDataUpdate.RunSystemGraphs)
		adminGroup.POST("/map-data/region-layout", deps.adminMapDataUpdate.RunRegionLayout)
		adminGroup.POST("/characters/refresh-all", deps.admin.RefreshAllCharacters)
		adminGroup.POST("/jobs/cleanup", deps.admin.RunCleanupJob)
		adminGroup.POST("/jobs/jumpbridges/update", deps.admin.RunUpdateJumpbridgesJob)
		if cfg.TimersEnabled && cfg.TimerSource == config.TimerSourceStandalone {
			adminGroup.POST("/jobs/timers/sovereignty-campaign-sync", deps.admin.RunSovereigntyCampaignSyncJob)
			adminGroup.POST("/jobs/timers/structure-notifications-sync", deps.admin.RunStructureNotificationsSyncJob)
		}
		adminGroup.POST("/announcement", deps.admin.CreateSiteAnnouncement)
		adminGroup.POST("/announcement/archive-latest", deps.admin.ArchiveLatestSiteAnnouncement)
		adminGroup.GET("/site-settings/allowed-organizations", deps.admin.ListAllowedOrganizations)
		adminGroup.POST("/site-settings/allowed-organizations", deps.admin.UpsertAllowedOrganization)
		adminGroup.DELETE("/site-settings/allowed-organizations/{type}/{eve_id}", deps.admin.DeleteAllowedOrganization)
		adminGroup.POST("/characters/{id}/refresh", deps.admin.RefreshCharacter)
		adminGroup.POST("/characters/{id}/revoke", deps.admin.RevokeCharacterTokens)
		adminGroup.POST("/characters/{id}/move", deps.admin.MoveCharacter)
		adminGroup.DELETE("/characters/{id}", deps.admin.RemoveCharacter)
		adminGroup.POST("/jobs/{id}/cancel", deps.admin.CancelJob)

		if cfg.FrontendDevProxy != "" {
			_ = web.MountFrontendProxy(e.Router, cfg.FrontendDevProxy)
		} else {
			web.MountFrontend(e.Router)
		}

		return e.Next()
	})
}
