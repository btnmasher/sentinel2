//go:build !headless

package gui

import (
	"image/color"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

const badgeSize = float32(10)

type statusBadge struct {
	widget.BaseWidget

	fill    color.NRGBA
	tooltip string

	dot   *canvas.Circle
	label *widget.Label
	pop   *widget.PopUp

	hoverTimer *time.Timer
	hoverSeq   atomic.Uint64
	shown      bool
}

var _ desktop.Hoverable = (*statusBadge)(nil)

func newStatusBadge() *statusBadge {
	b := &statusBadge{
		fill: channelRedColor,
	}
	b.dot = canvas.NewCircle(b.fill)
	b.label = widget.NewLabel("")
	b.ExtendBaseWidget(b)
	return b
}

func (b *statusBadge) SetStatus(fill color.NRGBA, tooltip string) {
	b.fill = fill
	b.tooltip = tooltip
	b.dot.FillColor = fill
	b.dot.Refresh()
	if tooltip == "" {
		b.hideTooltip()
	}
}

func (b *statusBadge) MinSize() fyne.Size {
	return fyne.NewSize(badgeSize, badgeSize)
}

func (b *statusBadge) CreateRenderer() fyne.WidgetRenderer {
	wrapped := container.NewGridWrap(fyne.NewSize(badgeSize, badgeSize), b.dot)
	return widget.NewSimpleRenderer(wrapped)
}

func (b *statusBadge) MouseIn(*desktop.MouseEvent) {
	b.scheduleTooltip()
}

func (b *statusBadge) MouseMoved(*desktop.MouseEvent) {
	if b.shown {
		b.showTooltipNow()
		return
	}
	b.scheduleTooltip()
}

func (b *statusBadge) MouseOut() {
	b.cancelTooltipTimer()
	b.hideTooltip()
}

func (b *statusBadge) scheduleTooltip() {
	if b.tooltip == "" {
		return
	}
	seq := b.hoverSeq.Add(1)
	b.cancelTooltipTimer()
	b.hoverTimer = time.AfterFunc(180*time.Millisecond, func() {
		fyne.Do(func() {
			if b.hoverSeq.Load() != seq {
				return
			}
			b.showTooltipNow()
		})
	})
}

func (b *statusBadge) cancelTooltipTimer() {
	b.hoverSeq.Add(1)
	if b.hoverTimer != nil {
		b.hoverTimer.Stop()
		b.hoverTimer = nil
	}
}

func (b *statusBadge) showTooltipNow() {
	if b.tooltip == "" {
		b.hideTooltip()
		return
	}

	app := fyne.CurrentApp()
	if app == nil {
		return
	}
	canvasForObject := app.Driver().CanvasForObject(b)
	if canvasForObject == nil {
		return
	}

	if b.pop == nil {
		b.label = widget.NewLabel(b.tooltip)
		b.label.Wrapping = fyne.TextWrapOff
		content := container.NewPadded(b.label)
		b.pop = widget.NewPopUp(content, canvasForObject)
	} else {
		b.label.SetText(b.tooltip)
	}
	b.pop.Resize(b.pop.Content.MinSize())

	pos := app.Driver().AbsolutePositionForObject(b)
	b.pop.ShowAtPosition(fyne.NewPos(pos.X+14, pos.Y-4))
	b.shown = true
}

func (b *statusBadge) hideTooltip() {
	if b.pop != nil {
		b.pop.Hide()
	}
	b.shown = false
}
