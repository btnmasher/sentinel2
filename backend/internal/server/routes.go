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

func registerRoutes(app *pocketbase.PocketBase, cfg config.Config, deps dependencies) {
	userKey := func(c *core.RequestEvent) string {
		if c.Auth != nil {
			return c.Auth.Id
		}
		return ""
	}

	intelRetrieveLimiter := middleware.NewRateLimiter(middleware.LimitPerHour(40), 5)
	mapRouteLimiter := middleware.NewRateLimiter(middleware.LimitPerMinute(30), 5)
	mapCharactersLimiter := middleware.NewRateLimiter(middleware.LimitPerHour(30), 5)
	mapLocationLimiter := middleware.NewRateLimiter(middleware.LimitPerHour(180), 10)
	mapSearchLimiter := middleware.NewRateLimiter(middleware.LimitPerMinute(100), 20)
	mapTopRoutesLimiter := middleware.NewRateLimiter(middleware.LimitPerMinute(10), 5)

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.Unbind(apis.DefaultActivityLoggerMiddlewareId)
		e.Router.Bind(middleware.ActivityLoggerWithMeta())

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
		authGroup.POST("/refresh", deps.authHandler.Refresh).BindFunc(middleware.RequireAuth)
		authGroup.GET("/uploader-token", deps.intelHandler.GetUploaderToken).BindFunc(middleware.RequireAuth, middleware.RequireMainCharacter(app))
		authGroup.POST("/uploader-token/rotate", deps.intelHandler.RotateUploaderToken).BindFunc(middleware.RequireAuth, middleware.RequireMainCharacter(app))
		if cfg.AuthBackend == "eve" {
			authGroup.GET("/profile", deps.authHandler.Profile).BindFunc(middleware.RequireAuth)
			authGroup.GET("/link", deps.authHandler.Link).BindFunc(middleware.RequireAuth)
		}

		testAuthGroup := apiGroup.Group("/testauth")
		testAuthGroup.GET("/authenticate", deps.authHandler.Authenticate)
		testAuthGroup.GET("/callback", deps.authHandler.Callback)
		testAuthGroup.GET("/logout", deps.authHandler.Logout)

		intelGroup := apiGroup.Group("/intel")
		intelGroup.GET("/reports", deps.intelHandler.ListReports).BindFunc(middleware.RequireAuth, middleware.RequireMainCharacter(app), middleware.RateLimit(intelRetrieveLimiter, userKey))

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
		staffGroup.POST("/jumpbridges/import", deps.staffJumpbridges.Import)
		staffGroup.POST("/jumpbridges/clear", deps.staffJumpbridges.Clear)
		staffGroup.POST("/jumpbridges/add", deps.staffJumpbridges.Add)
		staffGroup.POST("/jumpbridges/remove", deps.staffJumpbridges.Remove)
		staffGroup.POST("/jumpbridges/update", deps.staffJumpbridges.Update)

		if cfg.AuthBackend == "eve" {
			adminGroup := apiGroup.Group("/admin")
			adminGroup.BindFunc(middleware.RequireAdmin(app), middleware.RequireMainCharacter(app))
			adminGroup.GET("/search", deps.admin.Search)
			adminGroup.GET("/users/{id}", deps.admin.UserDetails)
			adminGroup.GET("/audit", deps.admin.AuditLogs)
			adminGroup.GET("/job-runs", deps.admin.JobRuns)
			adminGroup.POST("/users/{id}/main", deps.admin.SetMainCharacter)
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
			adminGroup.POST("/map-data/metro-positions", deps.adminMapDataUpdate.RunMetroPositions)
			adminGroup.POST("/map-data/system-graphs", deps.adminMapDataUpdate.RunSystemGraphs)
			adminGroup.POST("/map-data/region-layout", deps.adminMapDataUpdate.RunRegionLayout)
			adminGroup.POST("/characters/refresh-all", deps.admin.RefreshAllCharacters)
			adminGroup.POST("/jobs/cleanup", deps.admin.RunCleanupJob)
			if cfg.DebugEnabled {
				adminGroup.POST("/debug/seed-search-users", deps.admin.SeedSearchUsers)
			}
			adminGroup.POST("/characters/{id}/refresh", deps.admin.RefreshCharacter)
			adminGroup.POST("/characters/{id}/revoke", deps.admin.RevokeCharacterTokens)
			adminGroup.POST("/characters/{id}/move", deps.admin.MoveCharacter)
			adminGroup.DELETE("/characters/{id}", deps.admin.RemoveCharacter)
			adminGroup.POST("/jobs/{id}/cancel", deps.admin.CancelJob)
		}

		if cfg.FrontendDevProxy != "" {
			_ = web.MountFrontendProxy(e.Router, cfg.FrontendDevProxy)
		} else {
			web.MountFrontend(e.Router)
		}

		return e.Next()
	})
}
