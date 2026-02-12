package app

import (
	"context"
	"fmt"

	"sentinel2-uploader/internal/client"
	"sentinel2-uploader/internal/config"
	"sentinel2-uploader/internal/evelogs"
	"sentinel2-uploader/internal/logging"
)

type UploaderApp struct {
	opts   config.Options
	client *client.SentinelClient
	logger *logging.Logger
	hooks  Callbacks
}

type Callbacks struct {
	OnChannelsUpdate func([]client.ChannelConfig)
}

func New(opts config.Options, client *client.SentinelClient, logger *logging.Logger, hooks Callbacks) *UploaderApp {
	return &UploaderApp{opts: opts, client: client, logger: logger, hooks: hooks}
}

func (a *UploaderApp) Run() error {
	return a.RunContext(context.Background())
}

func (a *UploaderApp) RunContext(ctx context.Context) error {
	a.logger.Info("uploader app starting",
		logging.Field("log_dir", a.opts.LogDir),
		logging.Field("log_file", a.opts.LogFile),
	)
	channels, err := a.client.FetchChannels()
	if err != nil {
		return fmt.Errorf("failed to fetch channels: %w", err)
	}
	if len(channels) == 0 {
		return fmt.Errorf("no channels configured")
	}
	a.logger.Info("initial channels loaded", logging.Field("count", len(channels)))
	a.notifyChannels(channels)

	configUpdates := a.client.StartChannelConfigSync(ctx, channels)
	monitorUpdates := make(chan []client.ChannelConfig, 1)
	go a.forwardChannelUpdates(ctx, configUpdates, monitorUpdates)
	monitor := evelogs.NewMonitor(evelogs.MonitorOptions{
		LogDir:   a.opts.LogDir,
		LogFile:  a.opts.LogFile,
		Channels: channels,
	}, a.logger, evelogs.MonitorCallbacks{
		OnReport: func(event evelogs.ReportEvent) error {
			return a.client.Submit(client.SubmitPayload{Text: event.Line, ChannelID: event.Channel.ID})
		},
		OnError: func(err error) {
			a.logger.Warn("log monitor callback error", logging.Field("error", err))
		},
	})

	runErr := monitor.RunContext(ctx, monitorUpdates)
	if runErr != nil {
		a.logger.Warn("uploader app stopped with error", logging.Field("error", runErr))
		return runErr
	}
	a.logger.Info("uploader app stopped")
	return nil
}

func (a *UploaderApp) forwardChannelUpdates(ctx context.Context, source <-chan []client.ChannelConfig, target chan<- []client.ChannelConfig) {
	defer close(target)
	for {
		select {
		case <-ctx.Done():
			a.logger.Debug("stopping channel update forwarder: context canceled")
			return
		case channels, ok := <-source:
			if !ok {
				a.logger.Debug("stopping channel update forwarder: source closed")
				return
			}
			a.logger.Debug("forwarding channel update", logging.Field("count", len(channels)))
			a.notifyChannels(channels)
			select {
			case <-ctx.Done():
				a.logger.Debug("dropping forwarded channel update: context canceled")
				return
			case target <- channels:
				a.logger.Debug("channel update forwarded", logging.Field("count", len(channels)))
			}
		}
	}
}

func (a *UploaderApp) notifyChannels(channels []client.ChannelConfig) {
	if a.hooks.OnChannelsUpdate == nil {
		return
	}
	copied := append([]client.ChannelConfig(nil), channels...)
	a.hooks.OnChannelsUpdate(copied)
}
