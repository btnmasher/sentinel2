//go:build !unix

package devconsole

import (
	"os"
	"os/exec"
)

func configureProcess(_ *exec.Cmd, _ bool) {}

func signalInterrupt(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Signal(os.Interrupt)
}

func killProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
