//go:build !darwin && !windows

package platform

func FirewallHint() string        { return "" }
func OpenFirewallSettings() error { return nil }
