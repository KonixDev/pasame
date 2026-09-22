// Package dialog abre el selector nativo del OS (sin CGO: Win32, osascript, zenity/kdialog).
package dialog

import (
	"errors"
	"os/exec"

	"github.com/ncruces/zenity"
)

var (
	ErrCanceled    = errors.New("dialog: cancelado")
	ErrUnsupported = errors.New("dialog: no hay diálogo nativo")
)

func mapErr(err error) error {
	var execErr *exec.Error
	switch {
	case err == nil:
		return nil
	case errors.Is(err, zenity.ErrCanceled):
		return ErrCanceled
	case errors.Is(err, zenity.ErrUnsupported), errors.As(err, &execErr):
		return ErrUnsupported
	}
	return err
}

func PickFiles() ([]string, error) {
	paths, err := zenity.SelectFileMultiple(zenity.Title("Elegí los archivos para pasar"))
	return paths, mapErr(err)
}

func PickFolder() (string, error) {
	p, err := zenity.SelectFile(zenity.Directory(), zenity.Title("Elegí la carpeta para pasar"))
	return p, mapErr(err)
}
