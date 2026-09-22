package control

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/KonixDev/pasame/web"
)

func read(t *testing.T, name string) string {
	b, err := fs.ReadFile(web.Control, name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Spec §6: nunca aparecen estas palabras en textos que ve la persona.
func TestUIVoice(t *testing.T) {
	js := read(t, "control/app.js")
	start := strings.Index(js, "var T = {")
	end := strings.Index(js[start:], "};")
	if start < 0 || end < 0 {
		t.Fatal("no encuentro el objeto T en app.js")
	}
	texts := strings.ToLower(js[start : start+end])
	texts = regexp.MustCompile(`(?m)^\s*//.*$`).ReplaceAllString(texts, "") // comentarios de código
	// "Firewall de Windows Defender" es el título literal de la ventana que la persona va a ver: se permite.
	texts = strings.ReplaceAll(texts, "firewall de windows defender", "")
	// \b evita falsos positivos como "aeropuerto".
	for _, w := range []string{`\bip\b`, `\bpuerto`, `\bservidor`, `túnel`, `\btunel`, `\bfirewall`, `\btoken`, `\bmdns`, `\blan\b`} {
		if regexp.MustCompile(w).MatchString(texts) {
			t.Errorf("la UI usa la palabra prohibida %q", w)
		}
	}
}

func TestUIStructure(t *testing.T) {
	html := read(t, "control/index.html")
	for _, id := range []string{`id="app"`, `src="app.js"`, `href="app.css"`} {
		if !strings.Contains(html, id) {
			t.Errorf("index.html sin %s", id)
		}
	}
	css := read(t, "control/app.css")
	if !strings.Contains(css, "280px") || !strings.Contains(css, "480px") {
		t.Error("el QR tiene que medir entre 280 y 480 px")
	}
	js := read(t, "control/app.js")
	for _, must := range []string{"X-Pasame-Token", "EventSource", "function esc(", "45000"} {
		if !strings.Contains(js, must) {
			t.Errorf("app.js sin %q", must)
		}
	}
}
