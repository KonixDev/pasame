//go:build windows

package files

import "os"

// MarkDownloaded escribe el Zone.Identifier que usan los navegadores (ZoneId=3 = internet).
func MarkDownloaded(path string) error {
	return os.WriteFile(path+":Zone.Identifier", []byte("[ZoneTransfer]\r\nZoneId=3\r\n"), 0o644)
}
