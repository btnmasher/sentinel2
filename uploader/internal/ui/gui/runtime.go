//go:build !headless

package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"sentinel2-uploader/internal/client"
	"sentinel2-uploader/internal/config"
	"sentinel2-uploader/internal/logging"
	"sentinel2-uploader/internal/runtime"
)

func (c *controller) bindLogs() {
	logCh := make(chan string, 256)
	c.unsubscribe = c.logger.Subscribe(func(event logging.Event) {
		line := logging.FormatEventANSI(event)
		select {
		case logCh <- line:
		default:
			select {
			case <-logCh:
			default:
			}
			logCh <- line
		}
	})

	go func() {
		for {
			select {
			case <-c.quitLogs:
				return
			case line := <-logCh:
				text := line
				fyne.Do(func() {
					c.appendLog(text)
				})
			}
		}
	}()
}

func (c *controller) currentOptions() config.Options {
	debugEnabled := false
	if c.debugLogs != nil {
		debugEnabled = c.debugLogs.Checked
	}
	return config.Options{
		BaseURL: strings.TrimSpace(c.baseURL.Text),
		Token:   strings.TrimSpace(c.token.Text),
		LogFile: "",
		LogDir:  strings.TrimSpace(c.logDir.Text),
		Debug:   debugEnabled,
	}
}

func (c *controller) startUploader() {
	c.setStatus("Connecting", statusConnectingColor)
	opts := c.currentOptions()
	if strings.TrimSpace(opts.LogDir) == "" {
		c.setStatus("Error", statusErrorColor)
		dialog.ShowError(fmt.Errorf("log directory is required"), c.win)
		return
	}
	info, statErr := os.Stat(opts.LogDir)
	if statErr != nil || !info.IsDir() {
		c.setStatus("Error", statusErrorColor)
		if statErr != nil {
			dialog.ShowError(fmt.Errorf("log directory is not accessible: %w", statErr), c.win)
		} else {
			dialog.ShowError(fmt.Errorf("log directory is not a directory"), c.win)
		}
		return
	}
	if err := config.ValidateRequired(opts); err != nil {
		c.setStatus("Error", statusErrorColor)
		dialog.ShowError(err, c.win)
		return
	}

	done, err := c.runner.Start(opts, c.logger, runtime.StartHooks{
		OnChannelsUpdate: c.onChannelsUpdate,
	})
	if err != nil {
		c.setStatus("Error", statusErrorColor)
		dialog.ShowError(err, c.win)
		return
	}
	c.setRunningState(true)
	c.setStatus("Running", statusRunningColor)

	go func() {
		runErr := <-done
		fyne.Do(func() {
			c.setRunningState(false)
			c.refreshTrayMenu()
			if runErr != nil {
				c.setStatus("Disconnected", statusErrorColor)
				dialog.ShowError(runErr, c.win)
				return
			}
			c.setStatus("Idle", statusIdleColor)
		})
	}()
}

func (c *controller) onChannelsUpdate(channels []client.ChannelConfig) {
	fyne.Do(func() {
		c.setChannels(channels)
	})
}

func (c *controller) stopUploader() {
	if c.runner.IsRunning() {
		c.setStatus("Stopping", statusStoppingColor)
	}
	c.runner.Stop()
}

func (c *controller) setRunningState(running bool) {
	if running {
		c.startButton.Disable()
		c.stopButton.Enable()
		c.refreshChannelPlaceholder()
		return
	}
	c.stopButton.Disable()
	c.refreshStartAvailability()
	c.refreshChannelPlaceholder()
}

func (c *controller) setLogVisibility(visible bool) {
	if visible {
		c.logWindowOpen = true
		c.logWindow.Show()
		c.logWindow.RequestFocus()
	} else {
		c.logWindowOpen = false
		c.logWindow.Hide()
	}
}

func (c *controller) selectLogDir() {
	start := c.ensureDirPickerStartPath(c.logDir.Text)
	c.dirPickerCurrent = start

	if c.dirPickerWindow == nil {
		c.dirPickerWindow = c.app.NewWindow("Select EVE Chat Logs Folder")
		c.dirPickerWindow.Resize(fyne.NewSize(760, 520))
		c.dirPickerPath = widget.NewEntry()
		c.dirPickerPath.OnSubmitted = func(value string) {
			candidate := c.ensureDirPickerStartPath(value)
			c.dirPickerCurrent = candidate
			c.dirPickerPath.SetText(candidate)
			c.refreshDirPickerList()
		}
		upButton := widget.NewButton("Up", func() {
			parent := filepath.Dir(c.dirPickerCurrent)
			if parent == "" || parent == c.dirPickerCurrent {
				return
			}
			c.dirPickerCurrent = parent
			c.dirPickerPath.SetText(parent)
			c.refreshDirPickerList()
		})
		useCurrent := widget.NewButton("Use Current Folder", func() {
			c.logDir.SetText(c.dirPickerCurrent)
			c.prefs.SetString(prefLogDir, strings.TrimSpace(c.logDir.Text))
			c.dirPickerWindow.Hide()
		})
		closeButton := widget.NewButton("Close", func() {
			c.dirPickerWindow.Hide()
		})

		c.dirPickerList = widget.NewList(
			func() int { return len(c.dirPickerItems) },
			func() fyne.CanvasObject { return widget.NewLabel("directory") },
			func(id widget.ListItemID, obj fyne.CanvasObject) {
				obj.(*widget.Label).SetText(c.dirPickerItems[id])
			},
		)
		c.dirPickerList.OnSelected = func(id widget.ListItemID) {
			if id < 0 || id >= len(c.dirPickerItems) {
				return
			}
			next := filepath.Join(c.dirPickerCurrent, c.dirPickerItems[id])
			c.dirPickerCurrent = c.ensureDirPickerStartPath(next)
			c.dirPickerPath.SetText(c.dirPickerCurrent)
			c.refreshDirPickerList()
		}

		header := container.NewBorder(nil, nil, upButton, nil, c.dirPickerPath)
		actions := container.NewHBox(useCurrent, closeButton)
		c.dirPickerWindow.SetContent(container.NewBorder(header, actions, nil, nil, c.dirPickerList))
	}

	c.dirPickerPath.SetText(c.dirPickerCurrent)
	c.refreshDirPickerList()
	c.dirPickerWindow.Show()
	c.dirPickerWindow.RequestFocus()
}

func (c *controller) appendLog(line string) {
	if c.logGrid == nil {
		return
	}
	rows := parseANSITextGridRows(line)
	if len(rows) == 0 {
		return
	}
	c.logRows = append(c.logRows, rows...)
	c.trimLogRows()
	c.logGrid.Rows = c.logRows
	c.logGrid.Refresh()
	if c.followLogs != nil && c.followLogs.Checked {
		c.logGrid.ScrollToBottom()
	}
}

func (c *controller) trimLogRows() {
	const maxLogRows = 5000
	if len(c.logRows) <= maxLogRows {
		return
	}
	c.logRows = append([]widget.TextGridRow(nil), c.logRows[len(c.logRows)-maxLogRows:]...)
}

func (c *controller) cleanup() {
	c.cleanupOnce.Do(func() {
		if c.unsubscribe != nil {
			c.unsubscribe()
		}
		close(c.quitLogs)
		c.runner.Stop()
		if c.logWindow != nil {
			c.logWindow.Close()
		}
		if c.dirPickerWindow != nil {
			c.dirPickerWindow.Close()
		}
	})
}
