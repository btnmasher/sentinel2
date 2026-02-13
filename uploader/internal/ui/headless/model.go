package headless

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"sentinel2-uploader/internal/client"
	"sentinel2-uploader/internal/config"
	"sentinel2-uploader/internal/evelogs"
	"sentinel2-uploader/internal/logging"
	"sentinel2-uploader/internal/runtime"
)

const headlessLogLimit = 200_000
const (
	minLogPanelHeight      = 8
	nonLogLayoutReserveMin = 24
)

var (
	panelStyle           = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	titleStyle           = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	focusStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errorStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	helpStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	disabledButtonBorder = lipgloss.Border{
		Top:         "╌",
		Bottom:      "╌",
		Left:        "┊",
		Right:       "┊",
		TopLeft:     "┌",
		TopRight:    "┐",
		BottomLeft:  "└",
		BottomRight: "┘",
	}
	disabledBorderColor = lipgloss.Color("240")
	disabledTextColor   = lipgloss.Color("240")

	buttonStyle                = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.NormalBorder())
	buttonFocusedStyle         = buttonStyle.Copy().BorderForeground(lipgloss.Color("10")).Foreground(lipgloss.Color("10"))
	buttonDisabledBaseStyle    = buttonStyle.Copy().Border(disabledButtonBorder).BorderForeground(disabledBorderColor)
	buttonDisabledStyle        = buttonDisabledBaseStyle.Copy().Foreground(disabledTextColor)
	buttonDisabledFocusedStyle = buttonStyle.Copy().BorderForeground(lipgloss.Color("255")).Foreground(lipgloss.Color("250"))
	segmentBaseStyle           = lipgloss.NewStyle().Padding(0, 1)
	segmentOnStyle             = segmentBaseStyle.Copy().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("10"))
	segmentOffStyle            = segmentBaseStyle.Copy().Foreground(lipgloss.Color("245")).Background(lipgloss.Color("236"))

	rainbowPalette = []rgb{
		{255, 0, 0},
		{255, 127, 0},
		{255, 255, 0},
		{0, 255, 0},
		{0, 180, 255},
		{75, 0, 130},
		{148, 0, 211},
	}
)

type rgb struct {
	r uint8
	g uint8
	b uint8
}

type logMsg string

type runDoneMsg struct {
	err error
}

type tickMsg struct{}

type startResultMsg struct {
	done <-chan error
	err  error
}

type statusKind int
type channelsUpdatedMsg []client.ChannelConfig

type tabKind int

const (
	statusIdle statusKind = iota
	statusConnecting
	statusConnected
	statusStopping
	statusError
)

const (
	tabOverview tabKind = iota
	tabSettings
)

const (
	channelStatusWarnAfter   = 10 * time.Minute
	channelStatusStaleAfter  = time.Hour
	channelStatusRefreshRate = 30 * time.Second
)

type channelHealthKind int

const (
	channelMissing channelHealthKind = iota
	channelActive
	channelWarn
	channelStale
)

type channelHealthRow struct {
	Name   string
	Kind   channelHealthKind
	Reason string
}

type headlessModel struct {
	buildVersion string

	runner      *runtime.Controller
	logger      *logging.Logger
	unsubscribe func()

	logCh  chan string
	cfgCh  chan []client.ChannelConfig
	doneCh <-chan error

	inputs []textinput.Model
	focus  int
	tab    tabKind

	running    bool
	connecting bool
	status     string
	kind       statusKind

	showLogs      bool
	autoConn      bool
	settingsDirty bool
	followLogs    bool
	debugOn       bool
	logText       string
	logView       viewport.Model
	leftView      viewport.Model
	rightView     viewport.Model
	settingsView  viewport.Model
	helpView      help.Model
	keys          keyMap

	width     int
	height    int
	animPhase int
	imgay     bool

	confirmQuit       bool
	confirmQuitChoice int
	errorModalText    string
	filePickerOpen    bool
	filePicker        filepicker.Model
	savedSettings     config.UploaderSettings
	draftSettings     config.UploaderSettings
	channels          []client.ChannelConfig
	channelHealth     []channelHealthRow
	healthDetail      string
	lastHealthRefresh time.Time
}

func Run(buildVersion string, opts config.Options) {
	if saved, loadErr := loadPersistedOptions(); loadErr == nil {
		opts = mergeHeadlessOptions(opts, saved)
	}

	logger := logging.New(false)
	logger.SetDebugEnabled(opts.Debug)
	logger.SetTerminalOutputEnabled(false)
	logger.Info("starting uploader TUI", logging.Field("version", buildVersion))

	m := newHeadlessModel(buildVersion, opts, logger)
	program := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	result, runErr := program.Run()
	model, _ := result.(*headlessModel)
	if model != nil {
		model.cleanup()
	}
	if runErr != nil {
		fmt.Fprintln(os.Stderr, runErr)
		os.Exit(1)
	}
}

func newHeadlessModel(buildVersion string, opts config.Options, logger *logging.Logger) *headlessModel {
	inputs := make([]textinput.Model, 3)
	for i := range inputs {
		inputs[i] = textinput.New()
		inputs[i].CharLimit = 2048
		inputs[i].Width = 80
	}
	inputs[0].Placeholder = "https://intel.example.com"
	inputs[0].SetValue(strings.TrimSpace(opts.BaseURL))
	inputs[1].Placeholder = "Uploader token"
	inputs[1].EchoMode = textinput.EchoPassword
	inputs[1].EchoCharacter = '•'
	inputs[1].SetValue(strings.TrimSpace(opts.Token))
	inputs[2].Placeholder = config.DefaultLogDir()
	inputs[2].SetValue(strings.TrimSpace(opts.LogDir))
	inputs[0].Focus()

	logView := viewport.New(80, 20)
	leftView := viewport.New(24, 8)
	rightView := viewport.New(24, 8)
	settingsView := viewport.New(80, 12)
	picker := filepicker.New()
	picker.FileAllowed = false
	picker.DirAllowed = true
	picker.ShowHidden = false
	picker.ShowSize = false
	picker.ShowPermissions = false
	// Keep directory navigation and selection separate:
	// - space/right/l: open directory
	// - enter: select current directory and close picker
	picker.KeyMap.Open = key.NewBinding(key.WithKeys(" ", "right", "l"), key.WithHelp("space", "open"))
	picker.KeyMap.Select = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select"))

	m := &headlessModel{
		buildVersion: buildVersion,
		runner:       runtime.NewController(),
		logger:       logger,
		logCh:        make(chan string, 512),
		cfgCh:        make(chan []client.ChannelConfig, 8),
		inputs:       inputs,
		status:       "Idle",
		kind:         statusIdle,
		logView:      logView,
		leftView:     leftView,
		rightView:    rightView,
		settingsView: settingsView,
		imgay:        opts.ImGay,
		autoConn:     opts.AutoConnect,
		debugOn:      opts.Debug,
		followLogs:   true,
		tab:          tabOverview,
		helpView:     help.New(),
		keys:         newKeyMap(),
		filePicker:   picker,
	}
	m.savedSettings = config.SettingsFromOptions(opts)
	m.draftSettings = m.savedSettings

	m.unsubscribe = logger.Subscribe(func(event logging.Event) {
		line := logging.FormatEventANSI(event)
		select {
		case m.logCh <- line:
		default:
			select {
			case <-m.logCh:
			default:
			}
			m.logCh <- line
		}
	})

	return m
}

func (m *headlessModel) Init() tea.Cmd {
	cmds := []tea.Cmd{waitForLog(m.logCh), waitForChannels(m.cfgCh), tickCmd()}
	if m.autoConn && m.canConnect() {
		cmds = append(cmds, m.startUploaderCmd(true))
	}
	return tea.Batch(cmds...)
}

func (m *headlessModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.filePickerOpen {
		if ws, ok := msg.(tea.WindowSizeMsg); ok {
			m.width = ws.Width
			m.height = ws.Height
			m.resizeLogs()
			m.resizePaneViewports()
			m.resizeFilePicker()
		}
		return m.updateFilePickerMsg(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeLogs()
		m.resizePaneViewports()
		return m, nil
	case logMsg:
		line := string(msg)
		wasAtBottom := m.logView.AtBottom()
		m.logText = logging.AppendWithLimit(m.logText, line, headlessLogLimit)
		m.setLogViewportContent()
		if m.followLogs || wasAtBottom {
			m.logView.GotoBottom()
			m.followLogs = true
		}
		return m, waitForLog(m.logCh)
	case channelsUpdatedMsg:
		m.channels = append([]client.ChannelConfig(nil), msg...)
		m.refreshChannelHealth()
		return m, waitForChannels(m.cfgCh)
	case runDoneMsg:
		m.running = false
		m.connecting = false
		m.doneCh = nil
		if msg.err != nil {
			m.status = "Disconnected (error)"
			m.kind = statusError
			m.errorModalText = msg.err.Error()
		} else {
			m.status = "Idle"
			m.kind = statusIdle
			m.errorModalText = ""
		}
		return m, nil
	case startResultMsg:
		m.connecting = false
		if msg.err != nil {
			m.status = "Disconnected (error)"
			m.kind = statusError
			m.errorModalText = msg.err.Error()
			return m, nil
		}
		m.running = true
		m.doneCh = msg.done
		m.status = "Connected"
		m.kind = statusConnected
		m.errorModalText = ""
		return m, waitForRunDone(msg.done)
	case tickMsg:
		m.animPhase++
		if m.animPhase > 1_000_000_000 {
			m.animPhase = 0
		}
		if time.Since(m.lastHealthRefresh) >= channelStatusRefreshRate {
			m.refreshChannelHealth()
		}
		return m, tickCmd()
	case tea.MouseMsg:
		var cmds []tea.Cmd
		if m.tab == tabOverview {
			var cmd tea.Cmd
			m.rightView, cmd = m.rightView.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			m.leftView, cmd = m.leftView.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if m.showLogs {
			var cmd tea.Cmd
			m.logView, cmd = m.logView.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			m.followLogs = m.logView.AtBottom()
		}
		return m, tea.Batch(cmds...)
	case tea.KeyMsg:
		if m.errorModalText != "" {
			if msg.String() == "esc" || key.Matches(msg, m.keys.activate) {
				m.errorModalText = ""
				return m, nil
			}
			return m, nil
		}
		if m.confirmQuit {
			return m.handleConfirmQuitKeys(msg)
		}
		switch {
		case key.Matches(msg, m.keys.quit):
			return m, m.requestQuitCmd()
		case msg.String() == "ctrl+f" && m.tab == tabOverview && m.showLogs:
			m.followLogs = true
			m.logView.GotoBottom()
			return m, nil
		case msg.String() == "ctrl+s" && m.tab == tabSettings:
			return m, m.saveSettingsDraft()
		case key.Matches(msg, m.keys.prevTab):
			m.tab = tabOverview
			m.focus = 0
			m.applyFocus()
			return m, nil
		case key.Matches(msg, m.keys.nextTab):
			m.tab = tabSettings
			m.focus = 0
			m.applyFocus()
			return m, nil
		case key.Matches(msg, m.keys.nextFocus):
			m.focus = (m.focus + 1) % m.focusCount()
			m.applyFocus()
			return m, nil
		case key.Matches(msg, m.keys.prevFocus):
			m.focus = (m.focus + m.focusCount() - 1) % m.focusCount()
			m.applyFocus()
			return m, nil
		case key.Matches(msg, m.keys.activate):
			if m.tab == tabOverview || m.focus >= len(m.inputs) {
				return m, m.activateFocusedControl()
			}
		}
	}

	if m.tab == tabSettings && m.focus < len(m.inputs) {
		updated, cmd := m.inputs[m.focus].Update(msg)
		m.inputs[m.focus] = updated
		m.syncDraftFromControls()
		m.settingsDirty = m.draftSettings != m.savedSettings
		return m, cmd
	}
	return m, nil
}

func (m *headlessModel) updateFilePickerMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c":
			m.filePickerOpen = false
			return m, m.requestQuitCmd()
		case "esc":
			m.filePickerOpen = false
			return m, nil
		case "left", "backspace":
			parent := filepath.Dir(m.filePicker.CurrentDirectory)
			if parent == "" || parent == m.filePicker.CurrentDirectory {
				return m, nil
			}
			m.filePicker.CurrentDirectory = parent
			return m, m.filePicker.Init()
		case "enter":
			path := strings.TrimSpace(m.filePicker.CurrentDirectory)
			if path == "" {
				path = "."
			}
			if abs, err := filepath.Abs(path); err == nil {
				path = abs
			}
			m.inputs[2].SetValue(path)
			m.filePickerOpen = false
			m.syncDraftFromControls()
			m.settingsDirty = m.draftSettings != m.savedSettings
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.filePicker, cmd = m.filePicker.Update(msg)
	if ok, path := m.filePicker.DidSelectFile(msg); ok {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			path = filepath.Dir(path)
		}
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
		m.inputs[2].SetValue(path)
		m.filePickerOpen = false
		m.syncDraftFromControls()
		m.settingsDirty = m.draftSettings != m.savedSettings
		return m, nil
	}
	return m, cmd
}

func (m *headlessModel) applyFocus() {
	for i := range m.inputs {
		if m.tab == tabSettings && i == m.focus {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
}

func (m *headlessModel) focusCount() int {
	if m.tab == tabOverview {
		if m.showLogs {
			return 4
		}
		return 3
	}
	return len(m.inputs) + 4
}

func (m *headlessModel) logsIndex() int {
	return 1
}

func (m *headlessModel) quitIndex() int {
	return 2
}

func (m *headlessModel) logsDebugIndex() int {
	return 3
}

func (m *headlessModel) autoConnectIndex() int {
	return len(m.inputs) + 1
}

func (m *headlessModel) browseIndex() int {
	return len(m.inputs)
}

func (m *headlessModel) saveIndex() int {
	return len(m.inputs) + 2
}

func (m *headlessModel) cancelIndex() int {
	return len(m.inputs) + 3
}

func (m *headlessModel) debugIndex() int {
	return -1
}

func (m *headlessModel) connectIndex() int {
	return 0
}

func (m *headlessModel) activateFocusedControl() tea.Cmd {
	if m.tab == tabOverview {
		switch m.focus {
		case m.connectIndex():
			if m.connecting {
				return nil
			}
			if !m.canConnect() {
				m.errorModalText = "Base URL and uploader token are required."
				return nil
			}
			if m.running {
				m.runner.Stop()
				m.status = "Stopping..."
				m.kind = statusStopping
				return nil
			}
			return m.startUploaderCmd(false)
		case m.logsIndex():
			m.showLogs = !m.showLogs
			m.resizePaneViewports()
			if m.showLogs {
				m.followLogs = true
				m.logView.GotoBottom()
			}
			if !m.showLogs && m.focus >= m.focusCount() {
				m.focus = m.focusCount() - 1
			}
			return nil
		case m.quitIndex():
			return m.requestQuitCmd()
		case m.logsDebugIndex():
			m.debugOn = !m.debugOn
			m.logger.SetDebugEnabled(m.debugOn)
			return nil
		default:
			return nil
		}
	}

	switch m.focus {
	case m.browseIndex():
		startDir := strings.TrimSpace(m.inputs[2].Value())
		if startDir == "" {
			startDir = config.DefaultLogDir()
		}
		if abs, err := filepath.Abs(startDir); err == nil {
			startDir = abs
		}
		if info, err := os.Stat(startDir); err != nil || !info.IsDir() {
			startDir = "."
			if abs, err := filepath.Abs(startDir); err == nil {
				startDir = abs
			}
		}
		m.filePicker.CurrentDirectory = startDir
		m.filePicker.Path = ""
		m.filePickerOpen = true
		m.resizeFilePicker()
		return m.filePicker.Init()
	case m.autoConnectIndex():
		m.autoConn = !m.autoConn
		m.syncDraftFromControls()
		m.settingsDirty = m.draftSettings != m.savedSettings
		return nil
	case m.saveIndex():
		return m.saveSettingsDraft()
	case m.cancelIndex():
		if !m.settingsDirty {
			return nil
		}
		m.draftSettings = m.savedSettings
		m.applyDraftToControls()
		m.settingsDirty = false
		return nil
	default:
		return nil
	}
}

func (m *headlessModel) requestQuitCmd() tea.Cmd {
	if m.running || m.connecting {
		m.confirmQuit = true
		m.confirmQuitChoice = 0
		return nil
	}
	m.cleanup()
	return tea.Quit
}

func (m *headlessModel) handleConfirmQuitKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "esc":
		m.confirmQuit = false
		return m, nil
	case key.Matches(msg, m.keys.modalToggle):
		m.confirmQuitChoice = (m.confirmQuitChoice + 1) % 2
		return m, nil
	case key.Matches(msg, m.keys.activate):
		if m.confirmQuitChoice == 1 {
			m.cleanup()
			return m, tea.Quit
		}
		m.confirmQuit = false
		return m, nil
	default:
		return m, nil
	}
}

func (m *headlessModel) currentOptions() config.Options {
	return config.Options{
		BaseURL:     strings.TrimSpace(m.inputs[0].Value()),
		Token:       strings.TrimSpace(m.inputs[1].Value()),
		AutoConnect: m.autoConn,
		ImGay:       m.imgay,
		LogFile:     "",
		LogDir:      strings.TrimSpace(m.inputs[2].Value()),
		Debug:       m.debugOn,
	}
}

func (m *headlessModel) canConnect() bool {
	return strings.TrimSpace(m.inputs[0].Value()) != "" && strings.TrimSpace(m.inputs[1].Value()) != ""
}

func (m *headlessModel) startUploaderCmd(auto bool) tea.Cmd {
	opts := m.currentOptions()
	if strings.TrimSpace(opts.LogDir) == "" {
		m.errorModalText = m.startErrorText(auto, "Log directory is required.")
		return nil
	}
	info, statErr := os.Stat(opts.LogDir)
	if statErr != nil || !info.IsDir() {
		if statErr != nil {
			m.errorModalText = m.startErrorText(auto, "Log directory is not accessible: "+statErr.Error())
		} else {
			m.errorModalText = m.startErrorText(auto, "Log directory is not a directory.")
		}
		return nil
	}
	if err := config.ValidateRequired(opts); err != nil {
		m.errorModalText = m.startErrorText(auto, err.Error())
		return nil
	}
	m.connecting = true
	m.status = "Connecting..."
	m.kind = statusConnecting
	m.errorModalText = ""
	return func() tea.Msg {
		done, err := m.runner.Start(opts, m.logger, runtime.StartHooks{
			OnChannelsUpdate: func(channels []client.ChannelConfig) {
				normalized := make([]client.ChannelConfig, 0, len(channels))
				for _, channel := range channels {
					name := strings.TrimSpace(channel.Name)
					id := strings.TrimSpace(channel.ID)
					if name == "" || id == "" {
						continue
					}
					normalized = append(normalized, client.ChannelConfig{ID: id, Name: name})
				}
				select {
				case m.cfgCh <- normalized:
				default:
					select {
					case <-m.cfgCh:
					default:
					}
					m.cfgCh <- normalized
				}
			},
		})
		return startResultMsg{done: done, err: err}
	}
}

func (m *headlessModel) refreshChannelHealth() {
	m.lastHealthRefresh = time.Now()
	rows := make([]channelHealthRow, 0, len(m.channels))
	m.healthDetail = ""

	logDir := strings.TrimSpace(m.inputs[2].Value())
	if logDir == "" {
		m.channelHealth = rows
		m.healthDetail = "Log directory is not configured."
		return
	}

	info, statErr := os.Stat(logDir)
	if statErr != nil {
		m.channelHealth = rows
		m.healthDetail = "Log directory is not accessible: " + statErr.Error()
		return
	}
	if !info.IsDir() {
		m.channelHealth = rows
		m.healthDetail = "Log path is not a directory."
		return
	}

	latestByChannel := map[string]time.Time{}
	latestFileByChannel := map[string]string{}
	logs, findErr := evelogs.FindLogs(logDir, m.channels)
	if findErr != nil {
		m.channelHealth = rows
		m.healthDetail = "Failed to scan logs: " + findErr.Error()
		return
	}
	for _, selection := range logs {
		stat, err := os.Stat(selection.Path)
		if err != nil {
			continue
		}
		id := strings.TrimSpace(selection.Channel.ID)
		if id == "" {
			continue
		}
		current, ok := latestByChannel[id]
		if !ok || stat.ModTime().After(current) {
			latestByChannel[id] = stat.ModTime()
			latestFileByChannel[id] = filepath.Base(selection.Path)
		}
	}

	now := time.Now()
	for _, channel := range m.channels {
		id := strings.TrimSpace(channel.ID)
		name := strings.TrimSpace(channel.Name)
		row := channelHealthRow{
			Name:   name,
			Kind:   channelMissing,
			Reason: "No matching log file found.",
		}
		if last, ok := latestByChannel[id]; ok {
			age := now.Sub(last)
			fileName := latestFileByChannel[id]
			switch {
			case age <= channelStatusWarnAfter:
				row.Kind = channelActive
				row.Reason = fmt.Sprintf("%s updated %s ago.", fileName, age.Round(time.Second))
			case age <= channelStatusStaleAfter:
				row.Kind = channelWarn
				row.Reason = fmt.Sprintf("%s has no updates for %s.", fileName, age.Round(time.Second))
			default:
				row.Kind = channelStale
				row.Reason = fmt.Sprintf("%s has no updates for %s.", fileName, age.Round(time.Second))
			}
		}
		rows = append(rows, row)
	}
	m.channelHealth = rows
}

func (m *headlessModel) cleanup() {
	if m.unsubscribe != nil {
		m.unsubscribe()
	}
	m.runner.Stop()
}

func (m *headlessModel) logPanelHeight() int {
	available := m.height - nonLogLayoutReserveMin
	if available < minLogPanelHeight {
		return minLogPanelHeight
	}
	return available
}

func (m *headlessModel) resizeLogs() {
	w := m.pageWidth() - 8 // reserve scrollbar + one extra safety column
	if w < 20 {
		w = 20
	}
	h := m.logPanelHeight() - 3
	if h < 3 {
		h = 3
	}
	m.logView.Width = w
	m.logView.Height = h
	m.setLogViewportContent()
}

func (m *headlessModel) fitLogViewportHeight(nonLogSections []string) {
	if m.height <= 0 {
		return
	}
	desired := m.logPanelHeight() - 3
	if desired < 3 {
		desired = 3
	}
	nonLogHeight := lipgloss.Height(strings.Join(nonLogSections, "\n\n"))
	availablePanel := m.height - 2 - nonLogHeight - 2 // outer frame + separator before log panel
	maxLogHeight := availablePanel - 4                // log panel frame + toolbar/newline
	if maxLogHeight < 3 {
		maxLogHeight = 3
	}
	if desired > maxLogHeight {
		desired = maxLogHeight
	}
	m.logView.Height = desired
}

func (m *headlessModel) setLogViewportContent() {
	width := m.logView.Width
	if width < 1 {
		width = 1
	}
	m.logView.SetContent(wrapLogText(m.logText, width))
}

func wrapLogText(text string, width int) string {
	if width <= 0 || text == "" {
		return text
	}
	return ansi.Wrap(text, width, "")
}

func (m *headlessModel) resizePaneViewports() {
	total := m.pageWidth()
	leftWidth, rightWidth, stacked := m.overviewPaneLayout(total)
	if stacked {
		rightWidth = total
	}

	leftInner := leftWidth - 4
	if leftInner < 1 {
		leftInner = 1
	}
	rightInner := rightWidth - 4
	if rightInner < 1 {
		rightInner = 1
	}
	settingsInner := total - 4
	if settingsInner < 1 {
		settingsInner = 1
	}

	paneHeight := 8
	if m.height >= 36 {
		paneHeight = 10
	}

	m.leftView.Width = leftInner
	m.leftView.Height = paneHeight
	m.rightView.Width = rightInner
	m.rightView.Height = paneHeight
	m.settingsView.Width = settingsInner
	m.settingsView.Height = maxInt(12, paneHeight+4)
}

func (m *headlessModel) resizeFilePicker() {
	h := m.height - 14
	if h < 8 {
		h = 8
	}
	m.filePicker.SetHeight(h)
}

func (m *headlessModel) syncDraftFromControls() {
	m.draftSettings.BaseURL = strings.TrimSpace(m.inputs[0].Value())
	m.draftSettings.Token = strings.TrimSpace(m.inputs[1].Value())
	m.draftSettings.LogDir = strings.TrimSpace(m.inputs[2].Value())
	m.draftSettings.AutoConnect = m.autoConn
	m.draftSettings.Debug = m.debugOn
}

func (m *headlessModel) applyDraftToControls() {
	m.inputs[0].SetValue(strings.TrimSpace(m.draftSettings.BaseURL))
	m.inputs[1].SetValue(strings.TrimSpace(m.draftSettings.Token))
	m.inputs[2].SetValue(strings.TrimSpace(m.draftSettings.LogDir))
	m.autoConn = m.draftSettings.AutoConnect
}

func (m *headlessModel) saveSettingsDraft() tea.Cmd {
	if !m.settingsDirty {
		return nil
	}
	m.syncDraftFromControls()
	m.savedSettings = m.draftSettings
	if err := savePersistedOptions(m.currentOptions()); err != nil {
		m.errorModalText = err.Error()
		return nil
	}
	m.settingsDirty = false
	return nil
}

func (m *headlessModel) startErrorText(auto bool, message string) string {
	if !auto {
		return message
	}
	return "Couldn't auto-connect due to: " + message
}

func waitForLog(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return nil
		}
		return logMsg(line)
	}
}

func waitForChannels(ch <-chan []client.ChannelConfig) tea.Cmd {
	return func() tea.Msg {
		channels, ok := <-ch
		if !ok {
			return nil
		}
		return channelsUpdatedMsg(channels)
	}
}

func waitForRunDone(ch <-chan error) tea.Cmd {
	return func() tea.Msg {
		err, ok := <-ch
		if !ok {
			return runDoneMsg{}
		}
		return runDoneMsg{err: err}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}
