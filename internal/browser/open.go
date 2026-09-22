// Package browser abre la UI en el navegador por defecto.
package browser

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/ncruces/zenity"
)

func command(url string) (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		return "xdg-open", []string{url}
	}
}

// Open abre url; si no puede, se la muestra a la persona para que la copie.
func Open(url string) error {
	name, args := command(url)
	err := exec.Command(name, args...).Start()
	if err != nil {
		// Todavía no se sabe el idioma de la persona: el aviso va en los dos.
		fmt.Fprintln(os.Stderr, "Abrí esta dirección en tu navegador / Open this address in your browser:", url)
		zenity.Info("Abrí esta dirección en tu navegador.\nOpen this address in your browser.\n\n"+url, zenity.Title("Pasame"))
	}
	return err
}
