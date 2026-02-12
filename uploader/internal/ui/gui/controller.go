//go:build !headless

package gui

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"sentinel2-uploader/internal/client"
	"sentinel2-uploader/internal/config"
	"sentinel2-uploader/internal/evelogs"
	"sentinel2-uploader/internal/logging"
	"sentinel2-uploader/internal/runtime"
)

const (
	prefBaseURL        = "uploader.base_url"
	prefToken          = "uploader.token"
	prefLogDir         = "uploader.log_dir"
	prefDebugLogs      = "uploader.debug_logs"
	prefAutoConnect    = "uploader.auto_connect"
	prefMinimizeToTray = "uploader.minimize_to_tray"
	prefStartMinimized = "uploader.start_minimized"
)

var (
	statusIdleColor       = color.NRGBA{R: 145, G: 145, B: 145, A: 255}
	statusConnectingColor = color.NRGBA{R: 219, G: 167, B: 74, A: 255}
	statusRunningColor    = color.NRGBA{R: 72, G: 189, B: 109, A: 255}
	statusStoppingColor   = color.NRGBA{R: 232, G: 145, B: 77, A: 255}
	statusErrorColor      = color.NRGBA{R: 220, G: 84, B: 84, A: 255}
	channelGreenColor     = color.NRGBA{R: 72, G: 189, B: 109, A: 255}
	channelYellowColor    = color.NRGBA{R: 219, G: 167, B: 74, A: 255}
	channelOrangeColor    = color.NRGBA{R: 232, G: 145, B: 77, A: 255}
	channelRedColor       = color.NRGBA{R: 220, G: 84, B: 84, A: 255}
)

const (
	channelStatusWarnAfter   = 10 * time.Minute
	channelStatusStaleAfter  = time.Hour
	channelStatusRefreshRate = 30 * time.Second
)

type channelHealth struct {
	Color  color.NRGBA
	Reason string
}

type channelStatusRow struct {
	Channel client.ChannelConfig
	Health  channelHealth
}

type controller struct {
	app    fyne.App
	prefs  fyne.Preferences
	win    fyne.Window
	logger *logging.Logger
	runner *runtime.Controller

	baseURL *widget.Entry
	token   *widget.Entry
	logDir  *widget.Entry

	debugLogs      *widget.Check
	connectOnStart *sliderToggle
	minimizeToTray *sliderToggle
	startMinimized *sliderToggle
	statusDot      *canvas.Circle
	statusDotWrap  fyne.CanvasObject
	statusText     *widget.Label

	startButton    *widget.Button
	stopButton     *widget.Button
	showLogsButton *widget.Button

	logWindow     fyne.Window
	logWindowOpen bool
	logGrid       *widget.TextGrid
	followLogs    *widget.Check
	logRows       []widget.TextGridRow
	channels      []client.ChannelConfig
	channelRows   []channelStatusRow
	channelList   *widget.List
	channelEmpty  *fyne.Container
	channelNotice *widget.Label

	dirPickerWindow  fyne.Window
	dirPickerPath    *widget.Entry
	dirPickerCurrent string
	dirPickerItems   []string
	dirPickerList    *widget.List

	cleanupOnce sync.Once
	unsubscribe func()
	quitLogs    chan struct{}
}

func Run(buildVersion string, defaults config.Options) {
	uiApp := app.NewWithID("com.sentinel2.uploader")
	c := newController(uiApp, defaults)
	c.logger.Info("starting uploader UI", logging.Field("version", buildVersion))
	c.run()
}

func newController(uiApp fyne.App, defaults config.Options) *controller {
	prefs := uiApp.Preferences()
	defaults = applyPreferenceDefaults(prefs, defaults)

	logger := logging.New(false)
	logger.SetDebugEnabled(false)

	c := &controller{
		app:      uiApp,
		prefs:    prefs,
		logger:   logger,
		runner:   runtime.NewController(),
		quitLogs: make(chan struct{}),
	}

	uiApp.SetIcon(uploaderIconResource())
	c.win = uiApp.NewWindow("Sentinel2 Uploader")
	c.win.Resize(fyne.NewSize(460, 390))
	c.buildUI(defaults)
	c.bindLogs()
	c.setupTray()
	return c
}

func (c *controller) run() {
	c.setRunningState(false)
	c.startChannelHealthLoop()
	c.win.SetCloseIntercept(func() {
		if c.minimizeToTray.Checked {
			c.win.Hide()
			return
		}
		c.cleanup()
		c.app.Quit()
	})

	if c.startMinimized.Checked {
		c.win.Show()
		c.win.Hide()
		c.tryAutoConnect()
		c.app.Run()
		return
	}

	c.win.Show()
	c.tryAutoConnect()
	c.app.Run()
}

func (c *controller) startChannelHealthLoop() {
	ticker := time.NewTicker(channelStatusRefreshRate)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-c.quitLogs:
				return
			case <-ticker.C:
				fyne.Do(func() {
					c.refreshChannelHealth()
				})
			}
		}
	}()
}

func (c *controller) buildUI(defaults config.Options) {
	c.baseURL = widget.NewEntry()
	c.baseURL.SetText(defaults.BaseURL)

	c.token = widget.NewPasswordEntry()
	c.token.SetText(defaults.Token)

	c.logDir = widget.NewEntry()
	c.logDir.SetText(defaults.LogDir)

	c.debugLogs = widget.NewCheck("Debug level", func(v bool) {
		c.prefs.SetBool(prefDebugLogs, v)
		c.logger.SetDebugEnabled(v)
	})
	c.debugLogs.SetChecked(c.prefs.BoolWithFallback(prefDebugLogs, false))
	c.logger.SetDebugEnabled(c.debugLogs.Checked)
	c.connectOnStart = newSliderToggle(func(v bool) {
		c.prefs.SetBool(prefAutoConnect, v)
	})
	c.connectOnStart.SetChecked(c.prefs.BoolWithFallback(prefAutoConnect, defaults.AutoConnect))

	c.minimizeToTray = newSliderToggle(func(v bool) {
		c.prefs.SetBool(prefMinimizeToTray, v)
		c.refreshTrayMenu()
	})
	c.minimizeToTray.SetChecked(c.prefs.BoolWithFallback(prefMinimizeToTray, false))

	c.startMinimized = newSliderToggle(func(v bool) {
		c.prefs.SetBool(prefStartMinimized, v)
		c.refreshTrayMenu()
	})
	c.startMinimized.SetChecked(c.prefs.BoolWithFallback(prefStartMinimized, false))

	c.statusDot = canvas.NewCircle(statusIdleColor)
	c.statusDotWrap = container.NewGridWrap(fyne.NewSize(12, 12), c.statusDot)
	c.statusText = widget.NewLabel("Idle")
	c.channelList = widget.NewList(
		func() int {
			return len(c.channelRows)
		},
		func() fyne.CanvasObject {
			badge := newStatusBadge()
			label := widget.NewLabel("channel")
			return container.NewBorder(nil, nil, nil, badge, label)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			row := obj.(*fyne.Container)
			badge := row.Objects[0].(*statusBadge)
			label := row.Objects[1].(*widget.Label)
			if id >= 0 && id < len(c.channelRows) {
				item := c.channelRows[id]
				badge.SetStatus(item.Health.Color, item.Health.Reason)
				label.SetText(item.Channel.Name)
				return
			}
			badge.SetStatus(channelRedColor, "")
			label.SetText("")
		},
	)

	c.initLogWindow()
	c.setStatus("Idle", statusIdleColor)

	c.startButton = widget.NewButton("Start uploader", func() {
		c.startUploader()
		c.refreshTrayMenu()
	})
	c.stopButton = widget.NewButton("Stop uploader", func() {
		c.stopUploader()
		c.refreshTrayMenu()
	})
	c.showLogsButton = widget.NewButton("Show logs", func() {
		c.setLogVisibility(true)
		c.refreshTrayMenu()
	})
	c.stopButton.Disable()

	c.baseURL.OnChanged = func(v string) {
		c.prefs.SetString(prefBaseURL, strings.TrimSpace(v))
		c.refreshStartAvailability()
	}
	c.token.OnChanged = func(v string) {
		c.prefs.SetString(prefToken, strings.TrimSpace(v))
		c.refreshStartAvailability()
	}
	c.logDir.OnChanged = func(v string) {
		c.prefs.SetString(prefLogDir, strings.TrimSpace(v))
		c.refreshChannelHealth()
	}

	browseLogDir := widget.NewButton("Browse...", c.selectLogDir)
	logDirRow := container.NewBorder(nil, nil, nil, browseLogDir, c.logDir)

	form := container.NewVBox(
		widget.NewLabel("Base URL"),
		c.baseURL,
		c.verticalGap(8),
		widget.NewLabel("Uploader Token"),
		c.token,
		c.verticalGap(8),
		widget.NewLabel("Log Directory"),
		logDirRow,
	)

	settingsRow := container.NewVBox(
		c.toggleRow("Connect on startup", c.connectOnStart),
		c.toggleRow("Close to tray", c.minimizeToTray),
		c.toggleRow("Start minimized", c.startMinimized),
	)
	statusRow := container.NewHBox(container.NewCenter(c.statusDotWrap), c.statusText)
	controls := container.NewHBox(c.startButton, c.stopButton, c.showLogsButton, widget.NewLabel("Status:"), statusRow)

	overviewTop := container.NewPadded(container.NewVBox(
		controls,
	))
	c.channelNotice = widget.NewLabel("Not connected")
	c.channelNotice.Alignment = fyne.TextAlignCenter
	c.channelEmpty = container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(c.channelNotice),
		layout.NewSpacer(),
	)
	channelStack := container.NewMax(c.channelList, c.channelEmpty)
	channelPanel := container.NewPadded(container.NewBorder(
		widget.NewLabel("Configured Channels"),
		nil,
		nil,
		nil,
		channelStack,
	))
	pad := func(obj fyne.CanvasObject) fyne.CanvasObject {
		return container.NewPadded(container.NewPadded(obj))
	}

	overviewTab := container.NewTabItem("Overview", pad(container.NewBorder(
		overviewTop,
		nil,
		nil,
		nil,
		channelPanel,
	)))
	settingsTab := container.NewTabItem("Settings", pad(container.NewVBox(
		form,
		c.verticalGap(12),
		settingsRow,
	)))
	tabs := container.NewAppTabs(overviewTab, settingsTab)
	tabs.SetTabLocation(container.TabLocationTop)
	minAnchor := canvas.NewRectangle(color.Transparent)
	minAnchor.SetMinSize(fyne.NewSize(500, 340))
	c.win.SetContent(container.NewStack(minAnchor, tabs))
	c.refreshStartAvailability()
	c.refreshChannelHealth()
}

func (c *controller) toggleRow(label string, sw *sliderToggle) fyne.CanvasObject {
	return container.NewBorder(nil, nil, widget.NewLabel(label), sw, nil)
}

func (c *controller) verticalGap(height float32) fyne.CanvasObject {
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(1, height))
	return spacer
}

func (c *controller) setStatus(text string, dotColor color.NRGBA) {
	c.statusText.SetText(text)
	c.statusDot.FillColor = dotColor
	c.statusDot.Refresh()
}

func (c *controller) refreshStartAvailability() {
	if c.runner.IsRunning() {
		return
	}
	baseURL := strings.TrimSpace(c.baseURL.Text)
	token := strings.TrimSpace(c.token.Text)
	if baseURL == "" || token == "" {
		c.startButton.Disable()
		return
	}
	c.startButton.Enable()
}

func (c *controller) tryAutoConnect() {
	if c.connectOnStart == nil || !c.connectOnStart.Checked || c.runner.IsRunning() {
		return
	}
	if strings.TrimSpace(c.baseURL.Text) == "" || strings.TrimSpace(c.token.Text) == "" {
		return
	}
	fyne.Do(func() {
		c.startUploader()
		c.refreshTrayMenu()
	})
}

func (c *controller) setChannels(channels []client.ChannelConfig) {
	c.channels = append([]client.ChannelConfig(nil), channels...)
	c.refreshChannelHealth()
}

func (c *controller) refreshChannelHealth() {
	rows := make([]channelStatusRow, 0, len(c.channels))
	now := time.Now()
	logDir := strings.TrimSpace(c.logDir.Text)

	latestByChannel := map[string]time.Time{}
	latestPathByChannel := map[string]string{}
	scanErrText := ""
	if logDir == "" {
		scanErrText = "Log directory is not configured."
	} else if info, statErr := os.Stat(logDir); statErr != nil {
		scanErrText = fmt.Sprintf("Log directory is not accessible: %v", statErr)
	} else if !info.IsDir() {
		scanErrText = "Log path is not a directory."
	} else {
		logs, findErr := evelogs.FindLogs(logDir, c.channels)
		if findErr != nil {
			scanErrText = fmt.Sprintf("Failed to scan logs: %v", findErr)
		} else {
			for _, selection := range logs {
				stat, statErr := os.Stat(selection.Path)
				if statErr != nil {
					continue
				}
				id := strings.TrimSpace(selection.Channel.ID)
				if id == "" {
					continue
				}
				current, ok := latestByChannel[id]
				if !ok || stat.ModTime().After(current) {
					latestByChannel[id] = stat.ModTime()
					latestPathByChannel[id] = filepath.Base(selection.Path)
				}
			}
		}
	}

	for _, channel := range c.channels {
		id := strings.TrimSpace(channel.ID)
		name := strings.TrimSpace(channel.Name)
		health := channelHealth{
			Color:  channelRedColor,
			Reason: "Channel log file was not found in the configured log directory.",
		}
		if scanErrText != "" {
			health.Reason = scanErrText
		} else if last, ok := latestByChannel[id]; ok {
			age := now.Sub(last)
			fileName := latestPathByChannel[id]
			if age <= channelStatusWarnAfter {
				health.Color = channelGreenColor
				health.Reason = fmt.Sprintf("Active: %s updated %s ago.", fileName, age.Round(time.Second))
			} else if age <= channelStatusStaleAfter {
				health.Color = channelYellowColor
				health.Reason = fmt.Sprintf("Stale: %s has no updates for %s.", fileName, age.Round(time.Second))
			} else {
				health.Color = channelOrangeColor
				health.Reason = fmt.Sprintf("Very stale: %s has no updates for %s.", fileName, age.Round(time.Second))
			}
		}

		rows = append(rows, channelStatusRow{
			Channel: client.ChannelConfig{
				ID:   id,
				Name: name,
			},
			Health: health,
		})
	}
	c.channelRows = rows
	if c.channelList != nil {
		c.channelList.Refresh()
	}
	c.refreshChannelPlaceholder()
}

func (c *controller) refreshChannelPlaceholder() {
	if c.channelList == nil || c.channelEmpty == nil || c.channelNotice == nil {
		return
	}
	if len(c.channelRows) > 0 {
		c.channelEmpty.Hide()
		c.channelList.Show()
		return
	}
	if c.runner != nil && c.runner.IsRunning() {
		c.channelNotice.SetText("No channels configured")
	} else {
		c.channelNotice.SetText("Not connected")
	}
	c.channelList.Hide()
	c.channelEmpty.Show()
}

func (c *controller) initLogWindow() {
	c.logGrid = widget.NewTextGrid()
	c.logGrid.Scroll = fyne.ScrollVerticalOnly
	c.followLogs = widget.NewCheck("Follow output", nil)
	c.followLogs.SetChecked(true)
	clearButton := widget.NewButton("Clear", func() {
		c.logRows = nil
		c.logGrid.Rows = nil
		c.logGrid.Refresh()
	})
	c.logWindow = c.app.NewWindow("Sentinel2 Uploader Logs")
	c.logWindow.Resize(fyne.NewSize(900, 520))
	header := container.NewHBox(c.debugLogs, c.followLogs, clearButton)
	c.logWindow.SetContent(container.NewBorder(header, nil, nil, nil, c.logGrid))
	c.logWindowOpen = false
	c.logWindow.SetCloseIntercept(func() {
		c.logWindowOpen = false
		c.logWindow.Hide()
		c.refreshTrayMenu()
	})
}

func (c *controller) ensureDirPickerStartPath(path string) string {
	candidate := strings.TrimSpace(path)
	if candidate == "" {
		candidate = config.DefaultLogDir()
	}
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return filepath.Clean(candidate)
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Clean(home)
	}
	return "/"
}

func (c *controller) refreshDirPickerList() {
	entries, err := os.ReadDir(c.dirPickerCurrent)
	if err != nil {
		c.dirPickerItems = nil
		c.dirPickerList.Refresh()
		return
	}
	items := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			items = append(items, entry.Name())
		}
	}
	sort.Strings(items)
	c.dirPickerItems = items
	c.dirPickerList.Refresh()
}
