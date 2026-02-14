//go:build unix

package devconsole

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
)

func startPTY(cmd *exec.Cmd) (*os.File, error) {
	return pty.Start(cmd)
}
