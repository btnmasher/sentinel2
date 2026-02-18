package devconsole

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m viewState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{waitForEvent(m.pm.events)}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		if m.followFrontend {
			m.frontend.GotoBottom()
		}
		if m.followBackend {
			m.backend.GotoBottom()
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		handled := false
		if m.lineMode {
			switch msg.String() {
			case "esc":
				m.lineMode = false
				m.status = "line selection mode off"
				m.resize(m.width, m.height)
				if m.mouseCaptured && !m.fullscreen {
					cmds = append(cmds, m.mouseEnableCmd())
				}
				handled = true
			case "up":
				m.moveLineMode(-1)
				handled = true
			case "down":
				m.moveLineMode(1)
				handled = true
			case "enter", "y":
				m.copyLineMode()
				handled = true
			}
			if handled {
				return m, tea.Batch(cmds...)
			}
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.pm.stopAll()
			return m, tea.Quit
		case "esc":
			if m.fullscreen {
				m.fullscreen = false
				m.mouseCaptured = true
				m.resize(m.width, m.height)
				cmds = append(cmds, m.mouseEnableCmd())
				handled = true
			}
		case "tab":
			m.focus = (m.focus + 1) % 2
			handled = true
		case "left":
			m.focus = 0
			handled = true
		case "right":
			m.focus = 1
			handled = true
		case "1":
			m.focus = 0
			handled = true
		case "2":
			m.focus = 1
			handled = true
		case "s":
			m.verticalSplit = !m.verticalSplit
			m.resize(m.width, m.height)
			handled = true
		case "x":
			m.swappedPanes = !m.swappedPanes
			handled = true
		case "z":
			m.fullscreen = !m.fullscreen
			m.resize(m.width, m.height)
			if m.fullscreen {
				m.mouseCaptured = false
				cmds = append(cmds, tea.DisableMouse)
			} else {
				m.mouseCaptured = true
				cmds = append(cmds, m.mouseEnableCmd())
			}
			handled = true
		case "m":
			m.mouseCaptured = !m.mouseCaptured
			if m.mouseCaptured {
				cmds = append(cmds, m.mouseEnableCmd())
				m.status = "mouse capture enabled"
			} else {
				cmds = append(cmds, tea.DisableMouse)
				m.status = "mouse capture disabled"
			}
			m.resize(m.width, m.height)
			handled = true
		case "h":
			m.showHelp = !m.showHelp
			m.resize(m.width, m.height)
			handled = true
		case "v":
			m.toggleLineMode()
			if m.mouseCaptured && !m.fullscreen {
				cmds = append(cmds, m.mouseEnableCmd())
			}
			handled = true
		case "end":
			m.gotoBottomFocused()
			m.setFollowFocused(true)
			handled = true
		case "c":
			m.clearFocusedLogs()
			handled = true
		case "home":
			m.gotoTopFocused()
			m.setFollowFocused(false)
			handled = true
		case "r":
			name := m.focusedName()
			label := "restart " + name
			cmds = append(cmds, m.actionCmd(label+" in progress", label+" succeeded", label+" failed", func() error { return m.pm.restart(name) }, []string{name}))
			m.resize(m.width, m.height)
			handled = true
		case "f":
			cmds = append(cmds, m.actionCmd("restart frontend in progress", "restart frontend succeeded", "restart frontend failed", func() error { return m.pm.restart("frontend") }, []string{"frontend"}))
			m.resize(m.width, m.height)
			handled = true
		case "b":
			cmds = append(cmds, m.actionCmd("restart backend in progress", "restart backend succeeded", "restart backend failed", func() error { return m.pm.restart("backend") }, []string{"backend"}))
			m.resize(m.width, m.height)
			handled = true
		case "ctrl+r":
			name := m.focusedName()
			label := "rebuild " + name + " + restart"
			cmds = append(cmds, m.actionCmd(label+" in progress", label+" succeeded", label+" failed", func() error {
				if name == "frontend" {
					if err := m.pm.rebuildFrontend(); err != nil {
						return err
					}
				} else {
					if err := m.pm.rebuildBackend(); err != nil {
						return err
					}
				}
				return m.pm.restart(name)
			}, []string{name}))
			m.resize(m.width, m.height)
			handled = true
		case "ctrl+b":
			cmds = append(cmds, m.actionCmd("rebuild backend + restart in progress", "rebuild backend + restart succeeded", "rebuild backend + restart failed", func() error {
				if err := m.pm.rebuildBackend(); err != nil {
					return err
				}
				return m.pm.restart("backend")
			}, []string{"backend"}))
			m.resize(m.width, m.height)
			handled = true
		case "ctrl+f":
			cmds = append(cmds, m.actionCmd("rebuild frontend + restart in progress", "rebuild frontend + restart succeeded", "rebuild frontend + restart failed", func() error {
				if err := m.pm.rebuildFrontend(); err != nil {
					return err
				}
				return m.pm.restart("frontend")
			}, []string{"frontend"}))
			m.resize(m.width, m.height)
			handled = true
		case "ctrl+g":
			cmds = append(cmds, m.actionCmd("stop + rebuild + migrate + restart backend in progress", "stop + rebuild + migrate + restart backend succeeded", "stop + rebuild + migrate + restart backend failed", func() error {
				m.pm.stop("backend")
				if err := m.pm.rebuildBackend(); err != nil {
					return err
				}
				if err := m.pm.runMigrate(); err != nil {
					return err
				}
				return m.pm.restart("backend")
			}, []string{"backend"}))
			m.resize(m.width, m.height)
			handled = true
		}
		if handled {
			return m, tea.Batch(cmds...)
		}

	case tea.MouseMsg:
		ev := tea.MouseEvent(msg)
		slot, proc, line, col, ok := m.hitTestMouse(ev.X, ev.Y)
		if ok {
			m.focus = slot
		}
		if m.lineMode {
			if ev.Action == tea.MouseActionMotion && ok {
				if src, found := m.sourceIndexAtWrapped(proc, line); found {
					m.hoverLine = true
					m.hoverProc = proc
					m.hoverSource = src
					return m, tea.Batch(cmds...)
				}
			} else if ev.Action == tea.MouseActionMotion && !ok {
				m.hoverLine = false
				m.hoverProc = ""
			}
		}
		if m.lineMode && ok && ev.Action == tea.MouseActionPress && ev.Button == tea.MouseButtonLeft {
			m.setLineModeAt(proc, line)
			m.copyLineMode()
			return m, tea.Batch(cmds...)
		}
		switch ev.Action {
		case tea.MouseActionPress:
			if ok && ev.Button == tea.MouseButtonLeft {
				m.selection = dragSelection{active: true, proc: proc, startLine: line, startCol: col, endLine: line, endCol: col}
				return m, tea.Batch(cmds...)
			}
		case tea.MouseActionMotion:
			if m.selection.active && ok && m.selection.proc == proc {
				m.selection.endLine = line
				m.selection.endCol = col
				m.selection.moved = (m.selection.startLine != line) || (m.selection.startCol != col)
				return m, tea.Batch(cmds...)
			}
		case tea.MouseActionRelease:
			if m.selection.active {
				if !m.selection.moved {
					m.selection = dragSelection{}
					return m, tea.Batch(cmds...)
				}
				if ok && m.selection.proc == proc {
					m.selection.endLine = line
					m.selection.endCol = col
				}
				text := m.selectedText()
				if strings.TrimSpace(text) == "" {
					m.status = fmt.Sprintf("no text selected in %s", m.selection.proc)
				} else if err := writeClipboard(text); err != nil {
					m.status = "clipboard copy failed: " + err.Error()
				} else {
					m.status = fmt.Sprintf("copied selection from %s", m.selection.proc)
				}
				m.resize(m.width, m.height)
				m.selection = dragSelection{}
				return m, tea.Batch(cmds...)
			}
		}
		if ev.IsWheel() && ok {
			m.updateViewportForProc(proc, msg)
			m.setFollowProc(proc, false)
			return m, tea.Batch(cmds...)
		}

	case lineMsg:
		m.appendLine(msg.proc, msg.line)
		return m, tea.Batch(cmds...)

	case lineBatchMsg:
		m.appendLines(msg.proc, msg.lines)
		return m, tea.Batch(cmds...)

	case procStartedMsg:
		p := m.procs[msg.proc]
		p.running = true
		p.pid = msg.pid
		p.lastExit = ""
		m.procs[msg.proc] = p
		m.appendSessionMarker(msg.proc, fmt.Sprintf("%s started (pid=%d)", msg.proc, msg.pid), "10")
		return m, tea.Batch(cmds...)

	case procExitMsg:
		p := m.procs[msg.proc]
		// Ignore stale exits from an older process instance after a restart.
		if p.running && p.pid > 0 && msg.pid > 0 && p.pid != msg.pid {
			return m, tea.Batch(cmds...)
		}
		p.running = false
		p.pid = 0
		p.lastExit = exitSummary(msg.err, msg.code)
		m.procs[msg.proc] = p
		markerColor := "9"
		if msg.code == 0 && msg.err == nil {
			markerColor = "10"
		}
		m.appendSessionMarker(msg.proc, fmt.Sprintf("%s stopped (%s)", msg.proc, p.lastExit), markerColor)
		m.status = fmt.Sprintf("%s %s", msg.proc, p.lastExit)
		m.resize(m.width, m.height)
		return m, tea.Batch(cmds...)

	case actionDoneMsg:
		if msg.message != "" {
			m.status = msg.message
			if msg.err == nil {
				for _, proc := range msg.markRunning {
					p := m.procs[proc]
					p.running = true
					m.procs[proc] = p
				}
			}
			m.resize(m.width, m.height)
		} else if msg.err != nil {
			m.status = "error: " + msg.err.Error()
			m.resize(m.width, m.height)
		}
		return m, tea.Batch(cmds...)
	}

	var cmd tea.Cmd
	switch m.focusedProc() {
	case "frontend":
		m.frontend, cmd = m.frontend.Update(msg)
	default:
		m.backend, cmd = m.backend.Update(msg)
	}
	if didManualScroll(msg) {
		if shouldResumeFollow(msg) && m.focusedAtBottom() {
			m.setFollowFocused(true)
		} else {
			m.setFollowFocused(false)
		}
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *viewState) copyLineMode() {
	if !m.lineMode {
		return
	}
	raw := m.rawLinesForProc(m.lineModeProc)
	if len(raw) == 0 || m.lineModeSource < 0 || m.lineModeSource >= len(raw) {
		m.status = "no line selected in " + m.lineModeProc
		m.resize(m.width, m.height)
		return
	}
	line := plainLineForCopy(raw[m.lineModeSource])
	if strings.TrimSpace(line) == "" {
		m.status = "selected line is empty in " + m.lineModeProc
		m.resize(m.width, m.height)
		return
	}
	if err := writeClipboard(line); err != nil {
		m.status = "clipboard copy failed: " + err.Error()
	} else {
		m.status = fmt.Sprintf("copied line %d from %s", m.lineModeSource+1, m.lineModeProc)
	}
	m.resize(m.width, m.height)
}
