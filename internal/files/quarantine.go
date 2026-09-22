package files

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// QuarantinePath devuelve la carpeta donde caen los archivos recibidos, sin crearla ni entrar.
// En macOS, acceder a Descargas dispara un pedido de permiso: tiene que pasar cuando llega el
// primer archivo (con la app ya abierta y con sentido), no al hacer doble clic.
func QuarantinePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	base := filepath.Join(home, "Downloads")
	if runtime.GOOS == "linux" {
		base = linuxDownloads(home)
	}
	return filepath.Join(base, "Pasame"), nil
}

// QuarantineDir es QuarantinePath más crear la carpeta.
func QuarantineDir() (string, error) {
	dir, err := QuarantinePath()
	if err != nil {
		return "", err
	}
	return dir, os.MkdirAll(dir, 0o755)
}

func linuxDownloads(home string) string {
	if x := os.Getenv("XDG_DOWNLOAD_DIR"); x != "" {
		return x
	}
	for _, n := range []string{"Descargas", "Downloads"} {
		if fi, err := os.Stat(filepath.Join(home, n)); err == nil && fi.IsDir() {
			return filepath.Join(home, n)
		}
	}
	return home
}

// UniquePath sanea name y devuelve una ruta libre dentro de dir.
// Una ruta está ocupada si existe el archivo o su ".part".
func UniquePath(dir, name string) (string, error) {
	clean, err := Sanitize(name)
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(clean)
	stem := strings.TrimSuffix(clean, ext)
	for i := 1; i < 10000; i++ {
		n := clean
		if i > 1 {
			n = fmt.Sprintf("%s (%d)%s", stem, i, ext)
		}
		p := filepath.Join(dir, n)
		if filepath.Dir(p) != filepath.Clean(dir) {
			return "", ErrBadName
		}
		if !exists(p) && !exists(p+".part") {
			return p, nil
		}
	}
	return "", errors.New("files: demasiadas colisiones")
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return !errors.Is(err, fs.ErrNotExist)
}
