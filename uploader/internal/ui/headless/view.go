package headless

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *headlessModel) View() string {
	if m.width == 0 {
		return "initializing..."
	}

	labels := []string{"Base URL", "Token", "Log Dir"}
	rows := make([]string, 0, len(m.inputs)+2)
	for i := range m.inputs {
		label := labels[i]
		if m.focus == i {
			label = focusStyle.Render("-> " + label)
		}
		rows = append(rows, fmt.Sprintf("%-8s %s", label+":", m.inputs[i].View()))
	}
	logToggle := "[ ] Show debug logs"
	if m.showLogs {
		logToggle = "[x] Show debug logs"
	}
	logLabel := "Debug"
	if m.focus == m.debugIndex() {
		logLabel = focusStyle.Render("-> Debug")
	}
	rows = append(rows, fmt.Sprintf("%-8s %s", logLabel+":", logToggle))
	autoToggle := "[ ] Connect on startup"
	if m.autoConn {
		autoToggle = "[x] Connect on startup"
	}
	autoLabel := "Startup"
	if m.focus == m.autoConnectIndex() {
		autoLabel = focusStyle.Render("-> Startup")
	}
	rows = append(rows, fmt.Sprintf("%-8s %s", autoLabel+":", autoToggle))

	rows = append(rows, m.renderActionsRow())

	header := titleStyle.Render(fmt.Sprintf("Sentinel2 Uploader TUI (%s)", m.buildVersion))
	status := "Status: " + m.renderStatus()

	form := m.renderFrame(strings.Join([]string{
		header,
		status,
		strings.Join(rows, "\n"),
		m.renderChannels(),
		helpStyle.Render("tab/shift+tab move • enter/space activate selected control • ctrl+c quits"),
	}, "\n\n"), m.contentWidth())

	sections := []string{form}
	if m.showLogs {
		logPanel := m.renderFrame(
			titleStyle.Render("Debug Logs")+"\n"+m.logView.View(),
			m.contentWidth(),
		)
		sections = append(sections, logPanel)
	}

	if m.errText != "" {
		sections = append(sections, errorStyle.Render("Error: "+m.errText))
	}
	if m.confirmQuit {
		sections = append(sections, m.renderQuitConfirmDialog())
	}
	return strings.Join(sections, "\n")
}

func (m *headlessModel) renderChannels() string {
	if len(m.channels) == 0 {
		return "Channels:\n  (none loaded)"
	}
	lines := make([]string, 0, len(m.channels)+1)
	lines = append(lines, "Channels:")
	for _, name := range m.channels {
		lines = append(lines, "  - "+name)
	}
	return strings.Join(lines, "\n")
}

func (m *headlessModel) renderConnectToggle() string {
	if !m.running && !m.connecting && !m.canConnect() {
		connect := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Connect")
		disconnect := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Disconnect")
		content := connect + segmentBaseStyle.Render("|") + disconnect
		if m.focus == m.connectIndex() {
			return buttonFocusedStyle.Render(content)
		}
		return buttonStyle.Render(content)
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
		helpStyle.Render("tab/arrow keys switch • enter confirms"),
	}, "\n")
	return m.renderFrame(body, m.contentWidth())
}

func (m *headlessModel) renderActionsRow() string {
	actionsLabel := "Actions"
	if m.focus == m.connectIndex() || m.focus == m.quitIndex() {
		actionsLabel = focusStyle.Render("-> Actions")
	}
	controls := lipgloss.JoinHorizontal(lipgloss.Top, m.renderConnectToggle(), " ", m.renderQuitButton())
	return actionsLabel + ":\n" + controls
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
	w := m.width - 8
	if w < 40 {
		w = 40
	}
	return w
}

func (m *headlessModel) renderFrame(content string, width int) string {
	if !m.imgay {
		return panelStyle.Width(width).Render(content)
	}
	return m.renderRainbowFrame(content, width)
}

func (m *headlessModel) renderRainbowFrame(content string, width int) string {
	lines := strings.Split(content, "\n")
	innerWidth := width - 2
	if innerWidth < 20 {
		innerWidth = 20
	}

	maxLineWidth := 0
	for _, line := range lines {
		if lw := lipgloss.Width(line); lw > maxLineWidth {
			maxLineWidth = lw
		}
	}
	if maxLineWidth > innerWidth {
		innerWidth = maxLineWidth
	}

	top := colorizeBorderLine("╭", "─", "╮", innerWidth, 0, m.animPhase)
	bottom := colorizeBorderLine("╰", "─", "╯", innerWidth, len(lines)+1, m.animPhase)
	framed := make([]string, 0, len(lines)+2)
	framed = append(framed, top)
	for i, line := range lines {
		padded := lipgloss.NewStyle().Width(innerWidth).Render(line)
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
