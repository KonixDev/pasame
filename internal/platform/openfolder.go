package platform

import (
	"os/exec"
	"runtime"
)

func openCommand(path string) (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{path}
	case "windows":
		return "explorer", []string{path}
	default:
		return "xdg-open", []string{path}
	}
}

// OpenFolder abre la carpeta en Finder / Explorador / el gestor de archivos.
func OpenFolder(path string) error {
	name, args := openCommand(path)
	return exec.Command(name, args...).Start()
}
