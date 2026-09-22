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

// En Windows no hay forma ordenada de pedirle a un programa de consola sin ventana que termine: taskkill
// sin /F manda WM_CLOSE, que cloudflared nunca recibe, y cada apagado esperaba los 5 s de stopGrace.
// Cortarlo de una es seguro: Cloudflare da de baja el túnel cuando se cae la conexión.
func interrupt(cmd *exec.Cmd) { kill(cmd) }

func kill(cmd *exec.Cmd) { taskkill("/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)) }

var stopGrace = 5 * time.Second
