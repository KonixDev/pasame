// Package session modela un envío: qué archivos, con qué token y PIN. Sin red, sin exec.
package session

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type File struct {
	Index   int
	Name    string // nombre para mostrar y para Content-Disposition
	Rel     string // ruta dentro del ZIP, con "/"
	Abs     string // ruta en disco; nunca sale del proceso
	Size    int64
	ModTime time.Time
}

type Session struct {
	Token     string
	PIN       string
	Files     []File // snapshot inmutable
	CreatedAt time.Time
}

// New arma una sesión. Con paths vacío es el modo "Solo recibir".
// Las rutas que no se pueden leer vuelven como errores y se omiten.
func New(paths []string) (*Session, []error) {
	s := &Session{Token: NewToken(), PIN: NewPIN(), CreatedAt: time.Now()}
	var errs []error
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		info, err := os.Stat(abs)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", filepath.Base(p), err))
			continue
		}
		if info.IsDir() {
			fs, _ := walk(abs)
			s.Files = append(s.Files, fs...)
			continue
		}
		s.Files = append(s.Files, File{
			Name: info.Name(), Rel: info.Name(), Abs: abs,
			Size: info.Size(), ModTime: info.ModTime(),
		})
	}
	for i := range s.Files {
		s.Files[i].Index = i
	}
	return s, errs
}

func (s *Session) File(i int) (File, bool) {
	if i < 0 || i >= len(s.Files) {
		return File{}, false
	}
	return s.Files[i], true
}

func (s *Session) TotalSize() int64 {
	var n int64
	for _, f := range s.Files {
		n += f.Size
	}
	return n
}

func (s *Session) Path() string { return "/s/" + s.Token }
