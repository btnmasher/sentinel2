package devconsole

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m *viewState) resize(width, height int) {
	prevFrontendBottom := m.frontend.YOffset + max(m.frontend.Height-1, 0)
	prevBackendBottom := m.backend.YOffset + max(m.backend.Height-1, 0)

	m.width, m.height = width, height
	if m.fullscreen {
		innerWidth := max(width, minViewportWidth)
		innerHeight := max(height, minViewportHeight)
		m.frontend.Width = innerWidth
		m.frontend.Height = innerHeight
		m.backend.Width = innerWidth
		m.backend.Height = innerHeight
		m.refreshViewportContent()
		m.restoreViewportAnchor(prevFrontendBottom, prevBackendBottom)
		return
	}

	headerRows := len(m.headerLines(width))
	if m.verticalSplit {
		leftPaneWidth, rightPaneWidth := splitPaneWidths(width)
		leftInnerWidth := max(leftPaneWidth-3, minViewportWidth)
		rightInnerWidth := max(rightPaneWidth-3, minViewportWidth)
		rowHeight := max(height-headerRows, minBodyHeight)
		innerHeight := max(rowHeight-3, minViewportHeight)
		m.frontend.Width = leftInnerWidth
		m.frontend.Height = innerHeight
		m.backend.Width = rightInnerWidth
		m.backend.Height = innerHeight
		m.refreshViewportContent()
		m.restoreViewportAnchor(prevFrontendBottom, prevBackendBottom)
		return
	}

	rowHeight := max(height-headerRows, minBodyHeight)
	paneOuterWidth := max(width, 1)
	innerWidth := max(paneOuterWidth-3, minViewportWidth)
	innerTotal := max(rowHeight-5, minViewportHeight*2)
	topInnerHeight := innerTotal / 2
	bottomInnerHeight := innerTotal - topInnerHeight
	m.frontend.Width = innerWidth
	m.frontend.Height = max(topInnerHeight, minViewportHeight)
	m.backend.Width = innerWidth
	m.backend.Height = max(bottomInnerHeight, minViewportHeight)
	m.refreshViewportContent()
	m.restoreViewportAnchor(prevFrontendBottom, prevBackendBottom)
}

func (m *viewState) restoreViewportAnchor(prevFrontendBottom, prevBackendBottom int) {
	if m.followFrontend {
		m.frontend.GotoBottom()
	} else {
		m.frontend.SetYOffset(max(prevFrontendBottom-m.frontend.Height+1, 0))
	}

	if m.followBackend {
		m.backend.GotoBottom()
	} else {
		m.backend.SetYOffset(max(prevBackendBottom-m.backend.Height+1, 0))
	}
}

func splitPaneWidths(totalWidth int) (int, int) {
	if totalWidth <= 2 {
		return 1, 1
	}
	left := totalWidth / 2
	right := totalWidth - left
	if left < minPaneWidth || right < minPaneWidth {
		left = max(left, minPaneWidth)
		right = max(totalWidth-left, 1)
	}
	return left, right
}

func clampWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = ansi.Cut(lines[i], 0, width)
	}
	return strings.Join(lines, "\n")
}

func wrapLinesForWidth(lines []string, width int) string {
	if width <= 0 || len(lines) == 0 {
		return strings.Join(lines, "\n")
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, wrapLineForWidth(line, width)...)
	}
	return strings.Join(out, "\n")
}

func wrapLineForWidth(line string, width int) []string {
	if width <= 0 {
		return []string{line}
	}
	line = renderDisplayLineForWidth(line, width)
	if line == "" {
		return []string{""}
	}
	lineWidth := ansi.StringWidth(line)
	if lineWidth <= width {
		return []string{line}
	}
	parts := make([]string, 0, (lineWidth/width)+1)
	for start := 0; start < lineWidth; start += width {
		end := min(start+width, lineWidth)
		parts = append(parts, ansi.Cut(line, start, end))
	}
	return parts
}

func withScrollbar(content string, width int, height int, percent float64) string {
	if height <= 0 {
		return content
	}
	lines := strings.Split(content, "\n")
	if len(lines) < height {
		lines = append(lines, make([]string, height-len(lines))...)
	}

	if len(lines) > height {
		lines = lines[:height]
	}
	thumb := int(percent * float64(max(height-1, 0)))
	thumb = max(thumb, 0)
	if thumb >= height {
		thumb = height - 1
	}
	inactive := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("┊")
	active := lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render("▯")
	out := make([]string, 0, height)
	for i := range height {
		line := ansi.Cut(lines[i], 0, width)
		if pad := width - ansi.StringWidth(line); pad > 0 {
			line += strings.Repeat(" ", pad)
		}
		bar := inactive
		if i == thumb {
			bar = active
		}
		out = append(out, line+bar)
	}
	return strings.Join(out, "\n")
}

func appendWithCap(lines []string, line string, max int) []string {
	lines = append(lines, line)
	if max <= 0 || len(lines) <= max {
		return lines
	}
	excess := len(lines) - max
	return lines[excess:]
}

func fitHeight(s string, height int) string {
	if height <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) < height {
		lines = append(lines, make([]string, height-len(lines))...)
	}

	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func procStateText(info procInfo) string {
	if info.running {
		return "running"
	}
	return "stopped"
}

func exitSummary(err error, code int) string {
	if err == nil {
		return "exit=0"
	}

	if errors.Is(err, context.Canceled) {
		return "canceled"
	}

	if code >= 0 {
		return fmt.Sprintf("exit=%d", code)
	}
	return "error"
}
