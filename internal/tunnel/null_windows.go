//go:build windows

package tunnel

import (
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

func nullConfig() string { return "NUL" }

// Pasame corre sin consola (-H=windowsgui); cloudflared y taskkill son programas de consola y, sin esto,
// cada uno abriría una ventana negra.
const createNoWindow = 0x08000000

func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

func taskkill(args ...string) {
	c := exec.Command("taskkill", args...)
	hideConsole(c)
	c.Run()
}

// En Windows no hay SIGINT para otro proceso: se usa taskkill con el árbol.
func interrupt(cmd *exec.Cmd) { taskkill("/T", "/PID", strconv.Itoa(cmd.Process.Pid)) }

func kill(cmd *exec.Cmd) { taskkill("/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)) }

var stopGrace = 5 * time.Second
