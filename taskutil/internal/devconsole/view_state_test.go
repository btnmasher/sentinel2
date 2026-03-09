package devconsole

import (
	"context"
	"strings"
	"testing"

	"sentinel2-taskutil/internal/project"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestAppendWithCap(t *testing.T) {
	got := appendWithCap([]string{"a", "b"}, "c", 2)
	if len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Fatalf("appendWithCap() = %#v, want [b c]", got)
	}
}

func TestFocusedName(t *testing.T) {
	m := viewState{focus: 0}

	if got := m.focusedName(); got != "frontend" {
		t.Fatalf("focusedName() = %q, want frontend", got)
	}
	m.focus = 1
	if got := m.focusedName(); got != "backend" {
		t.Fatalf("focusedName() = %q, want backend", got)
	}
}

func TestExitSummary(t *testing.T) {
	if got := exitSummary(nil, 0); got != "exit=0" {
		t.Fatalf("exitSummary(nil,0) = %q", got)
	}

	if got := exitSummary(context.Canceled, -1); got != "canceled" {
		t.Fatalf("exitSummary(canceled,-1) = %q", got)
	}

	if got := exitSummary(context.DeadlineExceeded, 3); got != "exit=3" {
		t.Fatalf("exitSummary(err,3) = %q", got)
	}

	if got := exitSummary(context.DeadlineExceeded, -1); got != "error" {
		t.Fatalf("exitSummary(err,-1) = %q", got)
	}
}

func TestUpdate_MouseClickFocusAndWheelScrollAffectsFocusedPane(t *testing.T) {
	m := newTestViewState(110, 16)
	for i := range 40 {
		m.appendLine("backend", "backend line "+strings.Repeat("x", i%5))
	}
	m.backend.GotoBottom()

	headerRows := len(m.headerLines(m.width))
	leftW, _ := splitPaneWidths(m.width)
	clickBackend := tea.MouseMsg(tea.MouseEvent{
		X:      leftW + 1,
		Y:      headerRows + 2,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	})

	model, _ := m.Update(clickBackend)
	m = model.(viewState)
	if m.focus != 1 {
		t.Fatalf("focus = %d, want 1 (backend)", m.focus)
	}
	m.selection = dragSelection{}

	m.followFrontend = true
	m.followBackend = true
	wheelBackend := tea.MouseMsg(tea.MouseEvent{
		X:      leftW + 1,
		Y:      headerRows + 2,
		Action: tea.MouseActionMotion,
		Button: tea.MouseButtonWheelUp,
	})
	model, _ = m.Update(wheelBackend)
	m = model.(viewState)

	if m.followBackend {
		t.Fatalf("followBackend should be false after backend wheel scroll")
	}

	if !m.followFrontend {
		t.Fatalf("followFrontend should remain true after backend wheel scroll")
	}
}

func TestSelectedText_UsesScrolledWrappedSourceLines(t *testing.T) {
	m := newTestViewState(50, 14)
	m.frontend.Width = 10
	m.frontend.Height = 4
	m.frontendLines = []string{
		"first",
		strings.Repeat("b", 35),
		"third",
	}
	m.refreshViewportContent()

	m.frontend.SetYOffset(1)
	start := m.lineIndexForProc("frontend", 0)
	end := m.lineIndexForProc("frontend", 1)
	m.selection = dragSelection{
		active:    true,
		proc:      "frontend",
		startLine: start,
		startCol:  2,
		endLine:   end,
		endCol:    4,
		moved:     true,
	}
	got := m.selectedText()
	want := strings.Repeat("b", 13)
	if got != want {
		t.Fatalf("selectedText() = %q, want %q", got, want)
	}

	m.selection = dragSelection{
		active:    true,
		proc:      "frontend",
		startLine: start,
		startCol:  0,
		endLine:   start,
		endCol:    9,
		moved:     true,
	}
	got = m.selectedText()
	want = strings.Repeat("b", 10)
	if got != want {
		t.Fatalf("selectedText() single wrapped line = %q, want %q", got, want)
	}

	m.selection = dragSelection{
		active:    true,
		proc:      "frontend",
		startLine: start,
		startCol:  0,
		endLine:   m.lineIndexForProc("frontend", 4),
		endCol:    4,
		moved:     true,
	}
	got = m.selectedText()
	want = strings.Repeat("b", 35) + "\nthird"
	if got != want {
		t.Fatalf("selectedText() across source lines = %q, want %q", got, want)
	}
}

func TestLineMode_MouseHoverAndClickSelectsAndCopiesLine(t *testing.T) {
	m := newTestViewState(110, 16)
	m.appendLine("backend", "alpha")
	m.appendLine("backend", "beta line")

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = model.(viewState)
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = model.(viewState)

	headerRows := len(m.headerLines(m.width))
	leftW, _ := splitPaneWidths(m.width)
	y := headerRows + 2
	x := leftW + 1

	model, _ = m.Update(tea.MouseMsg(tea.MouseEvent{
		X:      x,
		Y:      y,
		Action: tea.MouseActionMotion,
	}))
	m = model.(viewState)
	if !m.hoverLine || m.hoverProc != "backend" {
		t.Fatalf("hover state = (%v,%q), want (true,backend)", m.hoverLine, m.hoverProc)
	}

	model, _ = m.Update(tea.MouseMsg(tea.MouseEvent{
		X:      x,
		Y:      y,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}))
	m = model.(viewState)
	if !m.lineMode || m.lineModeProc != "backend" {
		t.Fatalf("line mode state = (%v,%q), want (true,backend)", m.lineMode, m.lineModeProc)
	}

	if !strings.Contains(m.status, "copied line") || !strings.Contains(m.status, "backend") {
		t.Fatalf("status = %q, want copied line status for backend", m.status)
	}
}

func TestUpdate_LifecycleStatusTransitions(t *testing.T) {
	m := newTestViewState(110, 16)

	model, _ := m.Update(actionDoneMsg{message: "rebuild backend + restart in progress"})
	m = model.(viewState)
	if m.status != "rebuild backend + restart in progress" {
		t.Fatalf("status = %q", m.status)
	}

	model, _ = m.Update(actionDoneMsg{message: "rebuild backend + restart succeeded", markRunning: []string{"backend"}})
	m = model.(viewState)
	if m.status != "rebuild backend + restart succeeded" {
		t.Fatalf("status = %q", m.status)
	}

	if !m.procs["backend"].running {
		t.Fatalf("backend should be marked running after success")
	}

	model, _ = m.Update(actionDoneMsg{message: "rebuild backend + restart failed: boom"})
	m = model.(viewState)
	if !strings.Contains(m.status, "failed") {
		t.Fatalf("status = %q, want failed", m.status)
	}

	model, _ = m.Update(procStartedMsg{proc: "frontend", pid: 77})
	m = model.(viewState)
	if !m.procs["frontend"].running || m.procs["frontend"].pid != 77 {
		t.Fatalf("frontend proc state = %#v", m.procs["frontend"])
	}

	if len(m.frontendLines) == 0 {
		t.Fatalf("expected session marker line after proc start")
	}
	_, startColor, startMsg, ok := parseMarkerToken(m.frontendLines[len(m.frontendLines)-1])
	if !ok {
		t.Fatalf("expected marker token in frontend log after start")
	}

	if startColor != "10" || !strings.Contains(startMsg, "started") {
		t.Fatalf("unexpected start marker color/message: color=%q msg=%q", startColor, startMsg)
	}

	model, _ = m.Update(procExitMsg{proc: "frontend", code: 2, err: context.DeadlineExceeded})
	m = model.(viewState)
	if m.procs["frontend"].running {
		t.Fatalf("frontend should not be running after exit")
	}

	if m.procs["frontend"].lastExit != "exit=2" {
		t.Fatalf("lastExit = %q, want exit=2", m.procs["frontend"].lastExit)
	}

	if !strings.Contains(m.status, "frontend exit=2") {
		t.Fatalf("status = %q", m.status)
	}

	if len(m.frontendLines) < 2 {
		t.Fatalf("expected stop marker line after proc exit")
	}
	_, stopColor, stopMsg, ok := parseMarkerToken(m.frontendLines[len(m.frontendLines)-1])
	if !ok {
		t.Fatalf("expected marker token in frontend log after exit")
	}

	if stopColor != "9" || !strings.Contains(stopMsg, "stopped (exit=2)") {
		t.Fatalf("unexpected stop marker color/message: color=%q msg=%q", stopColor, stopMsg)
	}
}

func TestUpdate_StaleProcExitIgnoredAfterRestart(t *testing.T) {
	m := newTestViewState(110, 16)

	model, _ := m.Update(procStartedMsg{proc: "backend", pid: 100})
	m = model.(viewState)
	model, _ = m.Update(procStartedMsg{proc: "backend", pid: 200})
	m = model.(viewState)

	model, _ = m.Update(procExitMsg{proc: "backend", pid: 100, code: -1, err: context.Canceled})
	m = model.(viewState)

	if !m.procs["backend"].running {
		t.Fatalf("backend should remain running after stale exit")
	}

	if m.procs["backend"].pid != 200 {
		t.Fatalf("backend pid = %d, want 200", m.procs["backend"].pid)
	}

	if m.procs["backend"].lastExit != "" {
		t.Fatalf("lastExit = %q, want empty", m.procs["backend"].lastExit)
	}
}

func TestUpdate_DownAtBottomResumesFollow(t *testing.T) {
	m := newTestViewState(110, 16)
	for i := range 80 {
		m.appendLine("frontend", strings.Repeat("x", 20)+strings.Repeat("y", i%3))
	}
	m.focus = 0
	m.followFrontend = false
	m.frontend.GotoBottom()
	if !m.frontend.AtBottom() {
		t.Fatalf("frontend should be at bottom for test setup")
	}

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = model.(viewState)
	if !m.followFrontend {
		t.Fatalf("followFrontend should resume when scrolling down at bottom")
	}
}

func TestSelectedText_MarkerCopiesPlainText(t *testing.T) {
	m := newTestViewState(80, 14)
	m.frontend.Width = 40
	m.frontend.Height = 6
	m.frontendLines = []string{
		"alpha",
		markerToken("12:34:56", "10", "backend started"),
		"omega",
	}
	m.refreshViewportContent()

	start := 1
	end := 1
	m.selection = dragSelection{
		active:    true,
		proc:      "frontend",
		startLine: start,
		startCol:  0,
		endLine:   end,
		endCol:    39,
		moved:     true,
	}
	got := m.selectedText()
	if !strings.Contains(got, "12:34:56 backend started") {
		t.Fatalf("selectedText() = %q, want marker plain text", got)
	}
}

func TestMarkerLine_ReflowsOnResize(t *testing.T) {
	m := newTestViewState(110, 16)
	m.frontendLines = []string{
		markerToken("12:34:56", "10", "backend started"),
	}
	m.refreshViewportContent()
	wide := m.frontend.View()

	m.resize(70, 16)
	narrow := m.frontend.View()
	if wide == narrow {
		t.Fatalf("expected marker rendering to change after resize")
	}

	if !strings.Contains(ansi.Strip(narrow), "12:34:56  backend started") {
		t.Fatalf("narrow marker missing expected body: %q", ansi.Strip(narrow))
	}
}

func newTestViewState(width, height int) viewState {
	pm := &processManager{
		cfg:    project.Config{AppName: "sentinel2"},
		events: make(chan tea.Msg, 32),
	}
	m := newViewState(pm)
	m.resize(width, height)
	return m
}
