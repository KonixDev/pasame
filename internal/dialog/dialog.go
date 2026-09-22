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

func PickFiles(title string) ([]string, error) {
	paths, err := zenity.SelectFileMultiple(zenity.Title(title))
	return paths, mapErr(err)
}

func PickFolder(title string) (string, error) {
	p, err := zenity.SelectFile(zenity.Directory(), zenity.Title(title))
	return p, mapErr(err)
}
