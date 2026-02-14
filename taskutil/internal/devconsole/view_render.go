package devconsole

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m viewState) View() string {
	if m.fullscreen {
		return m.fullscreenView()
	}

	header := m.headerLines(m.width)

	pane := func(label string, info procInfo, focused bool, content string, width int, showTop, showRight, showBottom, showLeft bool, border lipgloss.Border) string {
		if focused {
			border = lipgloss.ThickBorder()
		}
		contentWidth := max(width-2, 1)
		style := lipgloss.NewStyle().
			Border(border, showTop, showRight, showBottom, showLeft).
			Padding(0, 0).
			Width(contentWidth)
		if focused {
			style = style.BorderForeground(lipgloss.Color("86"))
		}
		h := fmt.Sprintf("%s  [%s]", label, procStateText(info))
		if info.pid > 0 {
			h += fmt.Sprintf(" pid=%d", info.pid)
		}
		if info.lastExit != "" {
			h += " " + info.lastExit
		}
		return style.Render(h + "\n" + content)
	}

	frontendRaw := m.highlightSelection(m.frontend.View(), m.frontend, "frontend")
	backendRaw := m.highlightSelection(m.backend.View(), m.backend, "backend")
	frontendView := withScrollbar(frontendRaw, m.frontend.Width, m.frontend.Height, m.frontend.ScrollPercent())
	backendView := withScrollbar(backendRaw, m.backend.Width, m.backend.Height, m.backend.ScrollPercent())

	firstProc, secondProc := m.slotProcs()
	firstView, secondView := frontendView, backendView
	if firstProc == "backend" {
		firstView, secondView = backendView, frontendView
	}

	var row string
	if m.verticalSplit {
		leftPaneWidth, rightPaneWidth := splitPaneWidths(m.width)
		focusLeft := m.focus == 0
		leftShowRight := focusLeft
		rightShowLeft := !focusLeft
		leftBorder := lipgloss.NormalBorder()
		rightBorder := lipgloss.NormalBorder()
		if focusLeft {
			leftBorder.TopRight = "┬"
			leftBorder.BottomRight = "┴"
		} else {
			rightBorder.TopLeft = "┬"
			rightBorder.BottomLeft = "┴"
		}
		row = lipgloss.JoinHorizontal(
			lipgloss.Top,
			pane(firstProc, m.procs[firstProc], m.focus == 0, firstView, leftPaneWidth, true, leftShowRight, true, true, leftBorder),
			pane(secondProc, m.procs[secondProc], m.focus == 1, secondView, rightPaneWidth, true, true, true, rightShowLeft, rightBorder),
		)
	} else {
		topBorder := lipgloss.NormalBorder()
		bottomBorder := lipgloss.NormalBorder()
		focusTop := m.focus == 0
		topShowBottom := focusTop
		bottomShowTop := !focusTop
		if focusTop {
			topBorder.BottomLeft = "├"
			topBorder.BottomRight = "┤"
		} else {
			bottomBorder.TopLeft = "├"
			bottomBorder.TopRight = "┤"
		}
		fullWidth := max(m.width, 1)
		top := pane(firstProc, m.procs[firstProc], m.focus == 0, firstView, fullWidth, true, true, topShowBottom, true, topBorder)
		bottom := pane(secondProc, m.procs[secondProc], m.focus == 1, secondView, fullWidth, bottomShowTop, true, true, true, bottomBorder)
		row = strings.Join([]string{top, bottom}, "\n")
	}

	lines := append(header, clampWidth(row, m.width))
	return fitHeight(strings.Join(lines, "\n"), m.height)
}

func (m viewState) fullscreenView() string {
	body := ""
	if m.focusedProc() == "frontend" {
		body = clampWidth(m.frontend.View(), m.width)
	} else {
		body = clampWidth(m.backend.View(), m.width)
	}
	mode := "selection mode"
	if m.mouseCaptured {
		mode = "scroll mode"
	}
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("esc back | m toggle mouse (" + mode + ")")
	return fitHeight(strings.Join([]string{body, clampWidth(hint, m.width)}, "\n"), m.height)
}
