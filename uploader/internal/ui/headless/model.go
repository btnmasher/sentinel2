package headless

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"sentinel2-uploader/internal/client"
	"sentinel2-uploader/internal/config"
	"sentinel2-uploader/internal/logging"
	"sentinel2-uploader/internal/runtime"
)

const headlessLogLimit = 200_000

var (
	panelStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	focusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	buttonStyle        = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.NormalBorder())
	buttonFocusedStyle = buttonStyle.Copy().BorderForeground(lipgloss.Color("10")).Foreground(lipgloss.Color("10"))
	segmentBaseStyle   = lipgloss.NewStyle().Padding(0, 1)
	segmentOnStyle     = segmentBaseStyle.Copy().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("10"))
	segmentOffStyle    = segmentBaseStyle.Copy().Foreground(lipgloss.Color("245")).Background(lipgloss.Color("236"))

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
type channelsUpdatedMsg []string

const (
	statusIdle statusKind = iota
	statusConnecting
	statusConnected
	statusStopping
	statusError
)

type headlessModel struct {
	buildVersion string

	runner      *runtime.Controller
	logger      *logging.Logger
	unsubscribe func()

	logCh  chan string
	cfgCh  chan []string
	doneCh <-chan error

	inputs []textinput.Model
	focus  int

	running    bool
	connecting bool
	status     string
	errText    string
	kind       statusKind

	showLogs bool
	autoConn bool
	logText  string
	logView  viewport.Model

	width     int
	height    int
	animPhase int
	imgay     bool

	confirmQuit       bool
	confirmQuitChoice int
	channels          []string
}

func Run(buildVersion string, opts config.Options) {
	if saved, loadErr := loadPersistedOptions(); loadErr == nil {
		opts = mergeHeadlessOptions(opts, saved)
	}

	logger := logging.New(false)
	logger.SetDebugEnabled(false)
	logger.Info("starting uploader TUI", logging.Field("version", buildVersion))

	m := newHeadlessModel(buildVersion, opts, logger)
	program := tea.NewProgram(m, tea.WithAltScreen())
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

	m := &headlessModel{
		buildVersion: buildVersion,
		runner:       runtime.NewController(),
		logger:       logger,
		logCh:        make(chan string, 512),
		cfgCh:        make(chan []string, 8),
		inputs:       inputs,
		status:       "Idle",
		kind:         statusIdle,
		logView:      logView,
		imgay:        opts.ImGay,
		autoConn:     opts.AutoConnect,
	}

	m.unsubscribe = logger.Subscribe(func(event logging.Event) {
		line := logging.FormatEventLine(event)
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
		cmds = append(cmds, m.startUploaderCmd())
	}
	return tea.Batch(cmds...)
}

func (m *headlessModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeLogs()
		return m, nil
	case logMsg:
		line := string(msg)
		m.logText = logging.AppendWithLimit(m.logText, line, headlessLogLimit)
		m.logView.SetContent(m.logText)
		m.logView.GotoBottom()
		return m, waitForLog(m.logCh)
	case channelsUpdatedMsg:
		m.channels = append([]string(nil), msg...)
		return m, waitForChannels(m.cfgCh)
	case runDoneMsg:
		m.running = false
		m.connecting = false
		m.doneCh = nil
		if msg.err != nil {
			m.status = "Disconnected (error)"
			m.kind = statusError
			m.errText = msg.err.Error()
		} else {
			m.status = "Idle"
			m.kind = statusIdle
			m.errText = ""
		}
		return m, nil
	case startResultMsg:
		m.connecting = false
		if msg.err != nil {
			m.status = "Disconnected (error)"
			m.kind = statusError
			m.errText = msg.err.Error()
			return m, nil
		}
		m.running = true
		m.doneCh = msg.done
		m.status = "Connected"
		m.kind = statusConnected
		m.errText = ""
		return m, waitForRunDone(msg.done)
	case tickMsg:
		m.animPhase++
		if m.animPhase > 1_000_000_000 {
			m.animPhase = 0
		}
		return m, tickCmd()
	case tea.KeyMsg:
		if m.confirmQuit {
			return m.handleConfirmQuitKeys(msg.String())
		}
		switch msg.String() {
		case "ctrl+c":
			return m, m.requestQuitCmd()
		case "tab", "down":
			m.focus = (m.focus + 1) % m.focusCount()
			m.applyFocus()
			return m, nil
		case "shift+tab", "up":
			m.focus = (m.focus + m.focusCount() - 1) % m.focusCount()
			m.applyFocus()
			return m, nil
		case " ":
			if m.focus >= len(m.inputs) {
				return m, m.activateFocusedControl()
			}
		case "enter":
			if m.focus >= len(m.inputs) {
				return m, m.activateFocusedControl()
			}
		}
	}

	if m.focus < len(m.inputs) {
		updated, cmd := m.inputs[m.focus].Update(msg)
		m.inputs[m.focus] = updated
		return m, cmd
	}
	return m, nil
}

func (m *headlessModel) applyFocus() {
	for i := range m.inputs {
		if i == m.focus {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
}

func (m *headlessModel) focusCount() int {
	return len(m.inputs) + 4
}

func (m *headlessModel) debugIndex() int {
	return len(m.inputs)
}

func (m *headlessModel) autoConnectIndex() int {
	return len(m.inputs) + 1
}

func (m *headlessModel) connectIndex() int {
	return len(m.inputs) + 2
}

func (m *headlessModel) quitIndex() int {
	return len(m.inputs) + 3
}

func (m *headlessModel) activateFocusedControl() tea.Cmd {
	switch m.focus {
	case m.debugIndex():
		m.showLogs = !m.showLogs
		m.logger.SetDebugEnabled(m.showLogs)
		return nil
	case m.autoConnectIndex():
		m.autoConn = !m.autoConn
		_ = savePersistedOptions(m.currentOptions())
		return nil
	case m.connectIndex():
		if m.connecting {
			return nil
		}
		if !m.canConnect() {
			m.errText = "base URL and uploader token are required"
			return nil
		}
		if m.running {
			m.runner.Stop()
			m.status = "Stopping..."
			m.kind = statusStopping
			m.errText = ""
			return nil
		}
		return m.startUploaderCmd()
	case m.quitIndex():
		return m.requestQuitCmd()
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

func (m *headlessModel) handleConfirmQuitKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.confirmQuit = false
		return m, nil
	case "tab", "right", "down", "left", "up":
		m.confirmQuitChoice = (m.confirmQuitChoice + 1) % 2
		return m, nil
	case " ", "enter":
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
		Debug:       m.showLogs,
	}
}

func (m *headlessModel) canConnect() bool {
	return strings.TrimSpace(m.inputs[0].Value()) != "" && strings.TrimSpace(m.inputs[1].Value()) != ""
}

func (m *headlessModel) startUploaderCmd() tea.Cmd {
	opts := m.currentOptions()
	if strings.TrimSpace(opts.LogDir) == "" {
		m.errText = "log directory is required"
		return nil
	}
	info, statErr := os.Stat(opts.LogDir)
	if statErr != nil || !info.IsDir() {
		if statErr != nil {
			m.errText = "log directory is not accessible: " + statErr.Error()
		} else {
			m.errText = "log directory is not a directory"
		}
		return nil
	}
	if err := config.ValidateRequired(opts); err != nil {
		m.errText = err.Error()
		return nil
	}
	if err := savePersistedOptions(opts); err != nil {
		m.errText = err.Error()
		return nil
	}
	m.connecting = true
	m.status = "Connecting..."
	m.kind = statusConnecting
	m.errText = ""
	return func() tea.Msg {
		done, err := m.runner.Start(opts, m.logger, runtime.StartHooks{
			OnChannelsUpdate: func(channels []client.ChannelConfig) {
				names := make([]string, 0, len(channels))
				for _, channel := range channels {
					name := strings.TrimSpace(channel.Name)
					if name != "" {
						names = append(names, name)
					}
				}
				select {
				case m.cfgCh <- names:
				default:
					select {
					case <-m.cfgCh:
					default:
					}
					m.cfgCh <- names
				}
			},
		})
		return startResultMsg{done: done, err: err}
	}
}

func (m *headlessModel) cleanup() {
	if m.unsubscribe != nil {
		m.unsubscribe()
	}
	m.runner.Stop()
}

func (m *headlessModel) logPanelHeight() int {
	h := m.height - 18
	if h < 8 {
		return 8
	}
	return h
}

func (m *headlessModel) resizeLogs() {
	w := m.contentWidth() - 4
	if w < 20 {
		w = 20
	}
	h := m.logPanelHeight() - 3
	if h < 3 {
		h = 3
	}
	m.logView.Width = w
	m.logView.Height = h
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

func waitForChannels(ch <-chan []string) tea.Cmd {
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
