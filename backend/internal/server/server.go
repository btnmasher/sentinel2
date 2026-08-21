package server

import (
	"context"
	"fmt"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"golang.org/x/oauth2"

	"sentinel2/internal/audit"
	"sentinel2/internal/auth"
	"sentinel2/internal/cleanup"
	"sentinel2/internal/config"
	"sentinel2/internal/esi"
	"sentinel2/internal/intel"
	"sentinel2/internal/jumpbridges"
	"sentinel2/internal/logging"
	"sentinel2/internal/realtime"
	timerssvc "sentinel2/internal/timers"
	"sentinel2/internal/uploaderrelease"

	adminapi "sentinel2/internal/api/admin"
	authapi "sentinel2/internal/api/auth"
	intelapi "sentinel2/internal/api/intel"
	mapapi "sentinel2/internal/api/maps"
	staffapi "sentinel2/internal/api/staff"
	timerapi "sentinel2/internal/api/timers"
	timerwebhookapi "sentinel2/internal/api/timerwebhook"
	uploaderapi "sentinel2/internal/api/uploader"
)

type dependencies struct {
	audit              *audit.Service
	provider           auth.Provider
	authManager        *auth.Manager
	eveProvider        *auth.EVEProvider
	intelService       *intel.IntelService
	esiClient          esi.ESIClient
	publicESI          *esi.ESIPublicClient
	intelHandler       *intelapi.IntelHandler
	mapHandler         *mapapi.MapHandler
	authHandler        *authapi.AuthHandler
	cleanup            *cleanup.Service
	staffChannels      *staffapi.ChannelsHandler
	staffJumpbridges   *staffapi.JumpbridgeHandler
	staffOrgStandings  *staffapi.OrganizationStandingsHandler
	adminMapDataUpdate *adminapi.MapUpdateHandler
	characterRefresher *auth.CharacterRefresher
	admin              *adminapi.Handler
	timerService       *timerssvc.Service
	timers             *timerapi.Handler
	timerWebhook       *timerwebhookapi.Handler
	uploaderReleases   *uploaderrelease.Service
	uploaderHandler    *uploaderapi.Handler
	realtimePublisher  *realtime.Publisher
}

func Run(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("missing config")
	}
	if err := auth.ValidatePublicBaseURL(cfg.PublicBaseURL, cfg.DebugEnabled); err != nil {
		return fmt.Errorf("invalid public base URL: %w", err)
	}
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDev: cfg.DebugEnabled,
	})

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{Dir: "pb_migrations"})

	intelService := intel.NewIntelService(app)
	intelService.SetReportHashSlots(cfg.IntelReportHashSlots)

	publicESI := esi.NewESIPublicClient(cfg.ESIUserAgent)
	provider, providerErr := buildAuthProvider(cfg, app, publicESI, intelService)
	if providerErr != nil {
		return fmt.Errorf("auth init failed: %w", providerErr)
	}
	authManager := auth.NewManager(app, provider)
	auditSvc := audit.New(app)

	var eveProvider *auth.EVEProvider
	if value, ok := provider.(*auth.EVEProvider); ok {
		eveProvider = value
	}
	esiClient := buildESIClient(app, cfg)

	intelHandler := intelapi.NewIntelHandler(app, cfg, intelService)
	jumpbridgeService := jumpbridges.NewJumpbridgeService(app, esiClient, publicESI)
	timerService := timerssvc.NewService(app, publicESI, esiClient)
	realtimePublisher := realtime.NewPublisher(app)
	defer realtimePublisher.Stop()
	mapHandler := mapapi.NewMapHandler(
		app,
		cfg,
		esiClient,
		provider,
		eveProvider,
		intel.NewRoutePlanner(app),
		intel.NewTopRoutesService(app),
		timerService,
	)
	authHandler := authapi.NewAuthHandler(authManager, auditSvc)
	cleanupSvc := cleanup.New(app)
	staffChannels := staffapi.NewChannelsHandler(app, auditSvc)
	staffJumpbridges := staffapi.NewJumpbridgeHandler(app, jumpbridgeService, auditSvc)
	staffOrgStandings := staffapi.NewOrganizationStandingsHandler(app, auditSvc)
	adminMapDataUpdate := adminapi.NewMapUpdateHandler(app, auditSvc)
	characterRefresher := auth.NewCharacterRefresher(app, eveProvider, esiClient, publicESI, intelService, auditSvc)
	admin := adminapi.NewHandler(app, characterRefresher, provider, cleanupSvc, intelService, timerService, jumpbridgeService, auditSvc)
	timers := timerapi.NewHandler(timerService, auditSvc, provider)
	timerWebhook := timerwebhookapi.NewHandler(timerService)
	uploaderReleases := uploaderrelease.New(app, cfg)
	uploaderHandler := uploaderapi.NewHandler(uploaderReleases)

	deps := dependencies{
		audit:              auditSvc,
		provider:           provider,
		authManager:        authManager,
		eveProvider:        eveProvider,
		intelService:       intelService,
		esiClient:          esiClient,
		publicESI:          publicESI,
		intelHandler:       intelHandler,
		mapHandler:         mapHandler,
		authHandler:        authHandler,
		cleanup:            cleanupSvc,
		staffChannels:      staffChannels,
		staffJumpbridges:   staffJumpbridges,
		staffOrgStandings:  staffOrgStandings,
		adminMapDataUpdate: adminMapDataUpdate,
		characterRefresher: characterRefresher,
		admin:              admin,
		timerService:       timerService,
		timers:             timers,
		timerWebhook:       timerWebhook,
		uploaderReleases:   uploaderReleases,
		uploaderHandler:    uploaderHandler,
		realtimePublisher:  realtimePublisher,
	}

	registerRoutes(app, cfg, &deps)
	registerCrons(app, cfg, &deps)
	registerRealtime(app, intelService, realtimePublisher)
	registerTrustedProxyDefaults(app, cfg)

	if startErr := app.Start(); startErr != nil {
		logging.New(app).
			WithErr(startErr).
			Error("server start failed")
		return startErr
	}
	return nil
}

func buildAuthProvider(cfg *config.Config, app *pocketbase.PocketBase, publicESI *esi.ESIPublicClient, intelService *intel.IntelService) (auth.Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("missing config")
	}
	switch cfg.AuthBackend {
	case "eve":
		oauthConfig := oauth2.Config{
			ClientID:     cfg.EVEClientID,
			ClientSecret: cfg.EVEClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  cfg.EVEAuthURL,
				TokenURL: cfg.EVETokenURL,
			},
			Scopes: cfg.EVEScopeList(),
		}
		esiClient := esi.NewESIDirectClient(cfg.ESIUserAgent, logging.New(app))
		return auth.NewEVEProvider(context.Background(), app, &oauthConfig, esiClient, publicESI, intelService, cfg.PublicBaseURL, cfg.DebugEnabled)
	case "testauth":
		if cfg.TestAuthURL == "" {
			return nil, fmt.Errorf("TESTAUTH_URL required when AUTH_BACKEND=testauth")
		}
		if cfg.TestAuthClientID == "" {
			return nil, fmt.Errorf("TESTAUTH_CLIENT_ID required when AUTH_BACKEND=testauth")
		}
		if cfg.TestAuthClientSecret == "" {
			return nil, fmt.Errorf("TESTAUTH_CLIENT_SECRET required when AUTH_BACKEND=testauth")
		}
		oauthClient, oauthClientErr := auth.NewTestAuthClient(
			context.Background(),
			cfg.TestAuthURL,
			cfg.TestAuthClientID,
			cfg.TestAuthClientSecret,
			"",
			cfg.TestAuthScopes,
		)
		if oauthClientErr != nil {
			return nil, fmt.Errorf("failed to create testauth client: %w", oauthClientErr)
		}
		return auth.NewTestAuthProvider(app, oauthClient, publicESI, intelService, cfg, cfg.PublicBaseURL, cfg.DebugEnabled), nil
	default:
		return nil, fmt.Errorf("unknown auth backend: %s", cfg.AuthBackend)
	}
}

func buildESIClient(app *pocketbase.PocketBase, cfg *config.Config) esi.ESIClient {
	if cfg == nil {
		return esi.NewESIDirectClient("", logging.New(app))
	}

	if cfg.AuthBackend == "eve" {
		return esi.NewESIDirectClient(cfg.ESIUserAgent, logging.New(app))
	}
	return esi.NewTestAuthESIClient(cfg.TestAuthURL, cfg.ESIUserAgent, logging.New(app))
}
