package devconsole

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"golang.design/x/clipboard"
)

func (m viewState) highlightSelection(view string, vp viewport.Model, proc string) string {
	start, startCol, end, endCol := -1, 0, -2, 0
	if m.selection.active && m.selection.moved && m.selection.proc == proc {
		start, startCol, end, endCol = m.selectionSpan()
	}
	visibleStart := vp.YOffset
	visibleEnd := vp.YOffset + vp.Height - 1
	lineModeActive := m.lineMode && m.lineModeProc == proc
	if !lineModeActive && (end < visibleStart || start > visibleEnd) {
		return view
	}
	lines := strings.Split(view, "\n")
	selStyle := lipgloss.NewStyle().Background(lipgloss.Color("238"))
	hoverStyle := lipgloss.NewStyle().Background(lipgloss.Color("240"))
	_, sourceMap := m.wrappedLinesWithSourceForProc(proc)
	for i := range lines {
		global := visibleStart + i
		highlight := global >= start && global <= end
		if lineModeActive && global >= 0 && global < len(sourceMap) && sourceMap[global] == m.lineModeSource {
			highlight = true
		}
		hover := m.lineMode && m.hoverLine && m.hoverProc == proc && global >= 0 && global < len(sourceMap) && sourceMap[global] == m.hoverSource
		if !highlight && !hover {
			continue
		}
		text := ansi.Strip(lines[i])
		if highlight && !lineModeActive {
			lines[i] = highlightLineSlice(text, global, start, startCol, end, endCol, selStyle)
		} else if highlight {
			lines[i] = selStyle.Render(text)
		} else {
			lines[i] = hoverStyle.Render(text)
		}
	}
	return strings.Join(lines, "\n")
}

func (m viewState) selectedText() string {
	start, startCol, end, endCol := m.selectionSpan()
	if m.selection.proc == "" || end < start {
		return ""
	}
	rawLines := m.rawLinesForProc(m.selection.proc)
	if len(rawLines) == 0 {
		return ""
	}
	segments := m.wrappedSegmentsForProc(m.selection.proc)
	if len(segments) == 0 {
		return ""
	}
	if start < 0 {
		start = 0
	}
	if end >= len(segments) {
		end = len(segments) - 1
	}
	if start > end {
		return ""
	}

	startSeg := segments[start]
	endSeg := segments[end]
	sourceStart := startSeg.source
	sourceEnd := endSeg.source
	if sourceStart > sourceEnd {
		sourceStart, sourceEnd = sourceEnd, sourceStart
		startSeg, endSeg = endSeg, startSeg
		startCol, endCol = endCol, startCol
	}
	if sourceStart < 0 {
		sourceStart = 0
	}
	if sourceEnd >= len(rawLines) {
		sourceEnd = len(rawLines) - 1
	}
	startAbsCol := max(startSeg.startCol+startCol, 0)
	endAbsCol := max(endSeg.startCol+endCol, 0)

	out := make([]string, 0, sourceEnd-sourceStart+1)
	for i := sourceStart; i <= sourceEnd; i++ {
		line := ansi.Strip(rawLines[i])
		width := ansi.StringWidth(line)
		switch {
		case sourceStart == sourceEnd:
			s := min(startAbsCol, width)
			e := min(endAbsCol+1, width)
			out = append(out, ansi.Cut(line, s, e))
		case i == sourceStart:
			s := min(startAbsCol, width)
			out = append(out, ansi.Cut(line, s, width))
		case i == sourceEnd:
			e := min(endAbsCol+1, width)
			out = append(out, ansi.Cut(line, 0, e))
		default:
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

func (m viewState) rawLinesForProc(proc string) []string {
	if proc == "frontend" {
		return m.frontendLines
	}
	return m.backendLines
}

func (m viewState) wrappedLinesWithSourceForProc(proc string) ([]string, []int) {
	raw := m.rawLinesForProc(proc)
	width := minViewportWidth
	if proc == "frontend" {
		width = m.frontend.Width
	} else {
		width = m.backend.Width
	}
	return wrappedLinesWithSource(raw, width)
}

func wrappedLinesWithSource(raw []string, width int) ([]string, []int) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(raw))
	src := make([]int, 0, len(raw))
	for i, line := range raw {
		parts := wrapLineForWidth(line, width)
		if len(parts) == 0 {
			parts = []string{""}
		}
		for _, p := range parts {
			out = append(out, p)
			src = append(src, i)
		}
	}
	return out, src
}

type wrappedSegment struct {
	source   int
	startCol int
	endCol   int
}

func (m viewState) wrappedSegmentsForProc(proc string) []wrappedSegment {
	raw := m.rawLinesForProc(proc)
	width := minViewportWidth
	if proc == "frontend" {
		width = m.frontend.Width
	} else {
		width = m.backend.Width
	}
	return wrappedSegments(raw, width)
}

func wrappedSegments(raw []string, width int) []wrappedSegment {
	if len(raw) == 0 {
		return nil
	}
	out := make([]wrappedSegment, 0, len(raw))
	if width <= 0 {
		width = minViewportWidth
	}
	for i, line := range raw {
		plain := ansi.Strip(line)
		lineWidth := ansi.StringWidth(plain)
		if lineWidth == 0 {
			out = append(out, wrappedSegment{source: i, startCol: 0, endCol: 0})
			continue
		}
		for start := 0; start < lineWidth; start += width {
			end := min(start+width, lineWidth)
			out = append(out, wrappedSegment{source: i, startCol: start, endCol: end})
		}
	}
	return out
}

func (m viewState) selectionBounds() (int, int) {
	start, end := m.selection.startLine, m.selection.endLine
	if start > end {
		start, end = end, start
	}
	return start, end
}

func (m viewState) selectionSpan() (startLine int, startCol int, endLine int, endCol int) {
	sLine, sCol := m.selection.startLine, m.selection.startCol
	eLine, eCol := m.selection.endLine, m.selection.endCol
	if sLine > eLine || (sLine == eLine && sCol > eCol) {
		sLine, eLine = eLine, sLine
		sCol, eCol = eCol, sCol
	}
	return sLine, max(sCol, 0), eLine, max(eCol, 0)
}

func (m viewState) hitTestMouse(x, y int) (slot int, proc string, line int, col int, ok bool) {
	if m.fullscreen {
		return m.focus, m.focusedProc(), -1, -1, false
	}
	headerRows := len(m.headerLines(m.width))
	rowTop := headerRows
	if y < rowTop {
		return 0, "", -1, -1, false
	}
	if m.verticalSplit {
		leftW, rightW := splitPaneWidths(m.width)
		rowHeight := max(m.height-headerRows, minBodyHeight)
		paneBottom := rowTop + rowHeight - 1
		if y < rowTop || y > paneBottom {
			return 0, "", -1, -1, false
		}
		contentTop := rowTop + 2
		if x >= 0 && x < leftW {
			first, _ := m.slotProcs()
			return 0, first, m.lineIndexForProc(first, max(y-contentTop, 0)), max(x-1, 0), true
		}
		if x >= leftW && x < leftW+rightW {
			_, second := m.slotProcs()
			return 1, second, m.lineIndexForProc(second, max(y-contentTop, 0)), max(x-leftW-1, 0), true
		}
		return 0, "", -1, -1, false
	}
	focusTop := m.focus == 0
	topShowBottom := focusTop
	bottomShowTop := !focusTop
	first, second := m.slotProcs()
	topInnerHeight := m.procHeight(first)
	bottomInnerHeight := m.procHeight(second)
	topContentTop := rowTop + 2          // top border + title
	topOuterHeight := topInnerHeight + 2 // title + top border
	if topShowBottom {
		topOuterHeight++
	}
	bottomPaneTop := rowTop + topOuterHeight
	bottomContentTop := bottomPaneTop + 1 // title row
	if bottomShowTop {
		bottomContentTop++
	}
	if x < 0 || x >= m.width {
		return 0, "", -1, -1, false
	}
	if y >= topContentTop && y < topContentTop+topInnerHeight {
		return 0, first, m.lineIndexForProc(first, y-topContentTop), max(x-1, 0), true
	}
	if y >= bottomContentTop && y < bottomContentTop+bottomInnerHeight {
		return 1, second, m.lineIndexForProc(second, y-bottomContentTop), max(x-1, 0), true
	}
	return 0, "", -1, -1, false
}

func (m viewState) procHeight(proc string) int {
	if proc == "frontend" {
		return m.frontend.Height
	}
	return m.backend.Height
}

func (m viewState) lineIndexForProc(proc string, localY int) int {
	if localY < 0 {
		localY = 0
	}
	if proc == "frontend" {
		return m.frontend.YOffset + localY
	}
	return m.backend.YOffset + localY
}

func (m *viewState) toggleLineMode() {
	if m.lineMode {
		m.lineMode = false
		m.hoverLine = false
		m.hoverProc = ""
		m.status = "line selection mode off"
		m.resize(m.width, m.height)
		return
	}
	m.lineMode = true
	m.lineModeProc = m.focusedProc()
	m.hoverLine = false
	m.hoverProc = ""
	if src, ok := m.sourceIndexAtWrapped(m.lineModeProc, m.currentWrappedTop(m.lineModeProc)); ok {
		m.lineModeSource = src
	} else {
		m.lineModeSource = 0
	}
	m.ensureLineModeVisible()
	m.status = "line selection mode on"
	m.resize(m.width, m.height)
}

func (m *viewState) setLineModeAt(proc string, wrappedLine int) {
	m.lineMode = true
	m.lineModeProc = proc
	if src, ok := m.sourceIndexAtWrapped(proc, wrappedLine); ok {
		m.lineModeSource = src
	}
	m.ensureLineModeVisible()
	m.resize(m.width, m.height)
}

func (m *viewState) moveLineMode(delta int) {
	if !m.lineMode {
		return
	}
	raw := m.rawLinesForProc(m.lineModeProc)
	if len(raw) == 0 {
		m.lineModeSource = 0
		return
	}
	m.lineModeSource = max(0, min(m.lineModeSource+delta, len(raw)-1))
	m.ensureLineModeVisible()
}

func (m *viewState) ensureLineModeVisible() {
	if !m.lineMode {
		return
	}
	_, sourceMap := m.wrappedLinesWithSourceForProc(m.lineModeProc)
	if len(sourceMap) == 0 {
		return
	}
	first, last := -1, -1
	for i, src := range sourceMap {
		if src != m.lineModeSource {
			continue
		}
		if first == -1 {
			first = i
		}
		last = i
	}
	if first == -1 {
		return
	}
	if m.lineModeProc == "frontend" {
		if m.frontend.YOffset > first {
			m.frontend.SetYOffset(first)
			return
		}
		bottom := m.frontend.YOffset + m.frontend.Height - 1
		if last > bottom {
			m.frontend.SetYOffset(max(last-m.frontend.Height+1, 0))
		}
		return
	}
	if m.backend.YOffset > first {
		m.backend.SetYOffset(first)
		return
	}
	bottom := m.backend.YOffset + m.backend.Height - 1
	if last > bottom {
		m.backend.SetYOffset(max(last-m.backend.Height+1, 0))
	}
}

func (m viewState) sourceIndexAtWrapped(proc string, wrapped int) (int, bool) {
	_, sourceMap := m.wrappedLinesWithSourceForProc(proc)
	if wrapped < 0 || wrapped >= len(sourceMap) {
		return 0, false
	}
	return sourceMap[wrapped], true
}

func (m viewState) currentWrappedTop(proc string) int {
	if proc == "frontend" {
		return m.frontend.YOffset
	}
	return m.backend.YOffset
}

func writeClipboard(text string) error {
	clipboardInitOnce.Do(func() {
		clipboardInitErr = clipboard.Init()
	})
	if clipboardInitErr == nil {
		clipboard.Write(clipboard.FmtText, []byte(text))
		return nil
	}
	_, err := os.Stdout.WriteString(ansi.SetSystemClipboard(text))
	if err != nil {
		return fmt.Errorf("native clipboard unavailable (%v); osc52 fallback failed: %w", clipboardInitErr, err)
	}
	return nil
}

func highlightLineSlice(line string, globalLine, startLine, startCol, endLine, endCol int, style lipgloss.Style) string {
	width := ansi.StringWidth(line)
	if width <= 0 {
		return line
	}
	s, e := 0, width
	switch {
	case startLine == endLine:
		s, e = startCol, endCol+1
	case globalLine == startLine:
		s, e = startCol, width
	case globalLine == endLine:
		s, e = 0, endCol+1
	}
	s = min(max(s, 0), width)
	e = min(max(e, 0), width)
	if e <= s {
		return line
	}
	return ansi.Cut(line, 0, s) + style.Render(ansi.Cut(line, s, e)) + ansi.Cut(line, e, width)
}
