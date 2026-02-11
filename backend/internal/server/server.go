package server

import (
	"context"
	"fmt"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"golang.org/x/oauth2"

	"sentinel2/internal/auth"
	"sentinel2/internal/cleanup"
	"sentinel2/internal/config"
	"sentinel2/internal/esi"
	"sentinel2/internal/intel"
	"sentinel2/internal/logging"
	"sentinel2/internal/oidc"

	adminapi "sentinel2/internal/api/admin"
	authapi "sentinel2/internal/api/auth"
	intelapi "sentinel2/internal/api/intel"
	mapapi "sentinel2/internal/api/maps"
	staffapi "sentinel2/internal/api/staff"
)

type dependencies struct {
	provider           auth.Provider
	authManager        *auth.Manager
	eveProvider        *auth.EVEProvider
	esiClient          esi.ESIClient
	publicESI          *esi.ESIPublicClient
	intelHandler       *intelapi.IntelHandler
	mapHandler         *mapapi.MapHandler
	authHandler        *authapi.AuthHandler
	cleanup            *cleanup.Service
	staffChannels      *staffapi.ChannelsHandler
	staffJumpbridges   *staffapi.JumpbridgeHandler
	adminMapDataUpdate *adminapi.MapUpdateHandler
	characterRefresher *auth.CharacterRefresher
	admin              *adminapi.Handler
}

func Run(cfg config.Config) error {
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDev: cfg.DebugEnabled,
	})

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{Dir: "pb_migrations"})

	publicESI := esi.NewESIPublicClient(cfg.ESIUserAgent)
	provider, providerErr := buildAuthProvider(cfg, app, publicESI)
	if providerErr != nil {
		return fmt.Errorf("auth init failed: %w", providerErr)
	}
	authManager := auth.NewManager(app, provider)

	var eveProvider *auth.EVEProvider
	if value, ok := provider.(*auth.EVEProvider); ok {
		eveProvider = value
	}
	esiClient := buildESIClient(app, cfg)

	intelHandler := intelapi.NewIntelHandler(app, cfg)
	mapHandler := mapapi.NewMapHandler(
		app,
		cfg,
		esiClient,
		provider,
		eveProvider,
		intel.NewRoutePlanner(app),
		intel.NewTopRoutesService(app),
	)
	authHandler := authapi.NewAuthHandler(authManager)
	cleanup := cleanup.New(app)
	staffChannels := staffapi.NewChannelsHandler(app)
	staffJumpbridges := staffapi.NewJumpbridgeHandler(app)
	adminMapDataUpdate := adminapi.NewMapUpdateHandler(app)
	characterRefresher := auth.NewCharacterRefresher(app, eveProvider, esiClient, publicESI)
	admin := adminapi.NewHandler(app, characterRefresher, eveProvider, cleanup)

	deps := dependencies{
		provider:           provider,
		authManager:        authManager,
		eveProvider:        eveProvider,
		esiClient:          esiClient,
		publicESI:          publicESI,
		intelHandler:       intelHandler,
		mapHandler:         mapHandler,
		authHandler:        authHandler,
		cleanup:            cleanup,
		staffChannels:      staffChannels,
		staffJumpbridges:   staffJumpbridges,
		adminMapDataUpdate: adminMapDataUpdate,
		characterRefresher: characterRefresher,
		admin:              admin,
	}

	registerRoutes(app, cfg, deps)
	registerCrons(app, cfg, deps)

	if startErr := app.Start(); startErr != nil {
		logging.New(app).
			WithErr(startErr).
			Error("server start failed")
		return startErr
	}
	return nil
}

func buildAuthProvider(cfg config.Config, app *pocketbase.PocketBase, publicESI *esi.ESIPublicClient) (auth.Provider, error) {
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
		return auth.NewEVEProvider(app, oauthConfig, esiClient, publicESI), nil
	default:
		oidcClient, oidcErr := oidc.New(context.Background(), cfg)
		if oidcErr != nil {
			return nil, oidcErr
		}
		return auth.NewTestAuthProvider(app, oidcClient), nil
	}
}

func buildESIClient(app *pocketbase.PocketBase, cfg config.Config) esi.ESIClient {
	if cfg.AuthBackend == "eve" {
		return esi.NewESIDirectClient(cfg.ESIUserAgent, logging.New(app))
	}
	baseURL := cfg.ESIDirectBaseURL
	if cfg.ESIProxyBaseURL != "" {
		baseURL = cfg.ESIProxyBaseURL
	}
	return esi.NewESIProxyClient(baseURL, logging.New(app))
}
