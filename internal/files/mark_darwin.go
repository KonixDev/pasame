//go:build darwin

package files

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

// MarkDownloaded hace que Gatekeeper trate el archivo como bajado de internet.
func MarkDownloaded(path string) error {
	v := fmt.Sprintf("0081;%08x;Pasame;", time.Now().Unix())
	return unix.Setxattr(path, "com.apple.quarantine", []byte(v), 0)
}
