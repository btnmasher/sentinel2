package devconsole

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"sentinel2-taskutil/internal/project"

	tea "github.com/charmbracelet/bubbletea"
)

func TestProcessManagerStartAll_BackendStartFailureStopsFrontend(t *testing.T) {
	dir := t.TempDir()
	events := make(chan tea.Msg, 64)
	pm := &processManager{
		cfg:    project.Config{ExperimentalPTY: false},
		ctx:    context.Background(),
		events: events,
		processes: map[string]*managedProcess{
			"frontend": {spec: processSpec{
				name:    "frontend",
				dir:     dir,
				command: os.Args[0],
				args:    []string{"-test.run=TestProcessManagerHelperProcess", "--", "sleep"},
				env:     []string{"GO_WANT_HELPER_PROCESS=1"},
				logPath: filepath.Join(dir, "frontend.log"),
			}},
			"backend": {spec: processSpec{
				name:    "backend",
				dir:     dir,
				command: filepath.Join(dir, "missing-command"),
				logPath: filepath.Join(dir, "backend.log"),
			}},
		},
	}

	err := pm.startAll()
	if err == nil {
		t.Fatalf("startAll() expected error, got nil")
	}
	pm.processes["frontend"].mu.Lock()
	defer pm.processes["frontend"].mu.Unlock()
	if pm.processes["frontend"].cmd != nil {
		t.Fatalf("frontend cmd should be nil after backend start failure")
	}

	if pm.processes["frontend"].cancel != nil {
		t.Fatalf("frontend cancel should be nil after stop")
	}

	if pm.processes["frontend"].logFile != nil {
		t.Fatalf("frontend logFile should be nil after stop")
	}
}

func TestProcessManagerStart_ExperimentalPTYFallbackToPipes(t *testing.T) {
	dir := t.TempDir()
	events := make(chan tea.Msg, 64)
	pm := &processManager{
		cfg:    project.Config{ExperimentalPTY: true},
		ctx:    context.Background(),
		events: events,
		processes: map[string]*managedProcess{
			"frontend": {spec: processSpec{
				name:    "frontend",
				dir:     dir,
				command: os.Args[0],
				args:    []string{"-test.run=TestProcessManagerHelperProcess", "--", "echo"},
				env:     []string{"GO_WANT_HELPER_PROCESS=1"},
				logPath: filepath.Join(dir, "frontend.log"),
			}},
		},
	}

	prev := startPTYFn
	startPTYFn = func(_ *exec.Cmd) (*os.File, error) { return nil, errors.New("pty unavailable") }
	t.Cleanup(func() { startPTYFn = prev })

	if err := pm.start("frontend"); err != nil {
		t.Fatalf("start(frontend) error = %v", err)
	}
	t.Cleanup(func() { pm.stop("frontend") })

	var sawFallback bool
	timeout := time.After(2 * time.Second)
	for !sawFallback {
		select {
		case msg := <-events:
			if lm, ok := msg.(lineMsg); ok && lm.proc == "frontend" && lm.line == "[taskutil] experimental pty unavailable, falling back to stdio pipes" {
				sawFallback = true
			}
		case <-timeout:
			t.Fatalf("did not observe PTY fallback event")
		}
	}
}

func TestProcessManagerHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	mode := ""
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--" {
			mode = args[i+1]
			break
		}
	}
	switch mode {
	case "sleep":
		time.Sleep(5 * time.Second)
		os.Exit(0)
	case "echo":
		_, _ = os.Stdout.WriteString("hello\n")
		os.Exit(0)
	default:
		os.Exit(2)
	}
}
