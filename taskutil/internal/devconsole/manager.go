package devconsole

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
	"strings"
	"sync"
	"syscall"
	"time"

	"sentinel2-taskutil/internal/devlogs"
	"sentinel2-taskutil/internal/project"

	tea "github.com/charmbracelet/bubbletea"
)

type processManager struct {
	cfg           project.Config
	ctx           context.Context
	events        chan tea.Msg
	processes     map[string]*managedProcess
	runMigrations bool
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
	cmd     execCmd
	cancel  context.CancelFunc
	logFile *os.File
	ptyFile io.Closer
}

type execCmd interface {
	Wait() error
	ProcessID() int
	SignalInterrupt() error
	Kill() error
}

const (
	rebuildBackendTask  = "build:backend:skip-embed"
	rebuildFrontendTask = "build:frontend:dev"
)

var startPTYFn = startPTY

func Run(cfg project.Config, runMigrations bool) error {
	events := make(chan tea.Msg, 512)
	pm := newProcessManager(cfg, runMigrations, events)

	p := tea.NewProgram(newViewState(pm), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	pm.stopAll()
	return err
}

type osExecCmd struct {
	cmd *exec.Cmd
}

func (c *osExecCmd) Wait() error { return c.cmd.Wait() }
func (c *osExecCmd) ProcessID() int {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Pid
	}
	return 0
}
func (c *osExecCmd) SignalInterrupt() error {
	if c.cmd == nil || c.cmd.Process == nil {
		return nil
	}
	return signalInterrupt(c.cmd)
}
func (c *osExecCmd) Kill() error {
	if c.cmd == nil || c.cmd.Process == nil {
		return nil
	}
	return killProcessTree(c.cmd)
}

func newProcessManager(cfg project.Config, runMigrations bool, events chan tea.Msg) *processManager {
	baseLogDir := devlogs.BaseLogDir(cfg)
	sessionTS := time.Now().Format("20060102-150405")
	logDir := filepath.Join(baseLogDir, sessionTS)
	_ = os.MkdirAll(logDir, 0o755)
	_ = os.MkdirAll(baseLogDir, 0o755)
	_ = os.WriteFile(filepath.Join(baseLogDir, "latest"), []byte(logDir), 0o644)

	viteHost := cfg.ViteHost
	vitePort := cfg.VitePort
	devProxy := cfg.DevProxy
	jsonLogPath := cfg.LogJSONPath
	if strings.TrimSpace(jsonLogPath) == "" {
		jsonLogPath = filepath.Join(logDir, "backend.jsonl")
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	return &processManager{
		cfg:           cfg,
		ctx:           ctx,
		events:        events,
		runMigrations: runMigrations,
		processes: map[string]*managedProcess{
			"frontend": {
				spec: processSpec{
					name:    "frontend",
					dir:     cfg.FrontendDir(),
					command: "bun",
					args:    []string{"run", "dev", "--host", viteHost, "--port", vitePort},
					env: []string{
						"BUN_TMPDIR=" + filepath.Join(cfg.RootDir, ".tmp", "bun"),
						"BUN_INSTALL=" + filepath.Join(cfg.RootDir, ".tmp", "bun-install"),
					},
					logPath: filepath.Join(logDir, "vite.log"),
				},
			},
			"backend": {
				spec: processSpec{
					name:    "backend",
					dir:     cfg.BinDir(),
					command: cfg.BackendBinary(),
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
	usePTY := pm.cfg.ExperimentalPTY
	ctx, cancel := context.WithCancel(pm.ctx)
	cmd := pm.buildExecCommand(ctx, proc.spec, usePTY)

	procCmd := &osExecCmd{cmd: cmd}
	proc.cmd = procCmd
	proc.cancel = cancel
	proc.logFile = logFile

	if usePTY {
		ptyFile, ptyErr := startPTYFn(cmd)
		if ptyErr == nil {
			proc.ptyFile = ptyFile
			go pm.stream(name, ptyFile, logFile)
		} else {
			fmt.Fprintf(logFile, "[taskutil] experimental pty unavailable, falling back to stdio pipes: %v\n", ptyErr)
			pm.events <- lineMsg{
				proc: name,
				line: "[taskutil] experimental pty unavailable, falling back to stdio pipes",
			}
			// PTY setup may mutate cmd stdio fields. Recreate command for pipe mode.
			cmd = pm.buildExecCommand(ctx, proc.spec, false)
			procCmd.cmd = cmd
			if err := pm.startWithPipes(name, cmd, logFile); err != nil {
				proc.cmd = nil
				proc.cancel = nil
				proc.logFile = nil
				cancel()
				_ = logFile.Close()
				return err
			}
		}
	} else if err := pm.startWithPipes(name, cmd, logFile); err != nil {
		proc.cmd = nil
		proc.cancel = nil
		proc.logFile = nil
		cancel()
		_ = logFile.Close()
		return err
	}
	if procCmd.ProcessID() == 0 {
		proc.cmd = nil
		proc.cancel = nil
		proc.logFile = nil
		cancel()
		_ = logFile.Close()
		return fmt.Errorf("failed to start process %q", name)
	}
	go pm.wait(name, proc.cmd)
	pm.events <- procStartedMsg{proc: name, pid: proc.cmd.ProcessID()}
	return nil
}

func (pm *processManager) buildExecCommand(ctx context.Context, spec processSpec, usePTY bool) *exec.Cmd {
	cmd := exec.CommandContext(ctx, spec.command, spec.args...)
	cmd.Dir = spec.dir
	cmd.Env = append(os.Environ(), spec.env...)
	configureProcess(cmd, usePTY)
	return cmd
}

func (pm *processManager) startWithPipes(name string, cmd *exec.Cmd, logFile *os.File) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go pm.stream(name, stdout, logFile)
	go pm.stream(name, stderr, logFile)
	return nil
}

func (pm *processManager) stream(name string, reader io.Reader, logFile *os.File) {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 1024*64)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(logFile, line)
		pm.events <- lineMsg{proc: name, line: normalizeLineForViewport(line)}
	}
}

func (pm *processManager) wait(name string, cmd execCmd) {
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
	ptyFile := proc.ptyFile
	proc.cmd = nil
	proc.cancel = nil
	proc.logFile = nil
	proc.ptyFile = nil
	proc.mu.Unlock()

	if cmd != nil {
		_ = cmd.SignalInterrupt()
		time.Sleep(250 * time.Millisecond)
		_ = cmd.Kill()
	}
	if cancel != nil {
		cancel()
	}
	if ptyFile != nil {
		_ = ptyFile.Close()
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
	cmd := exec.Command("task", rebuildBackendTask)
	cmd.Dir = pm.cfg.RootDir
	return pm.runAuxCommand("backend", cmd)
}

func (pm *processManager) rebuildFrontend() error {
	cmd := exec.Command("task", rebuildFrontendTask)
	cmd.Dir = pm.cfg.RootDir
	return pm.runAuxCommand("frontend", cmd)
}

func (pm *processManager) runMigrate() error {
	cmd := exec.Command(pm.cfg.BackendBinary(), "migrate")
	cmd.Dir = pm.cfg.BinDir()
	return pm.runAuxCommand("backend", cmd)
}

func (pm *processManager) runAuxCommand(proc string, cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		pm.streamAux(proc, stdout)
	}()
	go func() {
		defer wg.Done()
		pm.streamAux(proc, stderr)
	}()

	waitErr := cmd.Wait()
	wg.Wait()
	return waitErr
}

func (pm *processManager) streamAux(proc string, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 1024*64)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		pm.events <- lineMsg{proc: proc, line: normalizeLineForViewport(scanner.Text())}
	}
}
