// Package platform aísla lo que cambia por sistema operativo.
package platform

import (
	"os"
	"path/filepath"
)

func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(base, "Pasame")
	return d, os.MkdirAll(d, 0o755)
}
