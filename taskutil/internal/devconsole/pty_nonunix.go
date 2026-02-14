//go:build !unix

package devconsole

import (
	"errors"
	"os"
	"os/exec"
)

var errPTYUnsupported = errors.New("pty is not supported on this platform")

func startPTY(_ *exec.Cmd) (*os.File, error) {
	return nil, errPTYUnsupported
}
