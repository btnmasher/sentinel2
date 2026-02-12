//go:build !headless

package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

func (c *controller) setupTray() {
	if _, ok := c.app.(desktop.App); !ok {
		return
	}
	c.refreshTrayMenu()
}

func (c *controller) refreshTrayMenu() {
	desk, ok := c.app.(desktop.App)
	if !ok {
		return
	}

	desk.SetSystemTrayIcon(uploaderIconResource())
	desk.SetSystemTrayWindow(c.win)

	running := c.runner.IsRunning()
	canStart := c.startButton != nil && !c.startButton.Disabled()

	openItem := fyne.NewMenuItem("Open Window", func() {
		c.win.Show()
		c.win.RequestFocus()
	})
	showLogsItem := fyne.NewMenuItem("Show Logs", func() {
		c.setLogVisibility(!c.logWindowOpen)
		c.refreshTrayMenu()
	})
	showLogsItem.Checked = c.logWindowOpen

	connectItem := fyne.NewMenuItem("Connect", c.startUploader)
	connectItem.Disabled = running || !canStart

	disconnectItem := fyne.NewMenuItem("Disconnect", c.stopUploader)
	disconnectItem.Disabled = !running

	minTrayItem := fyne.NewMenuItem("Minimize to tray", func() {
		c.minimizeToTray.SetChecked(!c.minimizeToTray.Checked)
	})
	minTrayItem.Checked = c.minimizeToTray.Checked

	startMinItem := fyne.NewMenuItem("Start minimized", func() {
		c.startMinimized.SetChecked(!c.startMinimized.Checked)
	})
	startMinItem.Checked = c.startMinimized.Checked

	exitItem := fyne.NewMenuItem("Exit", func() {
		c.cleanup()
		c.app.Quit()
	})

	tray := fyne.NewMenu("Sentinel2 Uploader",
		openItem,
		showLogsItem,
		connectItem,
		disconnectItem,
		fyne.NewMenuItemSeparator(),
		minTrayItem,
		startMinItem,
		fyne.NewMenuItemSeparator(),
		exitItem,
	)
	desk.SetSystemTrayMenu(tray)
}
