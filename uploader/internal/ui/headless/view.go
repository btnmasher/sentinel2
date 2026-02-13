package headless

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	tabActiveStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("27")).Padding(0, 1)
	tabInactiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Background(lipgloss.Color("236")).Padding(0, 1)
	modalBackdrop    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func (m *headlessModel) View() string {
	if m.width == 0 {
		return "initializing..."
	}

	base := m.renderBase()
	if m.filePickerOpen {
		return m.renderModalOverlay(base, m.renderFilePickerDialog())
	}
	if m.errorModalText != "" {
		return m.renderModalOverlay(base, m.renderErrorDialog())
	}
	if m.confirmQuit {
		return m.renderModalOverlay(base, m.renderQuitConfirmDialog())
	}
	return base
}

func (m *headlessModel) renderBase() string {
	header := titleStyle.Render("Sentinel2 Uploader (" + m.buildVersion + ")")
	tabs := m.renderTabs()

	var content string
	if m.tab == tabOverview {
		content = m.renderOverview()
	} else {
		content = m.renderSettings()
	}

	helpText := m.helpView.View(m.keys)
	if m.tab == tabSettings {
		helpText += " • ctrl+s save"
	}
	sections := []string{header, tabs, content}
	if m.tab == tabOverview && m.showLogs {
		m.fitLogViewportHeight([]string{header, tabs, content, helpText})
		logPanel := m.renderLogPanel()
		sections = append(sections, logPanel)
	}
	sections = append(sections, helpStyle.Render(helpText))
	root := strings.Join(sections, "\n\n")
	return m.renderFrame(root, m.contentWidth())
}

func (m *headlessModel) renderTabs() string {
	overview := tabInactiveStyle.Render("Overview")
	settings := tabInactiveStyle.Render("Settings")
	if m.tab == tabOverview {
		overview = tabActiveStyle.Render("Overview")
	} else {
		settings = tabActiveStyle.Render("Settings")
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, overview, " ", settings)
}

func (m *headlessModel) renderOverview() string {
	total := m.pageWidth()
	gap := 2
	m.resizePaneViewports()

	leftWidth, rightWidth, stacked := m.overviewPaneLayout(total)
	statusLine := "Status: " + m.renderStatus()
	leftRenderWidth := leftWidth
	if stacked {
		leftRenderWidth = total
	}

	leftContentWidth := leftRenderWidth - 4
	if leftContentWidth <= 0 {
		leftContentWidth = m.leftView.Width
	}
	// Keep spare columns to avoid edge-case overflow at exact breakpoints.
	if leftContentWidth > 2 {
		leftContentWidth -= 2
	}
	if leftContentWidth < 12 {
		leftContentWidth = 12
	}
	m.leftView.Width = leftContentWidth
	actionsLine := m.renderActionsRow(leftContentWidth)
	requiredLeftHeight := 1 + 2 + lipgloss.Height(actionsLine) // status + blank line + actions block
	if requiredLeftHeight < 6 {
		requiredLeftHeight = 6
	}
	if m.leftView.Height < requiredLeftHeight {
		m.leftView.Height = requiredLeftHeight
	}
	if !stacked && m.rightView.Height < requiredLeftHeight {
		m.rightView.Height = requiredLeftHeight
	}

	m.leftView.SetContent(strings.Join([]string{
		statusLine,
		actionsLine,
	}, "\n\n"))
	left := m.renderFrame(m.leftView.View(), leftRenderWidth)

	if stacked {
		m.rightView.SetContent(m.renderChannelPanelBody(total-4, m.rightView.Height))
		right := m.renderFrame(m.rightView.View(), rightWidth)
		layout := left + "\n\n" + right
		return lipgloss.NewStyle().Width(total).Render(layout)
	}
	remaining := total - lipgloss.Width(left) - gap
	if remaining < 24 {
		remaining = 24
	}
	m.rightView.SetContent(m.renderChannelPanelBody(remaining-4, m.rightView.Height))
	right := m.renderFrame(m.rightView.View(), remaining)
	layout := lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", gap), right)
	return lipgloss.NewStyle().Width(total).Render(layout)
}

func (m *headlessModel) overviewLeftFrameWidth(total int) int {
	statusLine := "Status: " + m.renderStatus()
	actionsLine := m.renderActionsRow(10_000)
	leftInner := maxInt(lipgloss.Width(statusLine), lipgloss.Width(actionsLine))
	leftInner = maxInt(leftInner, m.actionsRowPreferredWidth())
	leftWidth := leftInner + 6
	if leftWidth < 24 {
		leftWidth = 24
	}
	if leftWidth > total {
		leftWidth = total
	}
	return leftWidth
}

func (m *headlessModel) actionsRowPreferredWidth() int {
	connect := m.renderConnectToggle()
	logs := buttonStyle.Render("Hide Logs")
	quit := buttonStyle.Render("Quit")
	row := lipgloss.JoinHorizontal(lipgloss.Top, connect, " ", logs, " ", quit)
	return lipgloss.Width(row)
}

func (m *headlessModel) overviewPaneLayout(total int) (leftWidth int, rightWidth int, stacked bool) {
	gap := 2
	minRightWidth := 32
	leftWidth = m.overviewLeftFrameWidth(total)
	rightWidth = total - leftWidth - gap
	if total < 84 || rightWidth < minRightWidth {
		return leftWidth, total, true
	}
	return leftWidth, rightWidth, false
}

func (m *headlessModel) renderSettings() string {
	labels := []string{"Base URL", "Token", "Log Dir"}
	labelWidth := 9
	rows := make([]string, 0, len(m.inputs)+5)
	controlWidth := m.settingsView.Width - labelWidth - 2
	if controlWidth < 16 {
		controlWidth = 16
	}

	for i := range m.inputs {
		label := labels[i]
		if m.focus == i {
			label = focusStyle.Render("-> " + label)
		}
		m.inputs[i].Width = controlWidth
		rows = append(rows, fmt.Sprintf("%-*s %s", labelWidth, label+":", m.inputs[i].View()))
	}

	browseButton := buttonStyle.Render("Choose Folder")
	if m.focus == m.browseIndex() {
		browseButton = buttonFocusedStyle.Render("Choose Folder")
	}
	browseLine := lipgloss.NewStyle().PaddingLeft(labelWidth + 1).Render(browseButton)
	rows = append(rows, browseLine)

	auto := "[ ] Auto-connect"
	if m.autoConn {
		auto = "[x] Auto-connect"
	}
	autoLabel := "Auto"
	if m.focus == m.autoConnectIndex() {
		autoLabel = focusStyle.Render("-> Auto")
	}
	rows = append(rows, fmt.Sprintf("%-*s %s", labelWidth, autoLabel+":", auto))

	saveLabel := "Save"
	cancelLabel := "Cancel"
	if m.settingsDirty {
		saveLabel = buttonStyle.Render("Save")
		cancelLabel = buttonStyle.Render("Cancel")
	} else {
		saveLabel = buttonDisabledStyle.Render("Save")
		cancelLabel = buttonDisabledStyle.Render("Cancel")
	}
	if m.focus == m.saveIndex() {
		if m.settingsDirty {
			saveLabel = buttonFocusedStyle.Render("Save")
		} else {
			saveLabel = buttonDisabledFocusedStyle.Render("Save")
		}
	}
	if m.focus == m.cancelIndex() {
		if m.settingsDirty {
			cancelLabel = buttonFocusedStyle.Render("Cancel")
		} else {
			cancelLabel = buttonDisabledFocusedStyle.Render("Cancel")
		}
	}
	rows = append(rows, "")
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Left, saveLabel, " ", cancelLabel))
	if m.settingsDirty {
		rows = append(rows, helpStyle.Render("unsaved changes"))
	}

	m.resizePaneViewports()
	m.settingsView.SetContent(strings.Join(rows, "\n"))
	return m.renderFrame(m.settingsView.View(), m.pageWidth())
}

func (m *headlessModel) renderFilePickerDialog() string {
	title := titleStyle.Render("Select Log Directory")
	picker := m.filePicker.View()
	help := helpStyle.Render("up/down move • space open • enter select • left/backspace up • esc close")
	body := strings.Join([]string{title, picker, help}, "\n")
	width := minInt(m.pageWidth(), 96)
	return m.renderFrame(body, width)
}

func (m *headlessModel) renderChannelPanelBody(width int, height int) string {
	header := titleStyle.Render("Configured Channels")
	if !m.running && !m.connecting {
		placeholder := "Not connected"
		if width < 8 {
			width = 8
		}
		if height < 3 {
			height = 3
		}
		content := lipgloss.NewStyle().Width(width).Height(height - 1).AlignHorizontal(lipgloss.Center).AlignVertical(lipgloss.Center).Foreground(lipgloss.Color("245")).Render(placeholder)
		return header + "\n" + content
	}

	if len(m.channelHealth) == 0 {
		placeholder := "No channels configured"
		if m.healthDetail != "" {
			placeholder = m.healthDetail
		}
		if width < 8 {
			width = 8
		}
		if height < 3 {
			height = 3
		}
		content := lipgloss.NewStyle().Width(width).Height(height - 1).AlignHorizontal(lipgloss.Center).AlignVertical(lipgloss.Center).Foreground(lipgloss.Color("245")).Render(placeholder)
		return header + "\n" + content
	}

	lines := make([]string, 0, len(m.channelHealth))
	if width < 10 {
		width = 10
	}
	for _, row := range m.channelHealth {
		dot, style := channelDotStyle(row.Kind)
		nameWidth := width - 2
		if nameWidth < 1 {
			nameWidth = 1
		}
		prefix := style.Render(dot) + " "
		availableName := width - ansi.StringWidth(prefix)
		if availableName < 1 {
			availableName = 1
		}
		name := truncateDisplayWidth(row.Name, availableName)
		lines = append(lines, prefix+name)
	}

	body := strings.Join(lines, "\n")
	if m.healthDetail != "" {
		body += "\n" + helpStyle.Render(m.healthDetail)
	}
	return header + "\n" + body
}

func channelDotStyle(kind channelHealthKind) (string, lipgloss.Style) {
	dot := "●"
	switch kind {
	case channelActive:
		return dot, lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	case channelWarn:
		return dot, lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	case channelStale:
		return dot, lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	default:
		return dot, lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	}
}

func (m *headlessModel) renderConnectToggle() string {
	if !m.running && !m.connecting && !m.canConnect() {
		connect := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Connect")
		disconnect := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Disconnect")
		content := connect + segmentBaseStyle.Render("|") + disconnect
		if m.focus == m.connectIndex() {
			return buttonDisabledFocusedStyle.Render(content)
		}
		return buttonDisabledStyle.Render(content)
	}
	if m.connecting {
		connecting := rainbowText("Connecting...", m.animPhase)
		content := segmentOnStyle.Render(connecting) + segmentBaseStyle.Render("|") + segmentOffStyle.Render("Disconnect")
		if m.focus == m.connectIndex() {
			return buttonFocusedStyle.Render(content)
		}
		return buttonStyle.Render(content)
	}

	connect := segmentOffStyle.Render("Connect")
	disconnect := segmentOffStyle.Render("Disconnect")
	if m.running {
		disconnect = segmentOnStyle.Render("Disconnect")
	} else {
		connect = segmentOnStyle.Render("Connect")
	}
	content := connect + segmentBaseStyle.Render("|") + disconnect
	if m.focus == m.connectIndex() {
		return buttonFocusedStyle.Render(content)
	}
	return buttonStyle.Render(content)
}

func (m *headlessModel) renderLogsButton() string {
	label := "Logs"
	if m.showLogs {
		label = "Hide Logs"
	}
	if m.focus == m.logsIndex() {
		return buttonFocusedStyle.Render(label)
	}
	return buttonStyle.Render(label)
}

func (m *headlessModel) renderLogsDebugToggle() string {
	check := "[ ] Debug"
	if m.debugOn {
		check = "[x] Debug"
	}
	if m.focus == m.logsDebugIndex() {
		return buttonFocusedStyle.Render(check)
	}
	return buttonStyle.Render(check)
}

func (m *headlessModel) renderLogPanel() string {
	followHint := helpStyle.Render("ctrl+f follow")
	toolbar := lipgloss.JoinHorizontal(lipgloss.Center, titleStyle.Render("Logs"), "  ", m.renderLogsDebugToggle(), "  ", followHint)
	content := m.logView.View()
	withBar := m.withScrollBar(content, m.logView.Width, m.logView.Height, m.logView.ScrollPercent())
	return m.renderFrame(toolbar+"\n"+withBar, m.pageWidth())
}

func (m *headlessModel) withScrollBar(content string, width int, height int, percent float64) string {
	if height <= 0 {
		return content
	}
	if width < 1 {
		width = 1
	}
	lines := strings.Split(content, "\n")
	if len(lines) < height {
		pad := make([]string, 0, height-len(lines))
		for i := 0; i < height-len(lines); i++ {
			pad = append(pad, "")
		}
		lines = append(lines, pad...)
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	thumb := int(percent * float64(height-1))
	if thumb < 0 {
		thumb = 0
	}
	if thumb >= height {
		thumb = height - 1
	}
	barInactive := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("┊")
	barActive := lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render("▯")

	out := make([]string, 0, height)
	for i := 0; i < height; i++ {
		bar := barInactive
		if i == thumb {
			bar = barActive
		}
		text := ansi.Cut(lines[i], 0, width)
		if pad := width - ansi.StringWidth(text); pad > 0 {
			text += strings.Repeat(" ", pad)
		}
		out = append(out, text+" "+bar)
	}
	return strings.Join(out, "\n")
}

func (m *headlessModel) renderQuitButton() string {
	label := "Quit"
	if m.focus == m.quitIndex() {
		return buttonFocusedStyle.Render(label)
	}
	return buttonStyle.Render(label)
}

func (m *headlessModel) renderQuitConfirmDialog() string {
	cancelButton := buttonStyle.Render("Cancel")
	quitButton := buttonStyle.Render("Quit")
	if m.confirmQuitChoice == 0 {
		cancelButton = buttonFocusedStyle.Render("Cancel")
	} else {
		quitButton = buttonFocusedStyle.Render("Quit")
	}
	body := strings.Join([]string{
		titleStyle.Render("Quit while connected?"),
		"This will stop the uploader connection.",
		cancelButton + "  " + quitButton,
		helpStyle.Render("tab/arrow switch • enter confirms"),
	}, "\n")
	return m.renderFrame(body, minInt(m.contentWidth()-8, 72))
}

func (m *headlessModel) renderErrorDialog() string {
	body := strings.Join([]string{
		errorStyle.Render("Error"),
		m.errorModalText,
		helpStyle.Render("Press Enter or Esc to close"),
	}, "\n")
	return m.renderFrame(body, minInt(m.contentWidth()-8, 78))
}

func (m *headlessModel) renderModalOverlay(base string, dialog string) string {
	faded := modalBackdrop.Render(base)
	overlay := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	return faded + "\n" + overlay
}

func (m *headlessModel) renderActionsRow(maxWidth int) string {
	segments := []string{m.renderConnectToggle(), m.renderLogsButton(), m.renderQuitButton()}
	if maxWidth <= 0 {
		maxWidth = 1
	}
	lines := make([]string, 0, len(segments))
	rowParts := make([]string, 0, len(segments))
	joinRow := func(parts []string) string {
		if len(parts) == 0 {
			return ""
		}
		row := parts[0]
		for i := 1; i < len(parts); i++ {
			row = lipgloss.JoinHorizontal(lipgloss.Top, row, " ", parts[i])
		}
		return row
	}
	for _, seg := range segments {
		if len(rowParts) == 0 {
			rowParts = append(rowParts, seg)
			continue
		}
		candidateParts := append(append([]string(nil), rowParts...), seg)
		candidate := joinRow(candidateParts)
		if lipgloss.Width(candidate) <= maxWidth {
			rowParts = candidateParts
			continue
		}
		lines = append(lines, joinRow(rowParts))
		rowParts = []string{seg}
	}
	if len(rowParts) > 0 {
		lines = append(lines, joinRow(rowParts))
	}
	return strings.Join(lines, "\n")
}

func (m *headlessModel) renderStatus() string {
	switch m.kind {
	case statusConnected:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(m.status)
	case statusConnecting:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Render(m.status)
	case statusStopping:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(m.status)
	case statusError:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.status)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(m.status)
	}
}

func rainbowText(value string, phase int) string {
	var b strings.Builder
	for i, r := range value {
		position := float64(i)/2.0 - float64(phase)*0.4
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(rainbowAt(position)))
		b.WriteString(style.Render(string(r)))
	}
	return b.String()
}

func (m *headlessModel) contentWidth() int {
	w := m.width
	if w < 40 {
		w = 40
	}
	return w
}

func (m *headlessModel) pageWidth() int {
	w := m.contentWidth() - 4 // account for the outer frame border+padding
	if w < 24 {
		w = 24
	}
	return w
}

func (m *headlessModel) renderFrame(content string, width int) string {
	if !m.imgay {
		innerWidth := width - 4 // rounded border (2) + horizontal padding (2)
		if innerWidth < 1 {
			innerWidth = 1
		}
		return panelStyle.Width(innerWidth).Render(content)
	}
	return m.renderRainbowFrame(content, width)
}

func (m *headlessModel) renderRainbowFrame(content string, width int) string {
	lines := strings.Split(content, "\n")
	innerWidth := width - 2
	if innerWidth < 20 {
		innerWidth = 20
	}
	clamp := lipgloss.NewStyle().MaxWidth(innerWidth).Width(innerWidth)

	top := colorizeBorderLine("╭", "─", "╮", innerWidth, 0, m.animPhase)
	bottom := colorizeBorderLine("╰", "─", "╯", innerWidth, len(lines)+1, m.animPhase)
	framed := make([]string, 0, len(lines)+2)
	framed = append(framed, top)
	for i, line := range lines {
		padded := clamp.Render(line)
		left := colorizeBorderChar("│", 0, i+1, m.animPhase)
		right := colorizeBorderChar("│", innerWidth+1, i+1, m.animPhase)
		framed = append(framed, left+padded+right)
	}
	framed = append(framed, bottom)
	return strings.Join(framed, "\n")
}

func colorizeBorderLine(left, fill, right string, width int, y int, phase int) string {
	var b strings.Builder
	b.WriteString(colorizeBorderChar(left, 0, y, phase))
	for x := 1; x <= width; x++ {
		b.WriteString(colorizeBorderChar(fill, x, y, phase))
	}
	b.WriteString(colorizeBorderChar(right, width+1, y, phase))
	return b.String()
}

func colorizeBorderChar(ch string, x int, y int, phase int) string {
	position := float64(x+y)/3.0 - float64(phase)*0.35
	return lipgloss.NewStyle().Foreground(lipgloss.Color(rainbowAt(position))).Render(ch)
}

func rainbowAt(position float64) string {
	n := float64(len(rainbowPalette))
	if n == 0 {
		return "#ffffff"
	}
	wrapped := math.Mod(position, n)
	if wrapped < 0 {
		wrapped += n
	}
	i0 := int(math.Floor(wrapped))
	i1 := (i0 + 1) % len(rainbowPalette)
	t := wrapped - float64(i0)
	c := lerpRGB(rainbowPalette[i0], rainbowPalette[i1], t)
	return fmt.Sprintf("#%02x%02x%02x", c.r, c.g, c.b)
}

func lerpRGB(a rgb, b rgb, t float64) rgb {
	return rgb{
		r: uint8(float64(a.r) + (float64(b.r)-float64(a.r))*t),
		g: uint8(float64(a.g) + (float64(b.g)-float64(a.g))*t),
		b: uint8(float64(a.b) + (float64(b.b)-float64(a.b))*t),
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func truncateDisplayWidth(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if ansi.StringWidth(value) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	limit := width - ansi.StringWidth("…")
	if limit < 0 {
		limit = 0
	}
	var b strings.Builder
	current := 0
	for _, r := range value {
		w := ansi.StringWidth(string(r))
		if current+w > limit {
			break
		}
		b.WriteRune(r)
		current += w
	}
	return b.String() + "…"
}
