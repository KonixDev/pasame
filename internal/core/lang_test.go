package core

import (
	"testing"

	"github.com/KonixDev/pasame/internal/i18n"
	"github.com/KonixDev/pasame/internal/platform"
)

func TestLangAutoDetectedIsNotPersisted(t *testing.T) {
	e := newEnv(t)
	if e.c.Lang() != i18n.ES || e.c.State().Lang != "es" {
		t.Fatal("por defecto: español")
	}
	if err := e.c.SetLang("en", false); err != nil {
		t.Fatal(err)
	}
	s := e.c.State()
	if e.c.Lang() != i18n.EN || s.Lang != "en" || s.LangExplicit {
		t.Fatalf("%+v", s)
	}
	if cfg, _, _ := platform.LoadConfig(e.cfg); cfg.Lang != "" {
		t.Fatal("lo detectado del navegador no se guarda: es una suposición")
	}
}

func TestLangExplicitWinsAndPersists(t *testing.T) {
	e := newEnv(t)
	e.c.SetLang("en", true)
	e.c.SetLang("es", false) // una detección posterior no pisa lo que eligió la persona
	if e.c.Lang() != i18n.EN || !e.c.State().LangExplicit {
		t.Fatal("la elección explícita tiene que ganar")
	}
	if cfg, _, _ := platform.LoadConfig(e.cfg); cfg.Lang != "en" {
		t.Fatal("la elección explícita se guarda")
	}
	if err := e.c.SetLang("fr", true); err == nil {
		t.Fatal("idioma no soportado aceptado")
	}
}
