// Package files maneja nombres y rutas de archivos recibidos: nada de red, nada de exec.
package files

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// ErrBadName indica que el nombre no se puede usar ni saneado.
var ErrBadName = errors.New("files: nombre inválido")

const maxNameBytes = 200

var reserved = map[string]bool{"CON": true, "PRN": true, "AUX": true, "NUL": true}

func init() {
	for i := '1'; i <= '9'; i++ {
		reserved["COM"+string(i)] = true
		reserved["LPT"+string(i)] = true
	}
}

// Sanitize convierte un nombre que viene de un navegador en uno seguro para
// cualquier sistema de archivos. Nunca devuelve algo con separadores de ruta.
func Sanitize(name string) (string, error) {
	// Último componente, tratando / y \ como separadores en cualquier OS.
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	var b strings.Builder
	for _, r := range name {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(`:*?"<>|`, r) {
			continue
		}
		b.WriteRune(r)
	}
	s := strings.TrimSpace(b.String())
	s = strings.TrimRight(s, ". ")
	if s == "" || strings.Trim(s, ".") == "" {
		return "", ErrBadName
	}
	stem := strings.ToUpper(strings.TrimSuffix(s, filepath.Ext(s)))
	if reserved[stem] {
		return "", ErrBadName
	}
	return truncate(s), nil
}

func truncate(s string) string {
	if len(s) <= maxNameBytes {
		return s
	}
	ext := filepath.Ext(s)
	if len(ext) > 20 {
		ext = ""
	}
	stem := s[:len(s)-len(ext)]
	limit := maxNameBytes - len(ext)
	for limit > 0 && !utf8.RuneStart(stem[limit]) {
		limit--
	}
	return stem[:limit] + ext
}
