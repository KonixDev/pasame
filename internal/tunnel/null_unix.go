//go:build !windows

package tunnel

import (
	"os"
	"os/exec"
	"time"
)

func nullConfig() string { return "/dev/null" }

func hideConsole(*exec.Cmd) {}

func interrupt(cmd *exec.Cmd) { cmd.Process.Signal(os.Interrupt) }

func kill(cmd *exec.Cmd) { cmd.Process.Kill() }

var stopGrace = 5 * time.Second
