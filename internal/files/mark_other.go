//go:build !darwin && !windows

package files

// MarkDownloaded no hace nada en Linux: no hay un equivalente estándar.
func MarkDownloaded(path string) error { return nil }
