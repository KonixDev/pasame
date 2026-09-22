package dialog

import (
	"errors"
	"testing"

	"github.com/ncruces/zenity"
)

func TestMapErr(t *testing.T) {
	if !errors.Is(mapErr(zenity.ErrCanceled), ErrCanceled) {
		t.Fatal("cancelar")
	}
	if !errors.Is(mapErr(zenity.ErrUnsupported), ErrUnsupported) {
		t.Fatal("sin soporte")
	}
	if mapErr(nil) != nil {
		t.Fatal("nil")
	}
}
