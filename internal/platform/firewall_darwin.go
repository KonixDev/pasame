//go:build darwin

package platform

import (
	"os/exec"
	"strings"
)

// FirewallHint devuelve "mac" solo si el firewall de macOS está activado (viene apagado de fábrica).
func FirewallHint() string {
	out, err := exec.Command("defaults", "read", "/Library/Preferences/com.apple.alf", "globalstate").Output()
	if err == nil && strings.TrimSpace(string(out)) != "0" {
		return "mac"
	}
	return ""
}

func OpenFirewallSettings() error { return nil }
