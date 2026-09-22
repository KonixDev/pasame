package share

import (
	"net/http/httptest"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestPageListsFiles(t *testing.T) {
	f := newFixture(t, map[string]int{"contrato.pdf": 1300, "foto.jpg": 4200, "notas.docx": 10})
	w := f.do(httptest.NewRequest("GET", f.sess.Path(), nil))
	body := w.Body.String()
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("%d %q", w.Code, w.Header().Get("Cache-Control"))
	}
	for _, want := range []string{
		"Martín te comparte 3 archivos",
		`href="` + f.sess.Path() + `/zip"`,
		"Descargar todo (5,5 KB)",
		`href="` + f.sess.Path() + `/f/0"`, "contrato.pdf",
		`href="` + f.sess.Path() + `/f/1?inline=1"`, // foto.jpg tiene "Ver"
		"¿Querés mandarle algo a Martín?",
		`enctype="multipart/form-data"`, `action="` + f.sess.Path() + `/up"`,
		`type="file"`, "multiple",
		"Los archivos viajan solo por esta red.",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("falta %q", want)
		}
	}
	// notas.docx no es visualizable: no hay link inline para /f/2
	if strings.Contains(body, "/f/2?inline=1") {
		t.Error("docx no debería tener Ver")
	}
}

func TestPageWorksWithoutJS(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	body := f.do(httptest.NewRequest("GET", f.sess.Path(), nil)).Body.String()
	// Hay un botón Enviar real dentro del form (lo oculta el JS, no el CSS).
	if !regexp.MustCompile(`<button[^>]*id="up-send"[^>]*type="submit"`).MatchString(body) {
		t.Fatal("falta el botón submit para navegadores sin JS")
	}
	if len(body) > 30*1024 {
		t.Fatalf("página de %d bytes (> 30 KB)", len(body))
	}
}

func TestPageSingular(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	if body := f.do(httptest.NewRequest("GET", f.sess.Path(), nil)).Body.String(); !strings.Contains(body, "Martín te comparte 1 archivo<") {
		t.Fatal("falta el singular")
	}
}

func TestPageReceiveOnly(t *testing.T) {
	f := newFixture(t, nil)
	body := f.do(httptest.NewRequest("GET", f.sess.Path(), nil)).Body.String()
	if !strings.Contains(body, "Mandale archivos a Martín") || strings.Contains(body, "/zip") {
		t.Fatalf("body: %s", body)
	}
}

func TestPageUploadResult(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	body := f.do(httptest.NewRequest("GET", f.sess.Path()+"?subido=5", nil)).Body.String()
	if !strings.Contains(body, "✓ Enviaste 5 archivos.") {
		t.Fatal("no muestra el resultado de la subida")
	}
	body = f.do(httptest.NewRequest("GET", f.sess.Path()+"?subido=1", nil)).Body.String()
	if !strings.Contains(body, "✓ Enviaste 1 archivo.") {
		t.Fatal("singular mal")
	}
	body = f.do(httptest.NewRequest("GET", f.sess.Path()+"?error=err_empty", nil)).Body.String()
	if !strings.Contains(body, "Ese archivo está vacío.") {
		t.Fatal("no muestra el error")
	}
	body = f.do(httptest.NewRequest("GET", f.sess.Path()+"?error=gone_title", nil)).Body.String()
	if strings.Contains(body, "Este envío ya terminó") {
		t.Fatal("aceptó una clave que no es err_*")
	}
}

func TestPageEscapesNames(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("'<' no es válido en nombres de Windows")
	}
	f := newFixture(t, map[string]int{"<script>x.txt": 1})
	body := f.do(httptest.NewRequest("GET", f.sess.Path(), nil)).Body.String()
	if strings.Contains(body, "<script>x") {
		t.Fatal("nombre sin escapar")
	}
}

func TestViewable(t *testing.T) {
	yes := []string{"a.JPG", "b.png", "c.heic", "d.mp4", "e.webm", "f.mp3", "g.m4a", "h.pdf"}
	no := []string{"a.docx", "b.zip", "c.mov", "d", "e.exe"}
	for _, n := range yes {
		if !Viewable(n) {
			t.Errorf("%s debería ser visualizable", n)
		}
	}
	for _, n := range no {
		if Viewable(n) {
			t.Errorf("%s no debería ser visualizable", n)
		}
	}
}

func TestTypeLabel(t *testing.T) {
	cases := map[string]string{"a.mp4": "MP4", "Foto.JPEG": "JPEG", "sin-extension": "—", "raro.abcdefg": "—", "x.pdf": "PDF"}
	for in, want := range cases {
		if got := typeLabel(in); got != want {
			t.Errorf("typeLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

// Mejoras de UX: "Ver" no reemplaza la página, el estado se anuncia, y el JS recibe los textos nuevos.
func TestPageUXHooks(t *testing.T) {
	f := newFixture(t, map[string]int{"foto.jpg": 10})
	body := f.do(httptest.NewRequest("GET", f.sess.Path(), nil)).Body.String()
	for _, want := range []string{
		`href="` + f.sess.Path() + `/f/0?inline=1" target="_blank" rel="noopener"`,
		`id="up-status" class="note" role="status" aria-live="polite"`,
		`id="dl-hint"`, `id="drop"`,
		"Empezó la descarga.",
		"Soltá los archivos acá para mandárselos a Martín",
		"__FILES__",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("falta %q", want)
		}
	}
}
