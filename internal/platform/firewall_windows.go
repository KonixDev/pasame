//go:build windows

package platform

import "os/exec"

// FirewallHint: en Windows siempre puede aparecer el diálogo; core decide mostrarlo solo en la primera ejecución.
func FirewallHint() string { return "windows" }

// OpenFirewallSettings abre "Permitir aplicaciones a través del Firewall de Windows".
func OpenFirewallSettings() error { return exec.Command("control", "firewall.cpl").Start() }
