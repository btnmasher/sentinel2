package devconsole

import (
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	minPaneWidth       = 10
	initialPaneHeight  = 10
	minBodyHeight      = 6
	defaultMaxLogLines = 2000
	minViewportWidth   = 1
	minViewportHeight  = 1
	minMarkerWidth     = 24
)

type viewState struct {
	pm *processManager

	frontend viewport.Model
	backend  viewport.Model

	frontendLines []string
	backendLines  []string
	maxLines      int

	width          int
	height         int
	focus          int
	verticalSplit  bool
	swappedPanes   bool
	fullscreen     bool
	mouseCaptured  bool
	followFrontend bool
	followBackend  bool
	showHelp       bool
	lineMode       bool
	lineModeProc   string
	lineModeSource int
	hoverLine      bool
	hoverProc      string
	hoverSource    int
	selection      dragSelection
	status         string
	procs          map[string]procInfo
}

type procInfo struct {
	running  bool
	pid      int
	lastExit string
}

type dragSelection struct {
	active    bool
	proc      string
	startLine int
	startCol  int
	endLine   int
	endCol    int
	moved     bool
}

var (
	clipboardInitOnce sync.Once
	clipboardInitErr  error
)

func newViewState(pm *processManager) viewState {
	v1 := viewport.New(minPaneWidth, initialPaneHeight)
	v2 := viewport.New(minPaneWidth, initialPaneHeight)
	return viewState{
		pm:             pm,
		frontend:       v1,
		backend:        v2,
		maxLines:       defaultMaxLogLines,
		focus:          0,
		verticalSplit:  true,
		mouseCaptured:  true,
		followFrontend: true,
		followBackend:  true,
		showHelp:       false,
		status:         "starting frontend/backend...",
		procs: map[string]procInfo{
			"frontend": {},
			"backend":  {},
		},
	}
}

func (m viewState) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			err := m.pm.startAll()
			if err != nil {
				return actionDoneMsg{err: err, message: "startup failed: " + err.Error()}
			}
			return actionDoneMsg{message: "dev processes started", markRunning: []string{"frontend", "backend"}}
		},
		waitForEvent(m.pm.events),
	)
}

func waitForEvent(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return actionDoneMsg{message: "event stream closed"}
		}
		return msg
	}
}

func (m viewState) mouseEnableCmd() tea.Cmd {
	if m.lineMode {
		return tea.EnableMouseAllMotion
	}
	return tea.EnableMouseCellMotion
}

func (m viewState) focusedName() string { return m.focusedProc() }

func (m viewState) focusedProc() string {
	first, second := m.slotProcs()
	if m.focus == 0 {
		return first
	}
	return second
}

func (m viewState) slotProcs() (string, string) {
	if m.swappedPanes {
		return "backend", "frontend"
	}
	return "frontend", "backend"
}

func (m *viewState) gotoBottomFocused() {
	if m.focusedProc() == "frontend" {
		m.frontend.GotoBottom()
		return
	}
	m.backend.GotoBottom()
}

func (m *viewState) gotoTopFocused() {
	if m.focusedProc() == "frontend" {
		m.frontend.GotoTop()
		return
	}
	m.backend.GotoTop()
}

func (m *viewState) setFollowFocused(v bool) {
	if m.focusedProc() == "frontend" {
		m.followFrontend = v
		return
	}
	m.followBackend = v
}

func (m *viewState) clearFocusedLogs() {
	switch m.focusedProc() {
	case "frontend":
		m.frontendLines = nil
		m.frontend.SetContent("")
		m.status = "cleared frontend logs"
	default:
		m.backendLines = nil
		m.backend.SetContent("")
		m.status = "cleared backend logs"
	}
	m.resize(m.width, m.height)
}

func (m *viewState) updateViewportForProc(proc string, msg tea.Msg) {
	var cmd tea.Cmd
	switch proc {
	case "frontend":
		m.frontend, cmd = m.frontend.Update(msg)
	case "backend":
		m.backend, cmd = m.backend.Update(msg)
	}
	_ = cmd
}

func (m *viewState) setFollowProc(proc string, v bool) {
	if proc == "frontend" {
		m.followFrontend = v
		return
	}
	m.followBackend = v
}

func (m viewState) focusedAtBottom() bool {
	if m.focusedProc() == "frontend" {
		return m.frontend.AtBottom()
	}
	return m.backend.AtBottom()
}

func didManualScroll(msg tea.Msg) bool {
	switch t := msg.(type) {
	case tea.KeyMsg:
		switch t.String() {
		case "up", "down", "pgup", "pgdown":
			return true
		}
	case tea.MouseMsg:
		return tea.MouseEvent(t).IsWheel()
	}
	return false
}

func shouldResumeFollow(msg tea.Msg) bool {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return false
	}
	switch key.String() {
	case "down", "pgdown":
		return true
	default:
		return false
	}
}

func (m viewState) actionCmd(startMsg, okMsg, failPrefix string, fn func() error, markRunning []string) tea.Cmd {
	m.status = startMsg
	return func() tea.Msg {
		err := fn()
		if err != nil {
			return actionDoneMsg{message: failPrefix + ": " + err.Error(), err: err}
		}
		return actionDoneMsg{message: okMsg, markRunning: markRunning}
	}
}

func (m *viewState) appendLine(proc, line string) {
	switch proc {
	case "frontend":
		m.frontendLines = appendWithCap(m.frontendLines, line, m.maxLines)
		m.frontend.SetContent(wrapLinesForWidth(m.frontendLines, m.frontend.Width))
		if m.followFrontend {
			m.frontend.GotoBottom()
		}
	case "backend":
		m.backendLines = appendWithCap(m.backendLines, line, m.maxLines)
		m.backend.SetContent(wrapLinesForWidth(m.backendLines, m.backend.Width))
		if m.followBackend {
			m.backend.GotoBottom()
		}
	}
}

func (m *viewState) appendSessionMarker(proc, message, color string) {
	ts := time.Now().Format("15:04:05")
	m.appendLine(proc, markerToken(ts, color, message))
}

func (m *viewState) refreshViewportContent() {
	m.frontend.SetContent(wrapLinesForWidth(m.frontendLines, m.frontend.Width))
	m.backend.SetContent(wrapLinesForWidth(m.backendLines, m.backend.Width))
}
