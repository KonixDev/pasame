package site

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func read(t *testing.T, n string) string {
	b, err := os.ReadFile(n)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

var assets = []string{
	"Pasame-Windows.exe", "Pasame-Windows-arm64.exe", "Pasame-macOS.dmg",
	"pasame-linux-amd64.tar.gz", "pasame-linux-arm64.tar.gz", "pasame-linux-armv7.tar.gz",
	"api.github.com/repos/KonixDev/pasame/releases?per_page=1",
	"https://github.com/KonixDev/pasame/releases", // sin JS: link a todas las descargas
}

func TestIndex(t *testing.T) {
	for file, texts := range map[string][]string{
		"index.html":    {"Descargar Pasame", "como-abrir.html", "Desde el celular no hace falta instalar nada", `href="en/"`},
		"en/index.html": {"Download Pasame", "how-to-open.html", "Nothing to install on the phone", `href="../"`},
	} {
		h := read(t, file)
		for _, must := range append(texts, assets...) {
			if !strings.Contains(h, must) {
				t.Errorf("%s sin %q", file, must)
			}
		}
		if regexp.MustCompile(`(?i)<script[^>]+src=`).MatchString(h) {
			t.Errorf("%s carga scripts externos", file)
		}
	}
}

func TestComoAbrir(t *testing.T) {
	for file, texts := range map[string][]string{
		"como-abrir.html":     {`id="windows"`, `id="mac"`, "Más información", "Ejecutar de todas formas", "Abrir de todos modos", "Privacidad y seguridad"},
		"en/how-to-open.html": {`id="windows"`, `id="mac"`, "More info", "Run anyway", "Open Anyway", "Privacy &amp; Security"},
	} {
		h := read(t, file)
		for _, must := range texts {
			if !strings.Contains(h, must) {
				t.Errorf("%s sin %q", file, must)
			}
		}
		dir := strings.TrimSuffix(file, "how-to-open.html")
		dir = strings.TrimSuffix(dir, "como-abrir.html")
		for _, img := range regexp.MustCompile(`src="([^"]+\.(?:png|svg))"`).FindAllStringSubmatch(h, -1) {
			if _, err := os.Stat(dir + img[1]); err != nil {
				t.Errorf("%s: falta la imagen %s", file, img[1])
			}
		}
	}
}

// Sin dominio propio todavía: GitHub Pages sirve en konixdev.github.io/pasame/, así que todos los links
// son relativos. Con pasame.com.ar alcanza con agregar site/CNAME (ver test/manual/tunel.md).
func TestRelativeLinks(t *testing.T) {
	for _, f := range []string{"index.html", "como-abrir.html", "en/index.html", "en/how-to-open.html"} {
		if m := regexp.MustCompile(`(?:href|src)="/[^/]`).FindString(read(t, f)); m != "" {
			t.Errorf("%s tiene una ruta absoluta (%s): se rompe en konixdev.github.io/pasame/", f, m)
		}
	}
}
