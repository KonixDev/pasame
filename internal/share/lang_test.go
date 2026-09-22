package share

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// El idioma por defecto es el de quien comparte, no el del navegador de quien recibe.
func TestLangDefaultsToSender(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	r := httptest.NewRequest("GET", f.sess.Path(), nil)
	r.Header.Set("Accept-Language", "en-US,en;q=0.9")
	body := f.do(r).Body.String()
	if !strings.Contains(body, "Martín te comparte 1 archivo") {
		t.Fatalf("con navegador en inglés debería seguir en español: %s", body)
	}
	// Pero se le ofrece el cambio arriba, porque su navegador prefiere otro idioma.
	if !strings.Contains(body, `id="lang-top"`) || !strings.Contains(body, `href="?lang=en"`) {
		t.Fatal("falta la sugerencia de cambiar a inglés")
	}
}

func TestLangNoSuggestionWhenBrowserMatches(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	r := httptest.NewRequest("GET", f.sess.Path(), nil)
	r.Header.Set("Accept-Language", "es-AR,es;q=0.9")
	body := f.do(r).Body.String()
	if strings.Contains(body, `id="lang-top"`) {
		t.Fatal("no hace falta sugerir si el navegador ya está en español")
	}
	if !strings.Contains(body, `href="?lang=en">English</a>`) {
		t.Fatal("el cambio de idioma siempre está disponible abajo")
	}
}

func TestLangSwitchSetsCookie(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	w := f.do(httptest.NewRequest("GET", f.sess.Path()+"?lang=en", nil))
	if !strings.Contains(w.Body.String(), "Martín is sharing 1 file with you") {
		t.Fatal("?lang=en no cambió el idioma")
	}
	var c *http.Cookie
	for _, k := range w.Result().Cookies() {
		if k.Name == "pasame_lang" {
			c = k
		}
	}
	if c == nil || c.Value != "en" || c.Path != "/" || c.MaxAge <= 0 {
		t.Fatalf("cookie %+v", c)
	}
	r := httptest.NewRequest("GET", f.sess.Path(), nil)
	r.AddCookie(c)
	if !strings.Contains(f.do(r).Body.String(), "is sharing") {
		t.Fatal("la cookie no recordó el idioma")
	}
}

func TestLangIgnoresUnknown(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	w := f.do(httptest.NewRequest("GET", f.sess.Path()+"?lang=fr", nil))
	if !strings.Contains(w.Body.String(), "te comparte") || len(w.Result().Cookies()) != 0 {
		t.Fatal("un idioma desconocido no debe cambiar nada")
	}
}
