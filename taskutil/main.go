package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "dev":
		if err := runDev(); err != nil {
			fmt.Fprintf(os.Stderr, "taskutil dev failed: %v\n", err)
			os.Exit(1)
		}
	case "version":
		version, err := deriveBuildVersion()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(version)
	case "clean-root":
		if err := runCleanRoot(); err != nil {
			fmt.Fprintf(os.Stderr, "taskutil clean-root failed: %v\n", err)
			os.Exit(1)
		}
	case "prepare-embed":
		if err := runPrepareEmbed(); err != nil {
			fmt.Fprintf(os.Stderr, "taskutil prepare-embed failed: %v\n", err)
			os.Exit(1)
		}
	case "dev-logs-tail":
		if err := runDevLogsTail(); err != nil {
			fmt.Fprintf(os.Stderr, "taskutil dev-logs-tail failed: %v\n", err)
			os.Exit(1)
		}
	case "dev-logs-clean":
		if err := runDevLogsClean(); err != nil {
			fmt.Fprintf(os.Stderr, "taskutil dev-logs-clean failed: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: taskutil <command>")
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  dev             run frontend+backend development processes")
	fmt.Fprintln(os.Stderr, "  version         derive build version from git state")
	fmt.Fprintln(os.Stderr, "  clean-root      clean root build artifacts/caches")
	fmt.Fprintln(os.Stderr, "  prepare-embed   copy frontend/dist into backend embed dir")
	fmt.Fprintln(os.Stderr, "  dev-logs-tail   tail dev logs for latest session")
	fmt.Fprintln(os.Stderr, "  dev-logs-clean  clean old dev log sessions")
}

type lineMsg struct {
	proc string
	line string
}

type procExitMsg struct {
	proc string
	err  error
	code int
}

type procStartedMsg struct {
	proc string
	pid  int
}

type actionDoneMsg struct {
	message     string
	err         error
	markRunning []string
}

type processSpec struct {
	name    string
	dir     string
	command string
	args    []string
	env     []string
	logPath string
}

type managedProcess struct {
	spec processSpec

	mu      sync.Mutex
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	logFile *os.File
}

type processManager struct {
	rootDir       string
	ctx           context.Context
	events        chan tea.Msg
	processes     map[string]*managedProcess
	runMigrations bool
}

func newProcessManager(rootDir string, runMigrations bool, events chan tea.Msg) *processManager {
	baseLogDir := envOrDefault("LOG_DIR", "/tmp/sentinel2-dev")
	if runtime.GOOS == "windows" && baseLogDir == "/tmp/sentinel2-dev" {
		baseLogDir = filepath.Join(os.TempDir(), "sentinel2-dev")
	}
	sessionTS := time.Now().Format("20060102-150405")
	logDir := filepath.Join(baseLogDir, sessionTS)
	_ = os.MkdirAll(logDir, 0o755)
	_ = os.MkdirAll(baseLogDir, 0o755)
	_ = os.WriteFile(filepath.Join(baseLogDir, "latest"), []byte(logDir), 0o644)

	viteHost := envOrDefault("VITE_HOST", "127.0.0.1")
	vitePort := envOrDefault("VITE_PORT", "5173")
	devProxy := envOrDefault("DEV_PROXY", "127.0.0.1:5173")
	jsonLogPath := envOrDefault("LOG_JSON_PATH", filepath.Join(logDir, "backend.jsonl"))

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	return &processManager{
		rootDir:       rootDir,
		ctx:           ctx,
		events:        events,
		runMigrations: runMigrations,
		processes: map[string]*managedProcess{
			"frontend": {
				spec: processSpec{
					name:    "frontend",
					dir:     filepath.Join(rootDir, "frontend"),
					command: "bun",
					args:    []string{"run", "dev", "--host", viteHost, "--port", vitePort},
					env: []string{
						"BUN_TMPDIR=" + filepath.Join(rootDir, ".tmp", "bun"),
						"BUN_INSTALL=" + filepath.Join(rootDir, ".tmp", "bun-install"),
					},
					logPath: filepath.Join(logDir, "vite.log"),
				},
			},
			"backend": {
				spec: processSpec{
					name:    "backend",
					dir:     filepath.Join(rootDir, "bin"),
					command: backendBinaryPath(rootDir),
					args:    []string{"serve", "--dev"},
					env: []string{
						"DEV_PROXY=" + devProxy,
						"LOG_PRETTY_PB=1",
						"LOG_JSON=1",
						"LOG_JSON_PB=1",
						"LOG_JSON_PATH=" + jsonLogPath,
					},
					logPath: filepath.Join(logDir, "backend.log"),
				},
			},
		},
	}
}

func (pm *processManager) startAll() error {
	if pm.runMigrations {
		if err := pm.runMigrate(); err != nil {
			return err
		}
	}
	if err := pm.start("frontend"); err != nil {
		return err
	}
	if err := pm.start("backend"); err != nil {
		pm.stop("frontend")
		return err
	}
	return nil
}

func (pm *processManager) start(name string) error {
	proc := pm.processes[name]
	if proc == nil {
		return fmt.Errorf("unknown process %q", name)
	}

	proc.mu.Lock()
	defer proc.mu.Unlock()
	if proc.cmd != nil {
		return nil
	}

	logFile, err := os.OpenFile(proc.spec.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(pm.ctx)
	cmd := exec.CommandContext(ctx, proc.spec.command, proc.spec.args...)
	cmd.Dir = proc.spec.dir
	cmd.Env = append(os.Environ(), proc.spec.env...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		_ = logFile.Close()
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		_ = logFile.Close()
		return err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		_ = logFile.Close()
		return err
	}

	proc.cmd = cmd
	proc.cancel = cancel
	proc.logFile = logFile

	go pm.stream(name, stdout, logFile)
	go pm.stream(name, stderr, logFile)
	go pm.wait(name, cmd)
	pm.events <- procStartedMsg{proc: name, pid: cmd.Process.Pid}
	return nil
}

func (pm *processManager) stream(name string, reader io.Reader, logFile *os.File) {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 1024*64)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(logFile, line)
		pm.events <- lineMsg{proc: name, line: line}
	}
}

func (pm *processManager) wait(name string, cmd *exec.Cmd) {
	err := cmd.Wait()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
		} else {
			code = -1
		}
	}
	pm.events <- procExitMsg{proc: name, err: err, code: code}
}

func (pm *processManager) stop(name string) {
	proc := pm.processes[name]
	if proc == nil {
		return
	}

	proc.mu.Lock()
	cmd := proc.cmd
	cancel := proc.cancel
	logFile := proc.logFile
	proc.cmd = nil
	proc.cancel = nil
	proc.logFile = nil
	proc.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(os.Interrupt)
		time.Sleep(250 * time.Millisecond)
		_ = cmd.Process.Kill()
	}
	if logFile != nil {
		_ = logFile.Close()
	}
}

func (pm *processManager) restart(name string) error {
	pm.stop(name)
	return pm.start(name)
}

func (pm *processManager) stopAll() {
	pm.stop("frontend")
	pm.stop("backend")
}

func (pm *processManager) rebuildBackend() error {
	cmd := exec.Command("task", "build:backend:skip-embed")
	cmd.Dir = pm.rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (pm *processManager) rebuildFrontend() error {
	cmd := exec.Command("task", "build:frontend:dev")
	cmd.Dir = pm.rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (pm *processManager) runMigrate() error {
	cmd := exec.Command(backendBinaryPath(pm.rootDir), "migrate")
	cmd.Dir = filepath.Join(pm.rootDir, "bin")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type model struct {
	pm *processManager

	frontend viewport.Model
	backend  viewport.Model

	frontendLines []string
	backendLines  []string
	maxLines      int

	width  int
	height int
	focus  int
	status string
	procs  map[string]procInfo
}

type procInfo struct {
	running  bool
	pid      int
	lastExit string
}

func newModel(pm *processManager) model {
	v1 := viewport.New(20, 10)
	v2 := viewport.New(20, 10)
	return model{
		pm:       pm,
		frontend: v1,
		backend:  v2,
		maxLines: 2000,
		focus:    0,
		status:   "starting frontend/backend...",
		procs: map[string]procInfo{
			"frontend": {},
			"backend":  {},
		},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			err := m.pm.startAll()
			if err != nil {
				return actionDoneMsg{err: err}
			}
			return actionDoneMsg{
				message:     "dev processes started",
				markRunning: []string{"frontend", "backend"},
			}
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{waitForEvent(m.pm.events)}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		bodyHeight := msg.Height - 4
		if bodyHeight < 6 {
			bodyHeight = 6
		}
		paneWidth := (msg.Width - 3) / 2
		if paneWidth < 20 {
			paneWidth = 20
		}
		m.frontend.Width = paneWidth
		m.frontend.Height = bodyHeight
		m.backend.Width = paneWidth
		m.backend.Height = bodyHeight
		m.frontend.GotoBottom()
		m.backend.GotoBottom()
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.pm.stopAll()
			return m, tea.Quit
		case "tab":
			m.focus = (m.focus + 1) % 2
		case "1":
			m.focus = 0
		case "2":
			m.focus = 1
		case "r":
			name := m.focusedName()
			cmds = append(cmds, m.actionCmd("restart "+name, func() error { return m.pm.restart(name) }, []string{name}))
		case "f":
			cmds = append(cmds, m.actionCmd("restart frontend", func() error { return m.pm.restart("frontend") }, []string{"frontend"}))
		case "b":
			cmds = append(cmds, m.actionCmd("restart backend", func() error { return m.pm.restart("backend") }, []string{"backend"}))
		case "R":
			cmds = append(cmds, m.actionCmd("rebuild backend + restart", func() error {
				if err := m.pm.rebuildBackend(); err != nil {
					return err
				}
				return m.pm.restart("backend")
			}, []string{"backend"}))
		case "F":
			cmds = append(cmds, m.actionCmd("rebuild frontend + restart", func() error {
				if err := m.pm.rebuildFrontend(); err != nil {
					return err
				}
				return m.pm.restart("frontend")
			}, []string{"frontend"}))
		case "m":
			cmds = append(cmds, m.actionCmd("migrate + restart backend", func() error {
				if err := m.pm.runMigrate(); err != nil {
					return err
				}
				return m.pm.restart("backend")
			}, []string{"backend"}))
		}
		return m, tea.Batch(cmds...)

	case lineMsg:
		m.appendLine(msg.proc, msg.line)
		return m, tea.Batch(cmds...)

	case procStartedMsg:
		p := m.procs[msg.proc]
		p.running = true
		p.pid = msg.pid
		p.lastExit = ""
		m.procs[msg.proc] = p
		return m, tea.Batch(cmds...)

	case procExitMsg:
		p := m.procs[msg.proc]
		p.running = false
		p.pid = 0
		p.lastExit = exitSummary(msg.err, msg.code)
		m.procs[msg.proc] = p
		m.status = fmt.Sprintf("%s %s", msg.proc, p.lastExit)
		return m, tea.Batch(cmds...)

	case actionDoneMsg:
		if msg.err != nil {
			m.status = "error: " + msg.err.Error()
		} else if msg.message != "" {
			m.status = msg.message
			for _, proc := range msg.markRunning {
				p := m.procs[proc]
				p.running = true
				m.procs[proc] = p
			}
		}
		return m, tea.Batch(cmds...)
	}

	var cmd tea.Cmd
	if m.focus == 0 {
		m.frontend, cmd = m.frontend.Update(msg)
	} else {
		m.backend, cmd = m.backend.Update(msg)
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("Sentinel2 Dev Supervisor")
	help := "keys: q quit | tab switch pane | 1/2 focus | r restart focused | f/b restart frontend/backend | F rebuild frontend+restart | R rebuild backend+restart | m migrate+restart backend"
	status := "status: " + m.status
	if m.status == "" {
		status = "status: running"
	}

	pane := func(label string, info procInfo, focused bool, content string, width int) string {
		border := lipgloss.NormalBorder()
		style := lipgloss.NewStyle().Border(border).Padding(0, 1).Width(width)
		if focused {
			style = style.BorderForeground(lipgloss.Color("86"))
		}
		header := fmt.Sprintf("%s  [%s]", label, procStateText(info))
		if info.pid > 0 {
			header += fmt.Sprintf(" pid=%d", info.pid)
		}
		if info.lastExit != "" {
			header += " " + info.lastExit
		}
		return style.Render(header + "\n" + content)
	}

	paneWidth := (m.width - 3) / 2
	if paneWidth < 20 {
		paneWidth = 20
	}
	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		pane("frontend", m.procs["frontend"], m.focus == 0, m.frontend.View(), paneWidth),
		" ",
		pane("backend", m.procs["backend"], m.focus == 1, m.backend.View(), paneWidth),
	)

	return strings.Join([]string{title, help, status, row}, "\n")
}

func (m model) actionCmd(label string, fn func() error, markRunning []string) tea.Cmd {
	m.status = label + "..."
	return func() tea.Msg {
		err := fn()
		return actionDoneMsg{message: label + " done", err: err, markRunning: markRunning}
	}
}

func (m *model) appendLine(proc, line string) {
	switch proc {
	case "frontend":
		m.frontendLines = appendWithCap(m.frontendLines, line, m.maxLines)
		m.frontend.SetContent(strings.Join(m.frontendLines, "\n"))
		m.frontend.GotoBottom()
	case "backend":
		m.backendLines = appendWithCap(m.backendLines, line, m.maxLines)
		m.backend.SetContent(strings.Join(m.backendLines, "\n"))
		m.backend.GotoBottom()
	}
}

func (m model) focusedName() string {
	if m.focus == 1 {
		return "backend"
	}
	return "frontend"
}

func appendWithCap(lines []string, line string, max int) []string {
	lines = append(lines, line)
	if max <= 0 || len(lines) <= max {
		return lines
	}
	excess := len(lines) - max
	return lines[excess:]
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

func runDev() error {
	rootDir, err := resolveRootDir()
	if err != nil {
		return err
	}
	runMigrations := strings.EqualFold(strings.TrimSpace(os.Getenv("DEV_MIGRATIONS")), "1")
	events := make(chan tea.Msg, 512)
	pm := newProcessManager(rootDir, runMigrations, events)

	p := tea.NewProgram(newModel(pm), tea.WithAltScreen())
	_, err = p.Run()
	pm.stopAll()
	return err
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func backendBinaryPath(rootDir string) string {
	name := "sentinel2-server"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(rootDir, "bin", name)
}

func runCleanRoot() error {
	rootDir, err := resolveRootDir()
	if err != nil {
		return err
	}
	removePaths := []string{
		filepath.Join(rootDir, "dist"),
		filepath.Join(rootDir, "frontend", "dist"),
		filepath.Join(rootDir, "backend", "internal", "web", "dist"),
		filepath.Join(rootDir, ".tmp", "go-build-cache"),
		filepath.Join(rootDir, ".tmp", "golangci-lint-cache"),
		filepath.Join(rootDir, ".tmp", "bun"),
		filepath.Join(rootDir, ".tmp", "bun-install"),
	}
	for _, path := range removePaths {
		if remErr := os.RemoveAll(path); remErr != nil {
			return remErr
		}
	}
	binDir := filepath.Join(rootDir, "bin")
	entries, readErr := os.ReadDir(binDir)
	if readErr != nil {
		if errors.Is(readErr, os.ErrNotExist) {
			return nil
		}
		return readErr
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == "pb_data" || name == ".env" {
			continue
		}
		if remErr := os.RemoveAll(filepath.Join(binDir, name)); remErr != nil {
			return remErr
		}
	}
	return nil
}

func runPrepareEmbed() error {
	rootDir, err := resolveRootDir()
	if err != nil {
		return err
	}
	srcDir := filepath.Join(rootDir, "frontend", "dist")
	destDir := filepath.Join(rootDir, "backend", "internal", "web", "dist")
	info, statErr := os.Stat(srcDir)
	if statErr != nil || !info.IsDir() {
		return fmt.Errorf("frontend/dist is missing. run 'task build:frontend' first")
	}
	if remErr := os.RemoveAll(destDir); remErr != nil {
		return remErr
	}
	if mkErr := os.MkdirAll(destDir, 0o755); mkErr != nil {
		return mkErr
	}
	return copyDirContents(srcDir, destDir)
}

func copyDirContents(srcDir, destDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == srcDir {
			return nil
		}
		rel, relErr := filepath.Rel(srcDir, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if mkErr := os.MkdirAll(filepath.Dir(target), 0o755); mkErr != nil {
			return mkErr
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func runDevLogsTail() error {
	rootDir, err := resolveRootDir()
	if err != nil {
		return err
	}
	logDir, err := resolveDevLogDir(rootDir)
	if err != nil {
		return err
	}
	lines := 200
	if raw := strings.TrimSpace(os.Getenv("TAIL_LINES")); raw != "" {
		if n, parseErr := strconv.Atoi(raw); parseErr == nil && n > 0 {
			lines = n
		}
	}
	files := []struct {
		label string
		path  string
	}{
		{label: "vite", path: filepath.Join(logDir, "vite.log")},
		{label: "backend", path: filepath.Join(logDir, "backend.log")},
	}
	offsets := map[string]int64{}
	for _, f := range files {
		content, readErr := os.ReadFile(f.path)
		if readErr != nil {
			if errors.Is(readErr, os.ErrNotExist) {
				continue
			}
			return readErr
		}
		linesOut := splitLines(string(content))
		if len(linesOut) > lines {
			linesOut = linesOut[len(linesOut)-lines:]
		}
		for _, line := range linesOut {
			fmt.Printf("[%s] %s\n", f.label, line)
		}
		offsets[f.path] = int64(len(content))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()
	remainders := map[string]string{}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for _, f := range files {
				next, rem, tailErr := readNewLines(f.path, offsets[f.path], remainders[f.path], f.label)
				if tailErr != nil {
					if errors.Is(tailErr, os.ErrNotExist) {
						continue
					}
					return tailErr
				}
				offsets[f.path] = next
				remainders[f.path] = rem
			}
		}
	}
}

func readNewLines(path string, offset int64, remainder string, label string) (int64, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return offset, remainder, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return offset, remainder, err
	}
	size := info.Size()
	if size < offset {
		offset = 0
	}
	if _, err = file.Seek(offset, io.SeekStart); err != nil {
		return offset, remainder, err
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return offset, remainder, err
	}
	if len(data) == 0 {
		return size, remainder, nil
	}
	chunk := remainder + string(data)
	lines := splitLinesKeepRemainder(chunk)
	for _, line := range lines.lines {
		fmt.Printf("[%s] %s\n", label, line)
	}
	return size, lines.remainder, nil
}

type lineSplitResult struct {
	lines     []string
	remainder string
}

func splitLinesKeepRemainder(s string) lineSplitResult {
	if s == "" {
		return lineSplitResult{}
	}
	hasTrailingNewline := strings.HasSuffix(s, "\n")
	raw := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	if hasTrailingNewline {
		raw = raw[:len(raw)-1]
		return lineSplitResult{lines: raw}
	}
	if len(raw) == 0 {
		return lineSplitResult{}
	}
	return lineSplitResult{
		lines:     raw[:len(raw)-1],
		remainder: raw[len(raw)-1],
	}
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func runDevLogsClean() error {
	rootDir, err := resolveRootDir()
	if err != nil {
		return err
	}
	baseLogDir := resolveBaseLogDir(rootDir)
	keepDays := 7
	if raw := strings.TrimSpace(os.Getenv("KEEP_DAYS")); raw != "" {
		if n, parseErr := strconv.Atoi(raw); parseErr == nil && n >= 0 {
			keepDays = n
		}
	}
	entries, err := os.ReadDir(baseLogDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	cutoff := time.Now().Add(-time.Duration(keepDays) * 24 * time.Hour)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "20") {
			continue
		}
		info, statErr := entry.Info()
		if statErr != nil {
			continue
		}
		if info.ModTime().After(cutoff) {
			continue
		}
		path := filepath.Join(baseLogDir, name)
		fmt.Println(path)
		if remErr := os.RemoveAll(path); remErr != nil {
			return remErr
		}
	}
	return nil
}

func resolveDevLogDir(rootDir string) (string, error) {
	baseLogDir := resolveBaseLogDir(rootDir)
	latestPath := filepath.Join(baseLogDir, "latest")
	if data, err := os.ReadFile(latestPath); err == nil {
		p := strings.TrimSpace(string(data))
		if p != "" {
			if info, statErr := os.Stat(p); statErr == nil && info.IsDir() {
				return p, nil
			}
		}
	}
	if info, err := os.Stat(baseLogDir); err == nil && info.IsDir() {
		entries, readErr := os.ReadDir(baseLogDir)
		if readErr == nil {
			dirs := make([]string, 0, len(entries))
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), "20") {
					dirs = append(dirs, entry.Name())
				}
			}
			sort.Strings(dirs)
			if len(dirs) > 0 {
				return filepath.Join(baseLogDir, dirs[len(dirs)-1]), nil
			}
		}
		return baseLogDir, nil
	}
	return "", fmt.Errorf("no dev logs found at %s", baseLogDir)
}

func resolveBaseLogDir(rootDir string) string {
	baseLogDir := envOrDefault("LOG_DIR", "/tmp/sentinel2-dev")
	if runtime.GOOS == "windows" && baseLogDir == "/tmp/sentinel2-dev" {
		baseLogDir = filepath.Join(os.TempDir(), "sentinel2-dev")
	}
	if !filepath.IsAbs(baseLogDir) {
		baseLogDir = filepath.Join(rootDir, baseLogDir)
	}
	return baseLogDir
}

func deriveBuildVersion() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("BUILD_VERSION")); explicit != "" {
		return explicit, nil
	}
	if _, err := runGit("rev-parse", "--is-inside-work-tree"); err != nil {
		return "", nil
	}
	branch := firstOrEmpty(runGit("rev-parse", "--abbrev-ref", "HEAD"))
	exactTag := firstOrEmpty(runGit("describe", "--tags", "--match", "v[0-9]*", "--exact-match"))
	latestTag := firstOrEmpty(runGit("describe", "--tags", "--match", "v[0-9]*", "--abbrev=0"))
	shortSHA := firstOrEmpty(runGit("rev-parse", "--short", "HEAD"))

	version := ""
	if exactTag != "" {
		version = exactTag
	} else {
		if latestTag != "" {
			version = latestTag
		} else {
			version = "v0.0.0"
		}
		if (branch == "main" || branch == "HEAD") && shortSHA != "" {
			version = version + "-" + shortSHA
		}
	}
	if version != "" {
		if dirty := firstOrEmpty(runGit("status", "--porcelain")); dirty != "" {
			version = version + "-dev"
		}
		if branch != "" && branch != "HEAD" && branch != "main" {
			version = version + "-branch-" + safeBranchName(branch)
		}
	}
	return version, nil
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func firstOrEmpty(value string, err error) string {
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func safeBranchName(branch string) string {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range branch {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}

func resolveRootDir() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("TASKUTIL_ROOT")); explicit != "" {
		return filepath.Abs(explicit)
	}
	start, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := start
	for {
		if looksLikeRepoRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if exe, exeErr := os.Executable(); exeErr == nil {
		exeDir := filepath.Dir(exe)
		candidate := filepath.Clean(filepath.Join(exeDir, "..", ".."))
		if looksLikeRepoRoot(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not resolve project root from %s", start)
}

func looksLikeRepoRoot(dir string) bool {
	if dir == "" {
		return false
	}
	required := []string{
		filepath.Join(dir, "Taskfile.yml"),
		filepath.Join(dir, "backend"),
		filepath.Join(dir, "frontend"),
		filepath.Join(dir, "taskutil"),
	}
	for _, p := range required {
		if _, err := os.Stat(p); err != nil {
			return false
		}
	}
	return true
}
