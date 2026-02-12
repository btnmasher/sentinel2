package runtime

import (
	"context"
	"fmt"
	"sync"

	"sentinel2-uploader/internal/client"
	"sentinel2-uploader/internal/config"
	"sentinel2-uploader/internal/logging"
)

type Controller struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
	done    chan error
}

type StartHooks struct {
	OnChannelsUpdate func([]client.ChannelConfig)
}

func NewController() *Controller {
	return &Controller{}
}

func (c *Controller) Start(opts config.Options, logger *logging.Logger, hooks StartHooks) (<-chan error, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return nil, fmt.Errorf("uploader is already running")
	}
	if err := config.ValidateRequired(opts); err != nil {
		return nil, err
	}
	if logger != nil {
		logger.Debug("runtime start requested",
			logging.Field("log_dir", opts.LogDir),
			logging.Field("log_file", opts.LogFile),
			logging.Field("has_channel_hook", hooks.OnChannelsUpdate != nil),
		)
	}

	service, err := NewServiceWithHooks(opts, logger, hooks)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	c.cancel = cancel
	c.running = true
	c.done = done

	go func() {
		runErr := service.RunContext(ctx)
		if logger != nil {
			if runErr != nil {
				logger.Warn("runtime service exited with error", logging.Field("error", runErr))
			} else {
				logger.Info("runtime service exited")
			}
		}
		done <- runErr
		close(done)

		c.mu.Lock()
		c.running = false
		c.cancel = nil
		c.done = nil
		c.mu.Unlock()
	}()

	return done, nil
}

func (c *Controller) Stop() {
	c.mu.Lock()
	cancel := c.cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (c *Controller) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}
