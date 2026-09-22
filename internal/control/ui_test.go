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

// Spec §6: nunca aparecen estas palabras en textos que ve la persona, en ningún idioma.
func TestUIVoice(t *testing.T) {
	js := read(t, "control/app.js")
	start := strings.Index(js, "var TT = {")
	end := strings.Index(js[start:], "\n  };")
	if start < 0 || end < 0 {
		t.Fatal("no encuentro el objeto TT en app.js")
	}
	block := js[start : start+end]
	es, en := block[:strings.Index(block, "\n    en: {")], block[strings.Index(block, "\n    en: {"):]
	clean := func(s string) string {
		s = strings.ToLower(s)
		s = regexp.MustCompile(`(?m)^\s*//.*$`).ReplaceAllString(s, "")  // comentarios de código
		s = regexp.MustCompile(`(?m)^\s*\w+:`).ReplaceAllString(s, "")   // nombres de clave (no se ven)
		s = regexp.MustCompile(`[\w-]+="[^"]*"`).ReplaceAllString(s, "") // atributos HTML (data-act="tunnel-on")
		s = strings.ReplaceAll(s, `\'`, "'")                             // comillas escapadas del JS
		// Títulos literales de ventanas del sistema que la persona va a ver: se permiten.
		s = strings.ReplaceAll(s, "firewall de windows defender", "")
		s = strings.ReplaceAll(s, "windows defender firewall", "")
		// Aviso de privacidad del túnel (Plan 2): tiene que decir exactamente por dónde pasan los archivos.
		s = strings.ReplaceAll(s, "los servidores de cloudflare", "")
		s = strings.ReplaceAll(s, "cloudflare's servers", "")
		return s
	}
	// \b evita falsos positivos como "aeropuerto" o "support".
	banned := map[string][]string{
		"es": {`\bip\b`, `\bpuerto`, `\bservidor`, `túnel`, `\btunel`, `\bfirewall`, `\btoken`, `\bmdns`, `\blan\b`},
		"en": {`\bip\b`, `\bport\b`, `\bserver`, `\btunnel`, `\bfirewall`, `\btoken`, `\bmdns`, `\blan\b`, `\blocalhost`, `\bupload`},
	}
	for lang, texts := range map[string]string{"es": clean(es), "en": clean(en)} {
		for _, w := range banned[lang] {
			if regexp.MustCompile(w).MatchString(texts) {
				t.Errorf("la UI (%s) usa la palabra prohibida %q", lang, w)
			}
		}
	}
}

// Las dos versiones tienen exactamente las mismas claves.
func TestUILangParity(t *testing.T) {
	js := read(t, "control/app.js")
	start := strings.Index(js, "var TT = {")
	block := js[start : start+strings.Index(js[start:], "\n  };")]
	i := strings.Index(block, "\n    en: {")
	keys := func(s string) map[string]bool {
		m := map[string]bool{}
		for _, k := range regexp.MustCompile(`(?m)^      (\w+):`).FindAllStringSubmatch(s, -1) {
			m[k[1]] = true
		}
		return m
	}
	es, en := keys(block[:i]), keys(block[i:])
	for k := range es {
		if !en[k] {
			t.Errorf("falta %q en inglés", k)
		}
	}
	for k := range en {
		if !es[k] {
			t.Errorf("falta %q en español", k)
		}
	}
	if len(es) < 50 {
		t.Fatalf("solo encontré %d claves: ¿cambió el formato?", len(es))
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
