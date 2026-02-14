//go:build unix

package devconsole

import (
	"os/exec"
	"syscall"
)

func configureProcess(cmd *exec.Cmd, usePTY bool) {
	if usePTY {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func signalInterrupt(cmd *exec.Cmd) error {
	pid := cmd.Process.Pid
	if pid <= 0 {
		return nil
	}
	return syscall.Kill(-pid, syscall.SIGINT)
}

func killProcessTree(cmd *exec.Cmd) error {
	pid := cmd.Process.Pid
	if pid <= 0 {
		return nil
	}
	return syscall.Kill(-pid, syscall.SIGKILL)
}
