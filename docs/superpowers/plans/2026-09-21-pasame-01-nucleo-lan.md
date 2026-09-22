# Pasame — Plan 1: Núcleo LAN — Plan de implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Un binario `pasame` que, con doble clic, abre una UI en el navegador local, deja elegir archivos, muestra un QR y una dirección corta, y permite que N dispositivos de la misma red WiFi descarguen (archivo por archivo o ZIP al vuelo) y suban archivos de vuelta, sin instalar nada del lado del receptor.

**Architecture:** Un proceso Go con dos listeners: `control` (`127.0.0.1:<aleatorio>`, UI del emisor protegida por token `t`) y `share` (`0.0.0.0:8080`, página del receptor que funciona sin JS). `internal/core` orquesta sesión, direcciones (`addr.Book` con proveedores enchufables) y estadísticas; la UI recibe el estado por SSE. Este plan implementa solo el proveedor LAN; el túnel y el PIN llegan en el Plan 2 sin tocar `share` más que para agregar un middleware.

**Tech Stack:** Go 1.27 (`net/http`, `archive/zip`, `embed`, `crypto/rand`), `github.com/ncruces/zenity`, `github.com/skip2/go-qrcode`, `golang.org/x/sys` (lock en Windows). Frontend HTML/CSS/JS vanilla embebido. Playwright (solo tests de la página del receptor).

**Spec:** `docs/superpowers/specs/2026-09-21-share-now-design.md` — leelo antes de empezar. Cada tarea cita la sección que implementa.

**Plan siguiente:** `docs/superpowers/plans/2026-09-21-pasame-02-tunel-y-distribucion.md` (PIN, túnel, mDNS, empaquetado, releases, sitio `pasame.com.ar`).

## Global Constraints

- **Los dos objetivos que mandan (spec §0–1):** máximo alcance de dispositivos y máxima simpleza de UI/UX. Ante cualquier duda de implementación no cubierta por el plan, gana la opción que suma dispositivos o resta pasos.
- Módulo: `github.com/KonixDev/pasame` (usuario de GitHub verificado con `gh api user`). Directiva `go 1.27`.
- Toolchain en la máquina de desarrollo: Go 1.27.x (hoy **no está instalado**: la Tarea 1 lo instala con `brew install go`) y Node 20+ solo para Playwright (Tarea 21).
- **`CGO_ENABLED=0` siempre.** Cualquier dependencia que requiera CGO queda descartada.
- Dependencias directas permitidas: `github.com/ncruces/zenity`, `github.com/skip2/go-qrcode`, `golang.org/x/sys`. En tests además se permite un decodificador de QR (`github.com/makiuchi-d/gozxing`). Nada más sin actualizar el spec.
- Listener `share`: puerto 8080 → 8081…8089 → aleatorio. Listener `control`: `127.0.0.1:0` siempre.
- Timeouts de `share`: `ReadHeaderTimeout: 10s`, `IdleTimeout: 120s`, **sin** `ReadTimeout` ni `WriteTimeout`. Watchdog de subida: 60 s sin bytes.
- Buffer de copia en ZIP y subida: 256 KB (`io.CopyBuffer`).
- Token de sesión: 5 caracteres de `abcdefghjkmnpqrstuvwxyz23456789`, `crypto/rand`. PIN: 4 dígitos, `crypto/rand`. Token de control `t`: 32 bytes aleatorios en hex.
- Regla de inactividad: 120 s sin clientes SSE **y** sin transferencias en curso → salir. Re-evaluación de interfaces cada 5 s. Hint "¿No pueden entrar?" a los 45 s sin requests no-loopback a `/s/<token>`.
- QR: corrección `M`, quiet zone de 4 módulos, mínimo 280 px CSS, máximo 480 px.
- Cuarentena: macOS `~/Downloads/Pasame`, Windows `%USERPROFILE%\Downloads\Pasame`, Linux `$XDG_DOWNLOAD_DIR/Pasame` → `~/Descargas/Pasame` → `~/Downloads/Pasame` → `~/Pasame`. Config: `os.UserConfigDir()` + `/Pasame`.
- **Voz de la UI:** rioplatense, voseo, sin las palabras "IP", "puerto", "servidor", "túnel", "firewall", "mDNS", "token" en textos visibles. Todos los strings del receptor en `internal/i18n`; los del emisor en el objeto `T` de `web/control/app.js`.
- Página del receptor: funciona **sin JavaScript**; JS ES5 opcional (`XMLHttpRequest`, sin `fetch`, sin módulos); CSS sin `grid` obligatorio; página total < 30 KB.
- Reglas de dependencia (spec §8): `session`, `files`, `qr`, `i18n` no importan `net/http` ni `os/exec`. `share` no importa `control`, `core`, `addr`, `tunnel`. `addr` no importa `control` ni `share`. `control` solo importa `core` (y stdlib/`web`). Se verifican en la Tarea 20.
- Commits: mensajes en español, estilo `feat(paquete): ...`, terminando con la línea `Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>`.
- Tests: `go test -race ./...` debe pasar al final de cada tarea.

---

## Mapa de archivos de este plan

| Archivo | Responsabilidad | Tarea |
|---|---|---|
| `go.mod`, `Makefile`, `.gitignore`, `LICENSE`, `.github/workflows/ci.yml` | Módulo, comandos, CI con los 6 targets | 1 |
| `internal/files/sanitize.go` | Nombre de archivo recibido → nombre seguro | 1 |
| `internal/files/quarantine.go` | Carpeta de cuarentena por OS, ruta sin colisión, marca "descargado de internet" | 2 |
| `internal/session/token.go` | Token de 5 caracteres y PIN | 3 |
| `internal/session/walk.go` | Snapshot de rutas/carpetas → `[]File` | 3 |
| `internal/session/session.go` | `Session`, `File`, `New(paths)` | 3 |
| `internal/session/stats.go` | Contadores concurrentes | 4 |
| `internal/i18n/i18n.go`, `es.go`, `en.go` | Strings del receptor | 5 |
| `internal/share/server.go`, `listen.go` | Mux, `/`, `/healthz`, 410, fallback de puertos | 6 |
| `internal/share/page.go`, `web/share/*` | Página del receptor | 7 |
| `internal/share/file.go` | Descarga individual con Range | 8 |
| `internal/share/zip.go` | ZIP al vuelo | 9 |
| `internal/share/upload.go` | Dirección inversa | 10 |
| `internal/addr/book.go` | `Address`, `Provider`, `Book` | 11 |
| `internal/addr/lan.go` | Elección de interfaz | 12 |
| `internal/qr/qr.go` | URL → SVG | 13 |
| `internal/platform/*` | Config dir, abrir carpeta, firewall | 14 |
| `internal/dialog/dialog.go`, `internal/browser/open.go` | Diálogo nativo, abrir navegador | 15 |
| `internal/core/*` | Orquestación, `State`, regla de inactividad | 16 |
| `internal/control/*` | API local + SSE | 17 |
| `web/control/*` | UI del emisor | 18 |
| `internal/lifecycle/lock*.go` | Instancia única + handoff | 19 |
| `cmd/pasame/main.go`, `internal/deps_test.go` | Cableado, señales, reglas de dependencia | 20 |
| `test/e2e/*` | Playwright de la página del receptor | 21 |
| `test/manual/matriz.md` | Checklist manual del núcleo LAN | 22 |

**Embebido:** `web/embed.go` (paquete `web`) expone `web.Control` y `web.Share` como `embed.FS`. Se crea en la Tarea 7 y se amplía en la 18.

---

### Task 1: Andamiaje del módulo + saneamiento de nombres

Spec: §5 (fila "Path traversal en subidas"), §8, §13 (flags de build, targets).

**Files:**
- Create: `go.mod`, `Makefile`, `.gitignore`, `LICENSE`, `.github/workflows/ci.yml`
- Create: `internal/files/sanitize.go`
- Test: `internal/files/sanitize_test.go`

**Interfaces:**
- Consumes: nada.
- Produces: `files.Sanitize(name string) (string, error)`; `files.ErrBadName` (error centinela).

- [ ] **Step 1: Instalar Go y crear el módulo**

```bash
brew install go            # debe quedar 1.27.x
go version                 # Expected: go version go1.27.x darwin/arm64
cd pasame   # la carpeta del proyecto
go mod init github.com/KonixDev/pasame
go mod edit -go=1.27
```

- [ ] **Step 2: Crear `.gitignore`, `LICENSE` (MIT, "Martin Coll", 2026) y `Makefile`**

`.gitignore`:
```
/dist/
/pasame
/pasame.exe
/test/e2e/node_modules/
/test/e2e/test-results/
*.part
```

`Makefile`:
```make
VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)
TARGETS := windows/amd64 windows/arm64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 linux/arm

.PHONY: test build cross run
test:
	CGO_ENABLED=1 go test -race ./...
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o pasame ./cmd/pasame
cross:
	@for t in $(TARGETS); do \
	  os=$${t%/*}; arch=$${t#*/}; \
	  echo "==> $$os/$$arch"; \
	  CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch GOARM=7 go build -trimpath -ldflags="$(LDFLAGS)" -o /dev/null ./... || exit 1; \
	done
run: build
	./pasame
```
(`-race` necesita CGO en el binario de test; el binario distribuido no. Es la única excepción a `CGO_ENABLED=0` y solo afecta a `go test`.)

- [ ] **Step 3: Escribir el test de saneamiento (falla)**

`internal/files/sanitize_test.go`:
```go
package files

import (
	"errors"
	"strings"
	"testing"
)

func TestSanitize(t *testing.T) {
	cases := []struct {
		in, want string
		err      bool
	}{
		{"foto.jpg", "foto.jpg", false},
		{"../../etc/passwd", "passwd", false},
		{`..\..\Windows\system.ini`, "system.ini", false},
		{"C:\\Users\\x\\a.txt", "a.txt", false},
		{"/abs/path/b.pdf", "b.pdf", false},
		{"foto.jpg\x00.exe", "foto.jpg.exe", false},
		{"a\x01b\x1fc.txt", "abc.txt", false},
		{`we*i?rd"<na>me|.txt`, "weirdname.txt", false},
		{"dos:puntos.txt", "dospuntos.txt", false},
		{"año ñandú 日本.jpg", "año ñandú 日本.jpg", false},
		{"  espacios  .txt  ", "espacios  .txt", false},
		{"trailing.dots...", "trailing.dots", false},
		{"CON", "", true},
		{"con.txt", "", true},
		{"NUL.jpg", "", true},
		{"COM1", "", true},
		{"lpt9.log", "", true},
		{"CONSOLE.txt", "CONSOLE.txt", false},
		{"", "", true},
		{".", "", true},
		{"..", "", true},
		{"...", "", true},
		{"/", "", true},
		{"\x00\x01", "", true},
		{".bashrc", ".bashrc", false},
	}
	for _, c := range cases {
		got, err := Sanitize(c.in)
		if c.err {
			if !errors.Is(err, ErrBadName) {
				t.Errorf("Sanitize(%q) err = %v, want ErrBadName", c.in, err)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("Sanitize(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}

func TestSanitizeTruncatesKeepingExtension(t *testing.T) {
	in := strings.Repeat("a", 300) + ".mp4"
	got, err := Sanitize(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) > 200 || !strings.HasSuffix(got, ".mp4") {
		t.Fatalf("len=%d suffix ok=%v", len(got), strings.HasSuffix(got, ".mp4"))
	}
}

func TestSanitizeTruncatesOnRuneBoundary(t *testing.T) {
	in := strings.Repeat("ñ", 150) + ".txt" // 300 bytes + 4
	got, err := Sanitize(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, ".txt") || len(got) > 200 || !utf8Valid(got) {
		t.Fatalf("got %q (len %d)", got, len(got))
	}
}

func utf8Valid(s string) bool { return strings.ToValidUTF8(s, "\uFFFD") == s }
```

- [ ] **Step 4: Correr y verificar que falla**

Run: `go test ./internal/files/`
Expected: FAIL — `undefined: Sanitize`, `undefined: ErrBadName`.

- [ ] **Step 5: Implementar**

`internal/files/sanitize.go`:
```go
// Package files maneja nombres y rutas de archivos recibidos: nada de red, nada de exec.
package files

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// ErrBadName indica que el nombre no se puede usar ni saneado.
var ErrBadName = errors.New("files: nombre inválido")

const maxNameBytes = 200

var reserved = map[string]bool{"CON": true, "PRN": true, "AUX": true, "NUL": true}

func init() {
	for i := '1'; i <= '9'; i++ {
		reserved["COM"+string(i)] = true
		reserved["LPT"+string(i)] = true
	}
}

// Sanitize convierte un nombre que viene de un navegador en uno seguro para
// cualquier sistema de archivos. Nunca devuelve algo con separadores de ruta.
func Sanitize(name string) (string, error) {
	// Último componente, tratando / y \ como separadores en cualquier OS.
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	var b strings.Builder
	for _, r := range name {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(`:*?"<>|`, r) {
			continue
		}
		b.WriteRune(r)
	}
	s := strings.TrimSpace(b.String())
	s = strings.TrimRight(s, ". ")
	if s == "" || strings.Trim(s, ".") == "" {
		return "", ErrBadName
	}
	stem := strings.ToUpper(strings.TrimSuffix(s, filepath.Ext(s)))
	if reserved[stem] {
		return "", ErrBadName
	}
	return truncate(s), nil
}

func truncate(s string) string {
	if len(s) <= maxNameBytes {
		return s
	}
	ext := filepath.Ext(s)
	if len(ext) > 20 {
		ext = ""
	}
	stem := s[:len(s)-len(ext)]
	limit := maxNameBytes - len(ext)
	for limit > 0 && !utf8.RuneStart(stem[limit]) {
		limit--
	}
	return stem[:limit] + ext
}
```

- [ ] **Step 6: Correr y verificar que pasa**

Run: `go test -race ./internal/files/`
Expected: `ok  github.com/KonixDev/pasame/internal/files`

- [ ] **Step 7: CI**

`.github/workflows/ci.yml`:
```yaml
name: ci
on: [push, pull_request]
jobs:
  test:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix: { os: [ubuntu-latest, macos-latest, windows-latest] }
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.27.x' }
      - run: go vet ./...
      - run: go test -race ./...
  cross:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.27.x' }
      - run: make cross
```
Run local: `make cross`
Expected: 7 líneas `==> os/arch` sin errores.

- [ ] **Step 8: Commit**

```bash
git add go.mod Makefile .gitignore LICENSE .github internal/files
git commit -m "feat(files): andamiaje del módulo y saneamiento de nombres recibidos

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Carpeta de cuarentena, rutas sin colisión y marca de "descargado"

Spec: §5 (filas "Path traversal en subidas" y "Archivos ejecutables recibidos"), §13 (carpeta de cuarentena).

**Files:**
- Create: `internal/files/quarantine.go`, `internal/files/mark_darwin.go`, `internal/files/mark_windows.go`, `internal/files/mark_other.go`
- Test: `internal/files/quarantine_test.go`

**Interfaces:**
- Consumes: `Sanitize` (Tarea 1).
- Produces:
  - `files.QuarantineDir() (string, error)` — crea la carpeta si no existe.
  - `files.UniquePath(dir, name string) (string, error)` — sanea, resuelve colisión `x (2).jpg`, garantiza `filepath.Dir(result) == dir`.
  - `files.MarkDownloaded(path string) error` — xattr `com.apple.quarantine` (macOS), ADS `Zone.Identifier` con `ZoneId=3` (Windows), no-op en Linux.

- [ ] **Step 1: Test que falla**

`internal/files/quarantine_test.go`:
```go
package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUniquePath(t *testing.T) {
	dir := t.TempDir()
	p, err := UniquePath(dir, "x.jpg")
	if err != nil || p != filepath.Join(dir, "x.jpg") {
		t.Fatalf("got %q %v", p, err)
	}
	os.WriteFile(p, nil, 0o644)
	p2, _ := UniquePath(dir, "x.jpg")
	if filepath.Base(p2) != "x (2).jpg" {
		t.Fatalf("got %q", p2)
	}
	os.WriteFile(p2, nil, 0o644)
	p3, _ := UniquePath(dir, "x.jpg")
	if filepath.Base(p3) != "x (3).jpg" {
		t.Fatalf("got %q", p3)
	}
	// Un .part en curso también cuenta como ocupado.
	os.WriteFile(filepath.Join(dir, "y.txt.part"), nil, 0o644)
	p4, _ := UniquePath(dir, "y.txt")
	if filepath.Base(p4) != "y (2).txt" {
		t.Fatalf("got %q", p4)
	}
}

func TestUniquePathNeverEscapes(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"../../evil", `..\..\evil`, "a/../../b"} {
		p, err := UniquePath(dir, n)
		if err != nil {
			continue
		}
		if filepath.Dir(p) != dir {
			t.Fatalf("%q escapó a %q", n, p)
		}
	}
	if _, err := UniquePath(dir, ".."); err == nil {
		t.Fatal("'..' debería fallar")
	}
}

func TestQuarantineDirIsCreated(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_DOWNLOAD_DIR", "")
	d, err := QuarantineDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(d) != "Pasame" {
		t.Fatalf("got %q", d)
	}
	if fi, err := os.Stat(d); err != nil || !fi.IsDir() {
		t.Fatalf("no se creó: %v", err)
	}
}

func TestMarkDownloadedDoesNotFail(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.exe")
	os.WriteFile(p, []byte("x"), 0o644)
	if err := MarkDownloaded(p); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Verificar que falla**

Run: `go test ./internal/files/`
Expected: FAIL — `undefined: UniquePath`, `QuarantineDir`, `MarkDownloaded`.

- [ ] **Step 3: Implementar `quarantine.go`**

```go
package files

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// QuarantineDir devuelve (y crea) la carpeta donde caen los archivos recibidos.
func QuarantineDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	base := filepath.Join(home, "Downloads")
	if runtime.GOOS == "linux" {
		base = linuxDownloads(home)
	}
	dir := filepath.Join(base, "Pasame")
	return dir, os.MkdirAll(dir, 0o755)
}

func linuxDownloads(home string) string {
	if x := os.Getenv("XDG_DOWNLOAD_DIR"); x != "" {
		return x
	}
	for _, n := range []string{"Descargas", "Downloads"} {
		if fi, err := os.Stat(filepath.Join(home, n)); err == nil && fi.IsDir() {
			return filepath.Join(home, n)
		}
	}
	return home
}

// UniquePath sanea name y devuelve una ruta libre dentro de dir.
// Una ruta está ocupada si existe el archivo o su ".part".
func UniquePath(dir, name string) (string, error) {
	clean, err := Sanitize(name)
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(clean)
	stem := strings.TrimSuffix(clean, ext)
	for i := 1; i < 10000; i++ {
		n := clean
		if i > 1 {
			n = fmt.Sprintf("%s (%d)%s", stem, i, ext)
		}
		p := filepath.Join(dir, n)
		if filepath.Dir(p) != filepath.Clean(dir) {
			return "", ErrBadName
		}
		if !exists(p) && !exists(p+".part") {
			return p, nil
		}
	}
	return "", errors.New("files: demasiadas colisiones")
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return !errors.Is(err, fs.ErrNotExist)
}
```

`mark_darwin.go`:
```go
//go:build darwin

package files

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

// MarkDownloaded hace que Gatekeeper trate el archivo como bajado de internet.
func MarkDownloaded(path string) error {
	v := fmt.Sprintf("0081;%08x;Pasame;", time.Now().Unix())
	return unix.Setxattr(path, "com.apple.quarantine", []byte(v), 0)
}
```

`mark_windows.go`:
```go
//go:build windows

package files

import "os"

// MarkDownloaded escribe el Zone.Identifier que usan los navegadores (ZoneId=3 = internet).
func MarkDownloaded(path string) error {
	return os.WriteFile(path+":Zone.Identifier", []byte("[ZoneTransfer]\r\nZoneId=3\r\n"), 0o644)
}
```

`mark_other.go`:
```go
//go:build !darwin && !windows

package files

// MarkDownloaded no hace nada en Linux: no hay un equivalente estándar.
func MarkDownloaded(path string) error { return nil }
```

Run: `go get golang.org/x/sys@latest && go mod tidy`

- [ ] **Step 4: Verificar que pasa**

Run: `go test -race ./internal/files/ && make cross`
Expected: `ok` y los 7 targets compilan (los build tags cubren darwin/windows/linux).

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/files
git commit -m "feat(files): carpeta de cuarentena, rutas sin colisión y marca de descarga

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Sesión — token, PIN y snapshot de archivos

Spec: §4.3.

**Files:**
- Create: `internal/session/token.go`, `internal/session/walk.go`, `internal/session/session.go`
- Test: `internal/session/token_test.go`, `internal/session/session_test.go`

**Interfaces:**
- Consumes: nada.
- Produces:
  ```go
  const Alphabet = "abcdefghjkmnpqrstuvwxyz23456789"
  func NewToken() string   // 5 chars
  func NewPIN() string     // "0000".."9999"
  type File struct { Index int; Name, Rel, Abs string; Size int64; ModTime time.Time }
  type Session struct { Token, PIN string; Files []File; CreatedAt time.Time }
  func New(paths []string) (*Session, []error) // errores por ruta ilegible; nunca nil *Session
  func (s *Session) File(i int) (File, bool)
  func (s *Session) TotalSize() int64
  func (s *Session) Path() string              // "/s/<token>"
  ```
  Ni `Strict` ni la clave HMAC del PIN viven en `Session`: dependen del túnel, cambian durante la sesión y son estado de `core` (Plan 2). La sesión queda inmutable.

- [ ] **Step 1: Tests que fallan**

`internal/session/token_test.go`:
```go
package session

import (
	"strings"
	"testing"
)

func TestNewToken(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 2000; i++ {
		tok := NewToken()
		if len(tok) != 5 {
			t.Fatalf("len(%q) = %d", tok, len(tok))
		}
		for _, r := range tok {
			if !strings.ContainsRune(Alphabet, r) {
				t.Fatalf("%q tiene %q fuera del alfabeto", tok, r)
			}
		}
		seen[tok] = true
	}
	if len(seen) < 1990 {
		t.Fatalf("solo %d tokens distintos de 2000", len(seen))
	}
}

func TestNewPIN(t *testing.T) {
	for i := 0; i < 500; i++ {
		p := NewPIN()
		if len(p) != 4 || strings.Trim(p, "0123456789") != "" {
			t.Fatalf("PIN inválido %q", p)
		}
	}
}
```

`internal/session/session_test.go`:
```go
package session

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func write(t *testing.T, p string, size int) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNewWithFilesAndFolder(t *testing.T) {
	d := t.TempDir()
	write(t, filepath.Join(d, "a.jpg"), 10)
	write(t, filepath.Join(d, "album", "b.jpg"), 20)
	write(t, filepath.Join(d, "album", "sub", "c.mp4"), 30)
	write(t, filepath.Join(d, "album", ".DS_Store"), 1)
	write(t, filepath.Join(d, "album", "Thumbs.db"), 1)
	write(t, filepath.Join(d, "album", "desktop.ini"), 1)
	write(t, filepath.Join(d, "album", ".oculta", "x.txt"), 1)
	if runtime.GOOS != "windows" {
		os.Symlink(filepath.Join(d, "a.jpg"), filepath.Join(d, "album", "link.jpg"))
	}

	s, errs := New([]string{filepath.Join(d, "a.jpg"), filepath.Join(d, "album")})
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	var rels []string
	for i, f := range s.Files {
		if f.Index != i {
			t.Fatalf("Index %d en posición %d", f.Index, i)
		}
		rels = append(rels, f.Rel)
	}
	want := []string{"a.jpg", "album/b.jpg", "album/sub/c.mp4"}
	if len(rels) != len(want) {
		t.Fatalf("rels = %v, want %v", rels, want)
	}
	for i := range want {
		if rels[i] != want[i] {
			t.Fatalf("rels = %v, want %v", rels, want)
		}
	}
	if s.Files[2].Name != "c.mp4" || s.TotalSize() != 60 {
		t.Fatalf("Name=%q total=%d", s.Files[2].Name, s.TotalSize())
	}
	if s.Path() != "/s/"+s.Token || len(s.PIN) != 4 {
		t.Fatalf("Path=%q PIN=%q", s.Path(), s.PIN)
	}
}

func TestNewReportsUnreadable(t *testing.T) {
	d := t.TempDir()
	write(t, filepath.Join(d, "ok.txt"), 1)
	s, errs := New([]string{filepath.Join(d, "ok.txt"), filepath.Join(d, "no-existe.mp4")})
	if len(s.Files) != 1 || len(errs) != 1 {
		t.Fatalf("files=%d errs=%v", len(s.Files), errs)
	}
}

func TestNewEmptyIsReceiveOnly(t *testing.T) {
	s, errs := New(nil)
	if s == nil || len(s.Files) != 0 || len(errs) != 0 {
		t.Fatalf("s=%v errs=%v", s, errs)
	}
}

func TestFileLookup(t *testing.T) {
	d := t.TempDir()
	write(t, filepath.Join(d, "a"), 1)
	s, _ := New([]string{filepath.Join(d, "a")})
	if _, ok := s.File(0); !ok {
		t.Fatal("File(0) debería existir")
	}
	for _, i := range []int{-1, 1, 99} {
		if _, ok := s.File(i); ok {
			t.Fatalf("File(%d) no debería existir", i)
		}
	}
}
```

- [ ] **Step 2: Verificar que falla**

Run: `go test ./internal/session/`
Expected: FAIL — `undefined: NewToken`, `New`, etc.

- [ ] **Step 3: Implementar**

`internal/session/token.go`:
```go
package session

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// Alphabet omite 0/o/1/l/i para que el token se pueda dictar.
const Alphabet = "abcdefghjkmnpqrstuvwxyz23456789"

func randInt(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err) // crypto/rand no falla en los OS soportados
	}
	return int(v.Int64())
}

func NewToken() string {
	b := make([]byte, 5)
	for i := range b {
		b[i] = Alphabet[randInt(len(Alphabet))]
	}
	return string(b)
}

func NewPIN() string { return fmt.Sprintf("%04d", randInt(10000)) }
```

`internal/session/walk.go`:
```go
package session

import (
	"io/fs"
	"path/filepath"
	"strings"
)

var junk = map[string]bool{"thumbs.db": true, "desktop.ini": true}

func hidden(name string) bool {
	return strings.HasPrefix(name, ".") || junk[strings.ToLower(name)]
}

// walk recorre root una sola vez. No sigue symlinks. Rel usa "/" siempre (va al ZIP).
func walk(root string) ([]File, error) {
	parent := filepath.Dir(root)
	var out []File
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // un subdirectorio ilegible no arruina el resto
		}
		if p != root && hidden(d.Name()) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil // directorios, symlinks, sockets
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(parent, p)
		out = append(out, File{
			Name: d.Name(), Rel: filepath.ToSlash(rel), Abs: p,
			Size: info.Size(), ModTime: info.ModTime(),
		})
		return nil
	})
	return out, err
}
```

`internal/session/session.go`:
```go
// Package session modela un envío: qué archivos, con qué token y PIN. Sin red, sin exec.
package session

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type File struct {
	Index   int
	Name    string // nombre para mostrar y para Content-Disposition
	Rel     string // ruta dentro del ZIP, con "/"
	Abs     string // ruta en disco; nunca sale del proceso
	Size    int64
	ModTime time.Time
}

type Session struct {
	Token     string
	PIN       string
	Files     []File // snapshot inmutable
	CreatedAt time.Time
}

// New arma una sesión. Con paths vacío es el modo "Solo recibir".
// Las rutas que no se pueden leer vuelven como errores y se omiten.
func New(paths []string) (*Session, []error) {
	s := &Session{Token: NewToken(), PIN: NewPIN(), CreatedAt: time.Now()}
	var errs []error
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		info, err := os.Stat(abs)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", filepath.Base(p), err))
			continue
		}
		if info.IsDir() {
			fs, _ := walk(abs)
			s.Files = append(s.Files, fs...)
			continue
		}
		s.Files = append(s.Files, File{
			Name: info.Name(), Rel: info.Name(), Abs: abs,
			Size: info.Size(), ModTime: info.ModTime(),
		})
	}
	for i := range s.Files {
		s.Files[i].Index = i
	}
	return s, errs
}

func (s *Session) File(i int) (File, bool) {
	if i < 0 || i >= len(s.Files) {
		return File{}, false
	}
	return s.Files[i], true
}

func (s *Session) TotalSize() int64 {
	var n int64
	for _, f := range s.Files {
		n += f.Size
	}
	return n
}

func (s *Session) Path() string { return "/s/" + s.Token }
```

- [ ] **Step 4: Verificar que pasa**

Run: `go test -race ./internal/session/`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/session
git commit -m "feat(session): token, PIN y snapshot inmutable de archivos y carpetas

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Estadísticas concurrentes

Spec: §4.3 (`stats`), §6.2 (Actividad), §13 (hint de 45 s: requests desde IP no-loopback).

**Files:**
- Create: `internal/session/stats.go`
- Test: `internal/session/stats_test.go`

**Interfaces:**
- Consumes: nada.
- Produces:
  ```go
  type Stats struct{ /* privado, seguro para concurrencia */ }
  func NewStats(onChange func()) *Stats      // onChange puede ser nil; se llama tras cada cambio
  func (s *Stats) ClientSeen(ip string)      // ignora loopback
  func (s *Stats) DownloadStarted(i int)
  func (s *Stats) DownloadDone(i int, ok bool)
  func (s *Stats) UploadStarted(name string)
  func (s *Stats) UploadDone(name string, ok bool)
  func (s *Stats) InFlight() int             // descargas + subidas en curso
  func (s *Stats) Snapshot() Snapshot
  type Snapshot struct {
      Clients   int            `json:"clients"`
      Active    map[int]int    `json:"active"`    // índice → descargas en curso
      Completed map[int]int    `json:"completed"` // índice → descargas completas
      Uploading []string       `json:"uploading"`
      Received  []string       `json:"received"`  // nombres finales, en orden de llegada
      FirstSeen time.Time      `json:"firstSeen"` // cero si nadie entró
  }
  ```
  Un `Stats` nuevo por sesión.

- [ ] **Step 1: Test que falla**

`internal/session/stats_test.go`:
```go
package session

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestStatsCountsAndIgnoresLoopback(t *testing.T) {
	var calls atomic.Int32
	s := NewStats(func() { calls.Add(1) })
	s.ClientSeen("127.0.0.1")
	s.ClientSeen("::1")
	if snap := s.Snapshot(); snap.Clients != 0 || !snap.FirstSeen.IsZero() {
		t.Fatalf("loopback contó: %+v", snap)
	}
	s.ClientSeen("192.168.1.50")
	s.ClientSeen("192.168.1.50")
	s.ClientSeen("192.168.1.51")
	s.DownloadStarted(0)
	s.DownloadStarted(0)
	s.DownloadDone(0, true)
	s.UploadStarted("IMG_1.jpg")
	if s.InFlight() != 2 {
		t.Fatalf("InFlight = %d", s.InFlight())
	}
	s.UploadDone("IMG_1.jpg", true)
	snap := s.Snapshot()
	if snap.Clients != 2 || snap.Active[0] != 1 || snap.Completed[0] != 1 ||
		len(snap.Received) != 1 || len(snap.Uploading) != 0 || snap.FirstSeen.IsZero() {
		t.Fatalf("snap = %+v", snap)
	}
	if calls.Load() == 0 {
		t.Fatal("onChange nunca se llamó")
	}
}

func TestStatsFailedDownloadNotCompleted(t *testing.T) {
	s := NewStats(nil)
	s.DownloadStarted(1)
	s.DownloadDone(1, false)
	if snap := s.Snapshot(); snap.Completed[1] != 0 || snap.Active[1] != 0 {
		t.Fatalf("snap = %+v", snap)
	}
}

func TestStatsConcurrent(t *testing.T) {
	s := NewStats(func() {})
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.ClientSeen("10.0.0.2")
			s.DownloadStarted(0)
			s.DownloadDone(0, true)
			_ = s.Snapshot()
		}()
	}
	wg.Wait()
	if s.Snapshot().Completed[0] != 100 || s.InFlight() != 0 {
		t.Fatal("conteo concurrente incorrecto")
	}
}

func TestSnapshotIsACopy(t *testing.T) {
	s := NewStats(nil)
	s.DownloadStarted(0)
	snap := s.Snapshot()
	snap.Active[0] = 99
	if s.Snapshot().Active[0] != 1 {
		t.Fatal("Snapshot comparte el mapa interno")
	}
}
```

- [ ] **Step 2: Verificar que falla**

Run: `go test ./internal/session/ -run Stats`
Expected: FAIL — `undefined: NewStats`.

- [ ] **Step 3: Implementar**

`internal/session/stats.go`:
```go
package session

import (
	"net"
	"sync"
	"time"
)

type Snapshot struct {
	Clients   int         `json:"clients"`
	Active    map[int]int `json:"active"`
	Completed map[int]int `json:"completed"`
	Uploading []string    `json:"uploading"`
	Received  []string    `json:"received"`
	FirstSeen time.Time   `json:"firstSeen"`
}

type Stats struct {
	mu        sync.Mutex
	onChange  func()
	clients   map[string]bool
	active    map[int]int
	completed map[int]int
	uploading map[string]int
	received  []string
	firstSeen time.Time
}

func NewStats(onChange func()) *Stats {
	return &Stats{
		onChange: onChange, clients: map[string]bool{},
		active: map[int]int{}, completed: map[int]int{}, uploading: map[string]int{},
	}
}

func (s *Stats) change(f func()) {
	s.mu.Lock()
	f()
	s.mu.Unlock()
	if s.onChange != nil {
		s.onChange()
	}
}

func (s *Stats) ClientSeen(ip string) {
	if p := net.ParseIP(ip); p == nil || p.IsLoopback() {
		return
	}
	s.mu.Lock()
	known := s.clients[ip]
	s.mu.Unlock()
	if known {
		return
	}
	s.change(func() {
		s.clients[ip] = true
		if s.firstSeen.IsZero() {
			s.firstSeen = time.Now()
		}
	})
}

func (s *Stats) DownloadStarted(i int) { s.change(func() { s.active[i]++ }) }

func (s *Stats) DownloadDone(i int, ok bool) {
	s.change(func() {
		if s.active[i]--; s.active[i] <= 0 {
			delete(s.active, i)
		}
		if ok {
			s.completed[i]++
		}
	})
}

func (s *Stats) UploadStarted(name string) { s.change(func() { s.uploading[name]++ }) }

func (s *Stats) UploadDone(name string, ok bool) {
	s.change(func() {
		if s.uploading[name]--; s.uploading[name] <= 0 {
			delete(s.uploading, name)
		}
		if ok {
			s.received = append(s.received, name)
		}
	})
}

func (s *Stats) InFlight() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, v := range s.active {
		n += v
	}
	for _, v := range s.uploading {
		n += v
	}
	return n
}

func (s *Stats) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := Snapshot{
		Clients: len(s.clients), Active: map[int]int{}, Completed: map[int]int{},
		Received: append([]string(nil), s.received...), FirstSeen: s.firstSeen,
	}
	for k, v := range s.active {
		snap.Active[k] = v
	}
	for k, v := range s.completed {
		snap.Completed[k] = v
	}
	for k := range s.uploading {
		snap.Uploading = append(snap.Uploading, k)
	}
	return snap
}
```

- [ ] **Step 4: Verificar que pasa**

Run: `go test -race ./internal/session/`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/session/stats.go internal/session/stats_test.go
git commit -m "feat(session): estadísticas concurrentes de clientes, descargas y subidas

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Textos del receptor (es/en) y tamaños legibles

Spec: §4.4 (i18n por `Accept-Language`), §6.4–6.5 (microcopy), §9.1 (fila `i18n`), §11 (solo es/en).

**Files:**
- Create: `internal/i18n/i18n.go`, `internal/i18n/es.go`, `internal/i18n/en.go`
- Test: `internal/i18n/i18n_test.go`

**Interfaces:**
- Consumes: nada.
- Produces:
  ```go
  type Lang string
  const ES Lang = "es"; const EN Lang = "en"
  func Pick(acceptLanguage string) Lang                  // es por defecto
  func T(l Lang, key string, args ...any) string         // fmt.Sprintf; clave inexistente → panic en tests, "[key]" en prod
  func Size(l Lang, n int64) string                      // "1,2 GB" (es) / "1.2 GB" (en)
  ```
  Claves (todas las usa la Tarea 7 o el Plan 2): `page_title`, `sharing_1`, `sharing_n`, `receive_only`, `download_all`, `view`, `send_back`, `choose_send`, `send`, `footer_lan`, `footer_tunnel`, `gone_title`, `gone_body`, `sent_1`, `sent_n`, `sent_done_js`, `sending_js`, `err_no_space`, `err_empty`, `err_generic`, `file_gone`, `pin_prompt`, `pin_wrong`, `pin_locked`, `pin_enter`, `root_strict`.

- [ ] **Step 1: Test que falla**

`internal/i18n/i18n_test.go`:
```go
package i18n

import "testing"

func TestSameKeys(t *testing.T) {
	for k := range es {
		if _, ok := en[k]; !ok {
			t.Errorf("falta %q en en", k)
		}
	}
	for k := range en {
		if _, ok := es[k]; !ok {
			t.Errorf("falta %q en es", k)
		}
	}
}

func TestPick(t *testing.T) {
	cases := map[string]Lang{
		"":                              ES,
		"es-AR,es;q=0.9":                ES,
		"en-US,en;q=0.9":                EN,
		"en-GB":                         EN,
		"pt-BR,pt;q=0.9":                ES,
		"fr-FR,en;q=0.8,es;q=0.9":       ES,
		"fr-FR,es;q=0.5,en;q=0.8":       EN,
		"*":                             ES,
	}
	for in, want := range cases {
		if got := Pick(in); got != want {
			t.Errorf("Pick(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestT(t *testing.T) {
	if got := T(ES, "sharing_n", "Martín", 3); got != "Martín te comparte 3 archivos" {
		t.Fatalf("got %q", got)
	}
	if got := T(EN, "sharing_n", "Martín", 3); got != "Martín is sharing 3 files with you" {
		t.Fatalf("got %q", got)
	}
}

func TestSize(t *testing.T) {
	cases := []struct {
		l    Lang
		n    int64
		want string
	}{
		{ES, 0, "0 B"}, {ES, 999, "999 B"}, {ES, 1300, "1,3 KB"},
		{ES, 4_200_000, "4,2 MB"}, {ES, 980_000_000, "980 MB"},
		{ES, 1_200_000_000, "1,2 GB"}, {EN, 1_200_000_000, "1.2 GB"},
	}
	for _, c := range cases {
		if got := Size(c.l, c.n); got != c.want {
			t.Errorf("Size(%s,%d) = %q, want %q", c.l, c.n, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Verificar que falla**

Run: `go test ./internal/i18n/`
Expected: FAIL — `undefined: es`, `Pick`, …

- [ ] **Step 3: Implementar**

`internal/i18n/es.go`:
```go
package i18n

var es = map[string]string{
	"page_title":    "Pasame",
	"sharing_1":     "%s te comparte 1 archivo",
	"sharing_n":     "%s te comparte %d archivos",
	"receive_only":  "Mandale archivos a %s",
	"download_all":  "Descargar todo (%s)",
	"view":          "Ver",
	"send_back":     "¿Querés mandarle algo a %s?",
	"choose_send":   "Elegir archivos para enviar",
	"send":          "Enviar",
	"footer_lan":    "Los archivos viajan solo por esta red.",
	"footer_tunnel": "Los archivos viajan cifrados por internet.",
	"gone_title":    "Este envío ya terminó",
	"gone_body":     "Pedile un código nuevo a quien te lo compartió.",
	"sent_1":        "✓ Enviaste 1 archivo.",
	"sent_n":        "✓ Enviaste %d archivos.",
	"sent_done_js":  "✓ Listo. %s ya los tiene.",
	"sending_js":    "Enviando %d de %d · %d %%",
	"err_no_space":  "No se pudo guardar: la computadora de %s no tiene espacio.",
	"err_empty":     "Ese archivo está vacío.",
	"err_generic":   "No se pudo enviar. Probá de nuevo.",
	"file_gone":     "Este archivo ya no está disponible.",
	"pin_prompt":    "Escribí la clave de 4 números que aparece en la pantalla de %s.",
	"pin_wrong":     "La clave no es esa. Fijate bien en la pantalla de %s.",
	"pin_locked":    "Demasiados intentos. Esperá un minuto y probá de nuevo.",
	"pin_enter":     "Entrar",
	"root_strict":   "Para entrar necesitás el código QR o el link de quien comparte.",
}
```

`internal/i18n/en.go`:
```go
package i18n

var en = map[string]string{
	"page_title":    "Pasame",
	"sharing_1":     "%s is sharing 1 file with you",
	"sharing_n":     "%s is sharing %d files with you",
	"receive_only":  "Send files to %s",
	"download_all":  "Download all (%s)",
	"view":          "View",
	"send_back":     "Want to send something to %s?",
	"choose_send":   "Choose files to send",
	"send":          "Send",
	"footer_lan":    "Files only travel over this network.",
	"footer_tunnel": "Files travel encrypted over the internet.",
	"gone_title":    "This share has ended",
	"gone_body":     "Ask the person who shared it for a new code.",
	"sent_1":        "✓ Sent 1 file.",
	"sent_n":        "✓ Sent %d files.",
	"sent_done_js":  "✓ Done. %s has them now.",
	"sending_js":    "Sending %d of %d · %d %%",
	"err_no_space":  "Couldn't save: %s's computer is out of space.",
	"err_empty":     "That file is empty.",
	"err_generic":   "Couldn't send. Please try again.",
	"file_gone":     "This file is no longer available.",
	"pin_prompt":    "Type the 4-digit code shown on %s's screen.",
	"pin_wrong":     "That's not the code. Check %s's screen again.",
	"pin_locked":    "Too many attempts. Wait a minute and try again.",
	"pin_enter":     "Enter",
	"root_strict":   "You need the QR code or the link from the person sharing.",
}
```

`internal/i18n/i18n.go`:
```go
// Package i18n tiene todos los textos que ve el receptor. Sin red, sin exec.
package i18n

import (
	"fmt"
	"strconv"
	"strings"
)

type Lang string

const (
	ES Lang = "es"
	EN Lang = "en"
)

var tables = map[Lang]map[string]string{ES: es, EN: en}

// Pick elige entre es y en según Accept-Language; ante empate o nada conocido, es.
func Pick(accept string) Lang {
	best, bestQ := ES, -1.0
	for _, part := range strings.Split(accept, ",") {
		fields := strings.Split(strings.TrimSpace(part), ";")
		tag := strings.ToLower(strings.SplitN(fields[0], "-", 2)[0])
		q := 1.0
		for _, f := range fields[1:] {
			if v, ok := strings.CutPrefix(strings.TrimSpace(f), "q="); ok {
				q, _ = strconv.ParseFloat(v, 64)
			}
		}
		if l := Lang(tag); (l == ES || l == EN) && q > bestQ {
			best, bestQ = l, q
		}
	}
	return best
}

func T(l Lang, key string, args ...any) string {
	s, ok := tables[l][key]
	if !ok {
		if s, ok = es[key]; !ok {
			return "[" + key + "]"
		}
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

func Size(l Lang, n int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	f, u := float64(n), 0
	for f >= 1000 && u < len(units)-1 {
		f /= 1000
		u++
	}
	var s string
	switch {
	case u == 0:
		s = strconv.FormatInt(n, 10)
	case f >= 100:
		s = strconv.FormatFloat(f, 'f', 0, 64)
	default:
		s = strconv.FormatFloat(f, 'f', 1, 64)
	}
	if l == ES {
		s = strings.Replace(s, ".", ",", 1)
	}
	return s + " " + units[u]
}
```
(Unidades decimales, como las muestran macOS e iOS: coincide con lo que la persona ve en su teléfono.)

- [ ] **Step 4: Verificar que pasa**

Run: `go test -race ./internal/i18n/`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/i18n
git commit -m "feat(i18n): textos del receptor en español y en inglés

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Servidor `share` — esqueleto, `/`, `/healthz`, sesión vencida y puertos

Spec: §4.2 (puerto 8080 y fallback), §4.4 (tabla de rutas: `/`, `/healthz`, `/s/<otro-token>`), §13 (timeouts).

**Files:**
- Create: `web/embed.go`, `web/share/base.html`, `web/share/gone.html`, `web/share/page.css`
- Create: `internal/share/server.go`, `internal/share/listen.go`
- Test: `internal/share/server_test.go`, `internal/share/helpers_test.go`

**Interfaces:**
- Consumes: `session.Session`, `session.Stats` (Tareas 3–4), `i18n.Pick/T/Size` (Tarea 5).
- Produces:
  ```go
  // package web
  var Share embed.FS // contiene share/*.html, share/page.css, share/page.js

  // package share
  type Deps struct {
      Current    func() *session.Session // nil = no hay envío activo
      Stats      func() *session.Stats   // stats de la sesión actual; nunca nil si Current != nil
      Quarantine string                  // carpeta de subidas (Tarea 10)
      Sender     func() string           // "Martín"
      Strict     func() bool             // Plan 2; en este plan core devuelve false
  }
  type Server struct{ /* http.Handler */ }
  func New(d Deps) (*Server, error)
  func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request)
  func Listen(host string, preferred int) (net.Listener, error) // preferred..preferred+9, luego :0
  func HTTPServer(h http.Handler) *http.Server                   // con los timeouts del spec
  ```
  Handlers internos que las Tareas 7–10 completan: `s.page`, `s.file`, `s.zip`, `s.upload` (en este paso, stubs que devuelven 501).

- [ ] **Step 1: Plantillas y embebido**

`web/embed.go`:
```go
// Package web contiene los archivos estáticos embebidos en el binario.
package web

import "embed"

//go:embed share
var Share embed.FS
```

`web/share/base.html`:
```html
{{define "head"}}<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{t .Lang "page_title"}}</title>
<style>{{css}}</style>{{end}}
```

`web/share/gone.html`:
```html
{{define "gone.html"}}<!DOCTYPE html>
<html lang="{{.Lang}}"><head>{{template "head" .}}</head>
<body><main>
<p class="brand">Pasame</p>
<h1>{{t .Lang "gone_title"}}</h1>
<p>{{t .Lang "gone_body"}}</p>
</main></body></html>{{end}}
```

`web/share/page.css` (sin `grid`, sin variables, botones enormes; < 4 KB):
```css
body{margin:0;font-family:-apple-system,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;font-size:18px;line-height:1.4;color:#111;background:#fff}
main{max-width:640px;margin:0 auto;padding:16px}
.brand{font-weight:bold;color:#555;margin:0 0 16px}
h1{font-size:24px;margin:0 0 20px}
.big{display:block;box-sizing:border-box;width:100%;padding:20px 16px;margin:0 0 20px;font-size:20px;font-weight:bold;text-align:center;text-decoration:none;color:#fff;background:#0a58ca;border:0;border-radius:12px}
.big.alt{background:#1f7a3a}
ul{list-style:none;padding:0;margin:0 0 24px;border-top:1px solid #ddd}
li{border-bottom:1px solid #ddd;padding:14px 0;overflow:hidden}
li a.name{color:#0a58ca;word-break:break-word}
li .size{color:#666;font-size:15px;margin-left:8px}
li .view{float:right;margin-left:12px;padding:6px 14px;border:1px solid #0a58ca;border-radius:8px;text-decoration:none;color:#0a58ca}
.note{color:#555;font-size:15px}
.ok{background:#e7f6ec;padding:12px;border-radius:8px}
.err{background:#fdecea;padding:12px;border-radius:8px}
input[type=file]{font-size:18px;margin:0 0 12px;max-width:100%}
.bar{height:14px;background:#eee;border-radius:7px;overflow:hidden;margin:8px 0}
.bar div{height:100%;width:0;background:#1f7a3a}
.pin input{font-size:32px;width:5em;letter-spacing:.3em;text-align:center;padding:8px}
```

- [ ] **Step 2: Helpers de test y tests que fallan**

`internal/share/helpers_test.go`:
```go
package share

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/KonixDev/pasame/internal/session"
)

type fixture struct {
	srv   *Server
	sess  *session.Session
	stats *session.Stats
	dir   string // archivos compartidos
	quar  string // cuarentena
	strict bool
}

// newFixture crea archivos con los tamaños dados y una sesión que los comparte.
func newFixture(t *testing.T, sizes map[string]int) *fixture {
	t.Helper()
	f := &fixture{dir: t.TempDir(), quar: t.TempDir(), stats: session.NewStats(nil)}
	var paths []string
	for name, n := range sizes {
		p := filepath.Join(f.dir, name)
		if err := os.WriteFile(p, pattern(n), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	sortStrings(paths)
	f.sess, _ = session.New(paths)
	srv, err := New(Deps{
		Current:    func() *session.Session { return f.sess },
		Stats:      func() *session.Stats { return f.stats },
		Quarantine: f.quar,
		Sender:     func() string { return "Martín" },
		Strict:     func() bool { return f.strict },
	})
	if err != nil {
		t.Fatal(err)
	}
	f.srv = srv
	return f
}

func (f *fixture) do(r *http.Request) *httptest.ResponseRecorder {
	if r.RemoteAddr == "" || r.RemoteAddr == "192.0.2.1:1234" {
		r.RemoteAddr = "192.168.1.50:5555"
	}
	w := httptest.NewRecorder()
	f.srv.ServeHTTP(w, r)
	return w
}

// pattern genera bytes deterministas para comparar descargas.
func pattern(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i % 251)
	}
	return b
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
```

`internal/share/server_test.go`:
```go
package share

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthz(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	w := f.do(httptest.NewRequest("GET", "/healthz", nil))
	if w.Code != 200 || w.Body.String() != "ok" {
		t.Fatalf("%d %q", w.Code, w.Body)
	}
}

func TestRootRedirectsToSession(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	w := f.do(httptest.NewRequest("GET", "/", nil))
	if w.Code != http.StatusFound || w.Header().Get("Location") != f.sess.Path() {
		t.Fatalf("%d %q", w.Code, w.Header().Get("Location"))
	}
}

func TestRootStrictDoesNotRevealToken(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.strict = true
	w := f.do(httptest.NewRequest("GET", "/", nil))
	if w.Code != http.StatusNotFound || strings.Contains(w.Body.String(), f.sess.Token) {
		t.Fatalf("%d, body contiene token=%v", w.Code, strings.Contains(w.Body.String(), f.sess.Token))
	}
	if !strings.Contains(w.Body.String(), "código QR") {
		t.Fatalf("body: %s", w.Body)
	}
}

func TestOldTokenIsGone(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	for _, p := range []string{"/s/zzzzz", "/s/zzzzz/f/0", "/s/zzzzz/zip"} {
		w := f.do(httptest.NewRequest("GET", p, nil))
		if w.Code != http.StatusGone || !strings.Contains(w.Body.String(), "Este envío ya terminó") {
			t.Fatalf("%s → %d", p, w.Code)
		}
	}
}

func TestNoSessionIsGone(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.sess = nil
	if w := f.do(httptest.NewRequest("GET", "/", nil)); w.Code != http.StatusGone {
		t.Fatalf("/ sin sesión → %d", w.Code)
	}
}

func TestGoneInEnglish(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	r := httptest.NewRequest("GET", "/s/zzzzz", nil)
	r.Header.Set("Accept-Language", "en-US")
	if w := f.do(r); !strings.Contains(w.Body.String(), "This share has ended") {
		t.Fatalf("body: %s", w.Body)
	}
}

func TestClientSeenOnSessionRequest(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.do(httptest.NewRequest("GET", f.sess.Path(), nil))
	if f.stats.Snapshot().Clients != 1 {
		t.Fatal("no registró al cliente")
	}
}

func TestListenFallback(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	port := busy.Addr().(*net.TCPAddr).Port
	l, err := Listen("127.0.0.1", port)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	got := l.Addr().(*net.TCPAddr).Port
	if got == port {
		t.Fatal("usó el puerto ocupado")
	}
}

func TestHTTPServerTimeouts(t *testing.T) {
	s := HTTPServer(http.NotFoundHandler())
	if s.ReadTimeout != 0 || s.WriteTimeout != 0 || s.ReadHeaderTimeout == 0 || s.IdleTimeout == 0 {
		t.Fatalf("%+v", s)
	}
}
```

- [ ] **Step 3: Verificar que falla**

Run: `go test ./internal/share/`
Expected: FAIL — `undefined: New`, `Deps`, `Listen`, `HTTPServer`.

- [ ] **Step 4: Implementar `listen.go`**

```go
package share

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

// Listen prueba preferred..preferred+9 y, si están todos ocupados, uno aleatorio.
func Listen(host string, preferred int) (net.Listener, error) {
	for p := preferred; p < preferred+10; p++ {
		if l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, p)); err == nil {
			return l, nil
		}
	}
	return net.Listen("tcp", net.JoinHostPort(host, "0"))
}

// HTTPServer aplica los timeouts del spec: nunca Read/WriteTimeout (matan descargas largas).
func HTTPServer(h http.Handler) *http.Server {
	return &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 120 * time.Second}
}
```

- [ ] **Step 5: Implementar `server.go`**

```go
// Package share es el listener público. SOLO lee la sesión; nunca la controla.
package share

import (
	"html/template"
	"io/fs"
	"net"
	"net/http"

	"github.com/KonixDev/pasame/internal/i18n"
	"github.com/KonixDev/pasame/internal/session"
	"github.com/KonixDev/pasame/web"
)

type Deps struct {
	Current    func() *session.Session
	Stats      func() *session.Stats
	Quarantine string
	Sender     func() string
	Strict     func() bool
}

type Server struct {
	d   Deps
	mux *http.ServeMux
	tpl *template.Template
}

// view es lo que reciben todas las plantillas.
type view struct {
	Lang   i18n.Lang
	Sender string
	Sess   *session.Session
	Strict bool
	Msg    string // aviso de resultado (subida OK, error)
	MsgErr bool
}

func New(d Deps) (*Server, error) {
	css, err := fs.ReadFile(web.Share, "share/page.css")
	if err != nil {
		return nil, err
	}
	tpl, err := template.New("").Funcs(template.FuncMap{
		"t":    i18n.T,
		"size": i18n.Size,
		"css":  func() template.CSS { return template.CSS(css) },
	}).ParseFS(web.Share, "share/*.html")
	if err != nil {
		return nil, err
	}
	s := &Server{d: d, mux: http.NewServeMux(), tpl: tpl}
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	s.mux.HandleFunc("GET /{$}", s.root)
	s.mux.HandleFunc("GET /s/{tok}", s.withSession(s.page))
	s.mux.HandleFunc("GET /s/{tok}/f/{i}", s.withSession(s.file))
	s.mux.HandleFunc("GET /s/{tok}/zip", s.withSession(s.zip))
	s.mux.HandleFunc("POST /s/{tok}/up", s.withSession(s.upload))
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Server) view(r *http.Request, sess *session.Session) view {
	return view{Lang: i18n.Pick(r.Header.Get("Accept-Language")), Sender: s.d.Sender(), Sess: sess, Strict: s.d.Strict()}
}

func (s *Server) render(w http.ResponseWriter, status int, name string, v view) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	s.tpl.ExecuteTemplate(w, name, v)
}

func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	sess := s.d.Current()
	switch {
	case sess == nil:
		s.render(w, http.StatusGone, "gone.html", s.view(r, nil))
	case s.d.Strict():
		v := s.view(r, nil)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		s.tpl.ExecuteTemplate(w, "gone.html", view{Lang: v.Lang, Msg: i18n.T(v.Lang, "root_strict")})
	default:
		http.Redirect(w, r, sess.Path(), http.StatusFound)
	}
}

type sessHandler func(http.ResponseWriter, *http.Request, *session.Session, *session.Stats)

// withSession resuelve el token; si no es el actual, 410. Registra al cliente.
func (s *Server) withSession(h sessHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess := s.d.Current()
		if sess == nil || r.PathValue("tok") != sess.Token {
			s.render(w, http.StatusGone, "gone.html", s.view(r, nil))
			return
		}
		st := s.d.Stats()
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			st.ClientSeen(host)
		}
		h(w, r, sess, st)
	}
}

// Stubs: los completan las Tareas 7 a 10.
func (s *Server) page(w http.ResponseWriter, r *http.Request, _ *session.Session, _ *session.Stats) {
	http.Error(w, "todavía no", http.StatusNotImplemented)
}
func (s *Server) file(w http.ResponseWriter, r *http.Request, _ *session.Session, _ *session.Stats) {
	http.Error(w, "todavía no", http.StatusNotImplemented)
}
func (s *Server) zip(w http.ResponseWriter, r *http.Request, _ *session.Session, _ *session.Stats) {
	http.Error(w, "todavía no", http.StatusNotImplemented)
}
func (s *Server) upload(w http.ResponseWriter, r *http.Request, _ *session.Session, _ *session.Stats) {
	http.Error(w, "todavía no", http.StatusNotImplemented)
}
```

Y `gone.html` muestra `.Msg` en lugar del título cuando viene lleno. Reemplazar el `<main>` de `web/share/gone.html` por:
```html
<main>
<p class="brand">Pasame</p>
{{if .Msg}}<h1>{{.Msg}}</h1>{{else}}<h1>{{t .Lang "gone_title"}}</h1>
<p>{{t .Lang "gone_body"}}</p>{{end}}
</main>
```

- [ ] **Step 6: Verificar que pasa**

Run: `go test -race ./internal/share/`
Expected: `ok`.

- [ ] **Step 7: Commit**

```bash
git add web internal/share
git commit -m "feat(share): esqueleto del listener público, sesión vencida y fallback de puertos

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Página del receptor (sin JS) + JS progresivo de subida

Spec: §3.3 (compromiso de compatibilidad), §4.4 (maqueta de la página, JS progresivo), §6.4, §13 (detección de tipo para "Ver").

**Files:**
- Create: `web/share/page.html`, `web/share/page.js`, `internal/share/page.go`
- Modify: `internal/share/server.go` (borrar el stub `page`)
- Test: `internal/share/page_test.go`

**Interfaces:**
- Consumes: `Server`, `view`, `render` (Tarea 6).
- Produces: `share.Viewable(name string) bool` (imagen/video mp4|webm/audio/pdf); plantilla `page.html` con los `id` que usa `page.js`: `up-form`, `up-input`, `up-send`, `up-status`, `up-bar`. La query `?subido=N` muestra `sent_n`; `?error=<clave>` muestra ese texto de i18n (solo claves `err_*`).

- [ ] **Step 1: Test que falla**

`internal/share/page_test.go`:
```go
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
```
(`.mov` queda afuera a propósito: el spec lista solo `video/mp4|webm`, que son los que reproducen todas las TVs y navegadores.)

- [ ] **Step 2: Verificar que falla**

Run: `go test ./internal/share/ -run 'Page|Viewable'`
Expected: FAIL — 501 del stub y `undefined: Viewable`.

- [ ] **Step 3: Plantilla `web/share/page.html`**

```html
{{define "page.html"}}<!DOCTYPE html>
<html lang="{{.Lang}}"><head>{{template "head" .}}</head>
<body><main>
<p class="brand">Pasame</p>
{{$p := .Sess.Path}}{{$l := .Lang}}
{{if .Sess.Files}}
<h1>{{if eq (len .Sess.Files) 1}}{{t $l "sharing_1" .Sender}}{{else}}{{t $l "sharing_n" .Sender (len .Sess.Files)}}{{end}}</h1>
<a class="big" href="{{$p}}/zip">⬇ {{t $l "download_all" (size $l .Sess.TotalSize)}}</a>
<ul>
{{range .Sess.Files}}<li>{{if viewable .Name}}<a class="view" href="{{$p}}/f/{{.Index}}?inline=1">{{t $l "view"}}</a>{{end}}<a class="name" href="{{$p}}/f/{{.Index}}">{{.Rel}}</a><span class="size">{{size $l .Size}}</span></li>
{{end}}</ul>
<h2>{{t $l "send_back" .Sender}}</h2>
{{else}}
<h1>{{t $l "receive_only" .Sender}}</h1>
{{end}}
{{if .Msg}}<p class="{{if .MsgErr}}err{{else}}ok{{end}}" id="up-result">{{.Msg}}</p>{{end}}
<form id="up-form" method="post" action="{{$p}}/up" enctype="multipart/form-data">
<label class="big alt" for="up-input">⬆ {{t $l "choose_send"}}</label>
<input id="up-input" type="file" name="f" multiple>
<button id="up-send" type="submit" class="big">{{t $l "send"}}</button>
</form>
<p id="up-status" class="note"></p>
<div class="bar" id="up-bar-wrap" style="display:none"><div id="up-bar"></div></div>
<p class="note">{{if .Strict}}{{t $l "footer_tunnel"}}{{else}}{{t $l "footer_lan"}}{{end}}</p>
</main>
<script>var T={sending:"{{t $l "sending_js" 0 0 0}}",done:"{{t $l "sent_done_js" .Sender}}",fail:"{{t $l "err_generic"}}"};</script>
<script>{{js}}</script>
</body></html>{{end}}
```
`T.sending` llega como `"Enviando 0 de 0 · 0 %"`; `page.js` reemplaza los tres `0` en orden. Evita meter `printf` del lado del cliente.

- [ ] **Step 4: `web/share/page.js` (ES5, < 4 KB)**

```js
(function () {
  var form = document.getElementById('up-form');
  var input = document.getElementById('up-input');
  var send = document.getElementById('up-send');
  var status = document.getElementById('up-status');
  var wrap = document.getElementById('up-bar-wrap');
  var bar = document.getElementById('up-bar');
  if (!form || !window.XMLHttpRequest || !window.FormData) return; // sin JS útil: queda el form clásico
  send.style.display = 'none';

  function fmt(i, n, pct) {
    var parts = [i, n, pct], k = 0;
    return T.sending.replace(/0/g, function () { return parts[k++]; });
  }

  function upload(files, i) {
    if (i >= files.length) {
      wrap.style.display = 'none';
      status.className = 'note ok';
      status.innerHTML = '';
      status.appendChild(document.createTextNode(T.done));
      input.value = '';
      return;
    }
    var fd = new FormData();
    fd.append('f', files[i]);
    var xhr = new XMLHttpRequest();
    xhr.open('POST', form.action);
    xhr.setRequestHeader('Accept', 'application/json');
    xhr.upload.onprogress = function (e) {
      if (!e.lengthComputable) return;
      var pct = Math.floor(e.loaded * 100 / e.total);
      bar.style.width = pct + '%';
      status.innerHTML = '';
      status.appendChild(document.createTextNode(fmt(i + 1, files.length, pct)));
    };
    xhr.onload = function () {
      if (xhr.status >= 200 && xhr.status < 300) return upload(files, i + 1);
      var msg = T.fail;
      try { msg = JSON.parse(xhr.responseText).error || msg; } catch (e) {}
      fail(msg);
    };
    xhr.onerror = function () { fail(T.fail); };
    wrap.style.display = 'block';
    bar.style.width = '0';
    xhr.send(fd);
  }

  function fail(msg) {
    wrap.style.display = 'none';
    status.className = 'note err';
    status.innerHTML = '';
    status.appendChild(document.createTextNode(msg));
  }

  input.onchange = function () {
    if (input.files && input.files.length) upload(input.files, 0);
  };
})();
```
(El `replace(/0/g, …)` solo funciona porque en `sending_js` no hay otros ceros. Si se cambia el texto, revisarlo: el test de Playwright de la Tarea 21 lo detecta.)

- [ ] **Step 5: Implementar `internal/share/page.go`**

```go
package share

import (
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/KonixDev/pasame/internal/i18n"
	"github.com/KonixDev/pasame/internal/session"
	"github.com/KonixDev/pasame/web"
)

var viewExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".heic": true,
	".mp4": true, ".webm": true,
	".mp3": true, ".m4a": true, ".aac": true, ".wav": true, ".ogg": true,
	".pdf": true,
}

// Viewable indica si el archivo merece botón "Ver" (abrir inline).
func Viewable(name string) bool { return viewExt[strings.ToLower(filepath.Ext(name))] }

func (s *Server) page(w http.ResponseWriter, r *http.Request, sess *session.Session, _ *session.Stats) {
	v := s.view(r, sess)
	if n, err := strconv.Atoi(r.URL.Query().Get("subido")); err == nil && n > 0 {
		v.Msg = i18n.T(v.Lang, "sent_n", n)
		if n == 1 {
			v.Msg = i18n.T(v.Lang, "sent_1")
		}
	}
	if k := r.URL.Query().Get("error"); strings.HasPrefix(k, "err_") {
		// Solo err_no_space lleva %s; pasarle el nombre a las demás agregaría "%!(EXTRA…)".
		msg := i18n.T(v.Lang, k)
		if strings.Contains(msg, "%s") {
			msg = i18n.T(v.Lang, k, v.Sender)
		}
		v.Msg, v.MsgErr = msg, true
	}
	s.render(w, http.StatusOK, "page.html", v)
}

func mustRead(name string) string {
	b, err := fs.ReadFile(web.Share, name)
	if err != nil {
		panic(fmt.Sprintf("falta %s embebido: %v", name, err))
	}
	return string(b)
}
```

En `server.go`: borrar el stub `page` y agregar al `FuncMap`:
```go
		"viewable": Viewable,
		"js":       func() template.JS { return template.JS(mustRead("share/page.js")) },
```

- [ ] **Step 6: Verificar que pasa**

Run: `go test -race ./internal/share/`
Expected: `ok`.

- [ ] **Step 7: Revisión visual rápida**

Run: `go test ./internal/share/ -run TestPageListsFiles -v` alcanza para la lógica. La verificación en navegadores reales (con y sin JS) es la Tarea 21.

- [ ] **Step 8: Commit**

```bash
git add web/share internal/share
git commit -m "feat(share): página del receptor sin JS y subida progresiva con XHR

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Descarga individual con Range y `Content-Disposition`

Spec: §4.4 (fila `GET /s/<token>/f/<i>`), §7 (archivo desaparece), §9.2 (descarga completa, Range, 20 concurrentes).

**Files:**
- Create: `internal/share/file.go`
- Modify: `internal/share/server.go` (borrar el stub `file`)
- Test: `internal/share/file_test.go`

**Interfaces:**
- Consumes: `withSession`, `Stats.DownloadStarted/DownloadDone` (Tareas 4 y 6).
- Produces: `share.contentDisposition(kind, name string) string` (lo reutiliza `zip.go`); `countingWriter` (envuelve `ResponseWriter` para saber si la descarga llegó completa).

- [ ] **Step 1: Test que falla**

`internal/share/file_test.go`:
```go
package share

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestDownloadFull(t *testing.T) {
	f := newFixture(t, map[string]int{"año 2026.bin": 3_000_000})
	w := f.do(httptest.NewRequest("GET", f.sess.Path()+"/f/0", nil))
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), pattern(3_000_000)) {
		t.Fatalf("code %d, len %d", w.Code, w.Body.Len())
	}
	cd := w.Header().Get("Content-Disposition")
	if !strings.HasPrefix(cd, "attachment;") ||
		!strings.Contains(cd, `filename*=UTF-8''a%C3%B1o%202026.bin`) ||
		!strings.Contains(cd, `filename="ao 2026.bin"`) {
		t.Fatalf("Content-Disposition = %q", cd)
	}
	if w.Header().Get("Content-Length") != "3000000" || w.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatalf("headers %v", w.Header())
	}
	if f.stats.Snapshot().Completed[0] != 1 {
		t.Fatal("no contó la descarga completa")
	}
}

func TestDownloadRange(t *testing.T) {
	f := newFixture(t, map[string]int{"v.mp4": 1_000_000})
	r := httptest.NewRequest("GET", f.sess.Path()+"/f/0", nil)
	r.Header.Set("Range", "bytes=500000-")
	w := f.do(r)
	if w.Code != http.StatusPartialContent || w.Header().Get("Content-Range") != "bytes 500000-999999/1000000" {
		t.Fatalf("%d %q", w.Code, w.Header().Get("Content-Range"))
	}
	if !bytes.Equal(w.Body.Bytes(), pattern(1_000_000)[500000:]) {
		t.Fatal("bytes distintos")
	}
}

func TestDownloadInline(t *testing.T) {
	f := newFixture(t, map[string]int{"foto.jpg": 10})
	w := f.do(httptest.NewRequest("GET", f.sess.Path()+"/f/0?inline=1", nil))
	if !strings.HasPrefix(w.Header().Get("Content-Disposition"), "inline;") ||
		w.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("%v", w.Header())
	}
}

func TestDownloadBadIndex(t *testing.T) {
	f := newFixture(t, map[string]int{"a": 1})
	// No hay caso "../": las URLs usan índices, así que un nombre de ruta nunca llega a os.Open.
	for _, p := range []string{"/f/1", "/f/-1", "/f/x", "/f/0x1", "/f/%2e%2e"} {
		if w := f.do(httptest.NewRequest("GET", f.sess.Path()+p, nil)); w.Code != http.StatusNotFound {
			t.Errorf("%s → %d", p, w.Code)
		}
	}
}

func TestDownloadFileVanished(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 10})
	os.Remove(f.sess.Files[0].Abs)
	w := f.do(httptest.NewRequest("GET", f.sess.Path()+"/f/0", nil))
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "ya no está disponible") {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
}

func TestDownloadFileChangedSize(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 10})
	os.WriteFile(f.sess.Files[0].Abs, []byte("otro contenido más largo"), 0o644)
	if w := f.do(httptest.NewRequest("GET", f.sess.Path()+"/f/0", nil)); w.Code != http.StatusNotFound {
		t.Fatalf("%d", w.Code)
	}
}

func TestDownloadConcurrentMemory(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	f := newFixture(t, map[string]int{"big.bin": 20_000_000})
	ts := httptest.NewServer(f.srv)
	defer ts.Close()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(ts.URL + f.sess.Path() + "/f/0")
			if err != nil {
				t.Error(err)
				return
			}
			n, _ := io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if n != 20_000_000 {
				t.Errorf("n = %d", n)
			}
		}()
	}
	wg.Wait()
	runtime.ReadMemStats(&after)
	if grown := int64(after.HeapInuse) - int64(before.HeapInuse); grown > 50<<20 {
		t.Fatalf("el heap creció %d MB", grown>>20)
	}
}
```

- [ ] **Step 2: Verificar que falla**

Run: `go test ./internal/share/ -run Download`
Expected: FAIL — 501.

- [ ] **Step 3: Implementar `internal/share/file.go`**

```go
package share

import (
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/KonixDev/pasame/internal/i18n"
	"github.com/KonixDev/pasame/internal/session"
)

func (s *Server) file(w http.ResponseWriter, r *http.Request, sess *session.Session, st *session.Stats) {
	i, err := strconv.Atoi(r.PathValue("i"))
	f, ok := sess.File(i)
	if err != nil || !ok {
		http.NotFound(w, r)
		return
	}
	fh, err := os.Open(f.Abs)
	var info os.FileInfo
	if err == nil {
		info, err = fh.Stat()
	}
	if err != nil || info.Size() != f.Size {
		if fh != nil {
			fh.Close()
		}
		v := s.view(r, sess)
		http.Error(w, i18n.T(v.Lang, "file_gone"), http.StatusNotFound)
		return
	}
	defer fh.Close()

	kind := "attachment"
	if r.URL.Query().Get("inline") == "1" {
		kind = "inline"
	}
	w.Header().Set("Content-Disposition", contentDisposition(kind, f.Name))
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(f.Name))); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	st.DownloadStarted(i)
	cw := &countingWriter{ResponseWriter: w}
	http.ServeContent(cw, r, f.Name, f.ModTime, fh)
	// "Completa" = se mandó el archivo entero en esta respuesta (no cuenta un Range parcial).
	st.DownloadDone(i, cw.status == http.StatusOK && cw.n == f.Size)
}

// contentDisposition arma el header con nombre UTF-8 (RFC 6266) y un fallback ASCII.
func contentDisposition(kind, name string) string {
	var ascii strings.Builder
	for _, r := range name {
		if r >= 0x20 && r < 0x7f && r != '"' && r != '\\' {
			ascii.WriteRune(r)
		}
	}
	return kind + `; filename="` + ascii.String() + `"; filename*=UTF-8''` + strings.ReplaceAll(url.QueryEscape(name), "+", "%20")
}

type countingWriter struct {
	http.ResponseWriter
	status int
	n      int64
}

func (c *countingWriter) WriteHeader(code int) { c.status = code; c.ResponseWriter.WriteHeader(code) }

func (c *countingWriter) Write(b []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	n, err := c.ResponseWriter.Write(b)
	c.n += int64(n)
	return n, err
}

// Unwrap deja que http.ResponseController llegue al writer real (flush, deadlines).
func (c *countingWriter) Unwrap() http.ResponseWriter { return c.ResponseWriter }
```
Borrar el stub `file` de `server.go`.

- [ ] **Step 4: Verificar que pasa**

Run: `go test -race ./internal/share/`
Expected: `ok`. (`TestDownloadConcurrentMemory` tarda ~1 s; con `-short` se saltea.)

- [ ] **Step 5: Commit**

```bash
git add internal/share
git commit -m "feat(share): descarga individual con Range, reanudación y nombre UTF-8

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: ZIP al vuelo

Spec: §4.4 (fila `GET /s/<token>/zip`), §7 (el ZIP omite archivos que fallan), §9.2 (ZIP64 con sparse de 5 GB, RSS estable), §13 (buffer 256 KB).

**Files:**
- Create: `internal/share/zip.go`
- Modify: `internal/share/server.go` (borrar el stub `zip`)
- Test: `internal/share/zip_test.go`

**Interfaces:**
- Consumes: `contentDisposition`, `countingWriter` (Tarea 8).
- Produces: `const copyBuf = 256 << 10` y `var bufPool sync.Pool` (los reutiliza `upload.go`).

- [ ] **Step 1: Test que falla**

`internal/share/zip_test.go`:
```go
package share

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KonixDev/pasame/internal/session"
)

func readZip(t *testing.T, b []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatalf("ZIP inválido: %v", err)
	}
	out := map[string][]byte{}
	for _, zf := range zr.File {
		if zf.Method != zip.Store {
			t.Errorf("%s comprimido con método %d", zf.Name, zf.Method)
		}
		rc, _ := zf.Open()
		out[zf.Name], _ = io.ReadAll(rc)
		rc.Close()
	}
	return out
}

func TestZipAll(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 100, "b.bin": 200_000})
	w := f.do(httptest.NewRequest("GET", f.sess.Path()+"/zip", nil))
	if w.Code != 200 || w.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("%d %v", w.Code, w.Header())
	}
	if cd := w.Header().Get("Content-Disposition"); !strings.Contains(cd, `filename="Pasame-`) || !strings.HasSuffix(cd, ".zip") {
		t.Fatalf("CD = %q", cd)
	}
	files := readZip(t, w.Body.Bytes())
	if !bytes.Equal(files["a.txt"], pattern(100)) || !bytes.Equal(files["b.bin"], pattern(200_000)) {
		t.Fatal("contenido distinto")
	}
}

func TestZipKeepsFolderStructure(t *testing.T) {
	d := t.TempDir()
	os.MkdirAll(filepath.Join(d, "album", "sub"), 0o755)
	os.WriteFile(filepath.Join(d, "album", "sub", "c.jpg"), []byte("c"), 0o644)
	f := newFixture(t, nil)
	f.sess, _ = session.New([]string{filepath.Join(d, "album")})
	files := readZip(t, f.do(httptest.NewRequest("GET", f.sess.Path()+"/zip", nil)).Body.Bytes())
	if string(files["album/sub/c.jpg"]) != "c" {
		t.Fatalf("entradas: %v", files)
	}
}

func TestZipSkipsVanishedFile(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 10, "b.txt": 10, "c.txt": 10})
	os.Remove(f.sess.Files[1].Abs)
	files := readZip(t, f.do(httptest.NewRequest("GET", f.sess.Path()+"/zip", nil)).Body.Bytes())
	if len(files) != 2 || files["a.txt"] == nil || files["c.txt"] == nil {
		t.Fatalf("entradas: %v", files)
	}
}

func TestZipEmptySessionIs404(t *testing.T) {
	f := newFixture(t, nil)
	if w := f.do(httptest.NewRequest("GET", f.sess.Path()+"/zip", nil)); w.Code != http.StatusNotFound {
		t.Fatalf("%d", w.Code)
	}
}

// ZIP64 real: un archivo sparse de 5 GB no ocupa disco pero obliga a ZIP64.
func TestZip64Sparse(t *testing.T) {
	if testing.Short() {
		t.Skip("lee 5 GB de ceros; ~10 s")
	}
	d := t.TempDir()
	big := filepath.Join(d, "big.bin")
	fh, _ := os.Create(big)
	if err := fh.Truncate(5 << 30); err != nil {
		t.Skip("el FS no soporta sparse:", err)
	}
	fh.Close()
	f := newFixture(t, nil)
	f.sess, _ = session.New([]string{big})
	ts := httptest.NewServer(f.srv)
	defer ts.Close()
	resp, err := http.Get(ts.URL + f.sess.Path() + "/zip")
	if err != nil {
		t.Fatal(err)
	}
	tmp, _ := os.CreateTemp(d, "*.zip")
	n, _ := io.Copy(tmp, resp.Body)
	resp.Body.Close()
	zr, err := zip.NewReader(tmp, n)
	if err != nil || len(zr.File) != 1 || zr.File[0].UncompressedSize64 != 5<<30 {
		t.Fatalf("zip64 inválido: %v", err)
	}
}
```

- [ ] **Step 2: Verificar que falla**

Run: `go test -short ./internal/share/ -run Zip`
Expected: FAIL — 501.

- [ ] **Step 3: Implementar `internal/share/zip.go`**

```go
package share

import (
	"archive/zip"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/KonixDev/pasame/internal/session"
)

const copyBuf = 256 << 10

var bufPool = sync.Pool{New: func() any { b := make([]byte, copyBuf); return &b }}

// zip escribe todos los archivos directo al ResponseWriter. Store: fotos y videos ya vienen comprimidos.
// Sin Content-Length (tamaño desconocido) y sin Range: un ZIP al vuelo no es reanudable.
func (s *Server) zip(w http.ResponseWriter, r *http.Request, sess *session.Session, st *session.Stats) {
	if len(sess.Files) == 0 {
		http.NotFound(w, r)
		return
	}
	name := "Pasame-" + time.Now().Format("2006-01-02") + ".zip"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", contentDisposition("attachment", name))
	w.Header().Set("Cache-Control", "no-store")

	// El ZIP cuenta como una descarga "en curso" de cada archivo (lo ve la Actividad del emisor).
	for _, f := range sess.Files {
		st.DownloadStarted(f.Index)
	}
	ok := map[int]bool{}
	defer func() {
		for _, f := range sess.Files {
			st.DownloadDone(f.Index, ok[f.Index])
		}
	}()

	zw := zip.NewWriter(w)
	bp := bufPool.Get().(*[]byte)
	defer bufPool.Put(bp)
	for _, f := range sess.Files {
		fh, err := os.Open(f.Abs)
		if err != nil {
			log.Printf("zip: omito %s: %v", f.Rel, err)
			continue
		}
		hdr := &zip.FileHeader{Name: f.Rel, Method: zip.Store, Modified: f.ModTime}
		hdr.SetMode(0o644)
		dst, err := zw.CreateHeader(hdr)
		if err == nil {
			_, err = io.CopyBuffer(dst, fh, *bp)
		}
		fh.Close()
		if err != nil {
			log.Printf("zip: cortado en %s: %v", f.Rel, err)
			return // el cliente se fue; no hay forma de "arreglar" un stream a medias
		}
		ok[f.Index] = true
	}
	zw.Close()
}
```
Borrar el stub `zip` de `server.go`.

Nota: `archive/zip` activa ZIP64 automáticamente cuando el tamaño supera 4 GB aunque el header no lo declare de antemano (usa data descriptor). El test `TestZip64Sparse` lo confirma.

- [ ] **Step 4: Verificar que pasa**

Run: `go test -race -short ./internal/share/ && go test ./internal/share/ -run Zip64`
Expected: `ok` en ambos.

- [ ] **Step 5: Commit**

```bash
git add internal/share
git commit -m "feat(share): ZIP al vuelo sin compresión, con ZIP64 y omisión de faltantes

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: Dirección inversa — subida multipart en streaming

Spec: §4.4 (fila `POST /s/<token>/up`, watchdog de 60 s), §5 (path traversal, ejecutables), §6.5 (disco lleno, archivo vacío), §7 (subida cortada), §9.2 (subida de 3 archivos, corte, disco lleno).

**Files:**
- Create: `internal/share/upload.go`
- Modify: `internal/share/server.go` (borrar el stub `upload`; agregar el campo `createPart`)
- Test: `internal/share/upload_test.go`

**Interfaces:**
- Consumes: `files.UniquePath`, `files.MarkDownloaded` (Tarea 2); `bufPool` (Tarea 9); `Stats.UploadStarted/UploadDone`.
- Produces: respuesta JSON `{"ok":true,"files":["a.jpg",...]}` o `{"ok":false,"error":"<texto>"}` cuando el request trae `Accept: application/json`; si no, `303` a `<path>?subido=N` o `<path>?error=err_*`. Campo `Server.createPart func(path string) (io.WriteCloser, error)` inyectable en tests.

- [ ] **Step 1: Test que falla**

`internal/share/upload_test.go`:
```go
package share

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func multipartBody(t *testing.T, files map[string][]byte) (*bytes.Buffer, string) {
	t.Helper()
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sortStrings(names)
	for _, n := range names {
		fw, _ := mw.CreateFormFile("f", n)
		fw.Write(files[n])
	}
	mw.Close()
	return &b, mw.FormDataContentType()
}

func (f *fixture) upload(t *testing.T, files map[string][]byte, jsonResp bool) *httptest.ResponseRecorder {
	body, ct := multipartBody(t, files)
	r := httptest.NewRequest("POST", f.sess.Path()+"/up", body)
	r.Header.Set("Content-Type", ct)
	if jsonResp {
		r.Header.Set("Accept", "application/json")
	}
	return f.do(r)
}

func listDir(t *testing.T, d string) []string {
	es, _ := os.ReadDir(d)
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

func TestUploadThreeFilesNoJS(t *testing.T) {
	f := newFixture(t, nil)
	w := f.upload(t, map[string][]byte{"a.jpg": []byte("aa"), "b.mp4": []byte("bbb"), "c.pdf": []byte("c")}, false)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != f.sess.Path()+"?subido=3" {
		t.Fatalf("%d %q", w.Code, w.Header().Get("Location"))
	}
	got := listDir(t, f.quar)
	if strings.Join(got, ",") != "a.jpg,b.mp4,c.pdf" {
		t.Fatalf("cuarentena: %v", got)
	}
	if b, _ := os.ReadFile(filepath.Join(f.quar, "b.mp4")); string(b) != "bbb" {
		t.Fatal("contenido distinto")
	}
	if len(f.stats.Snapshot().Received) != 3 {
		t.Fatal("stats no registró las subidas")
	}
}

func TestUploadJSONAndSanitizeAndCollision(t *testing.T) {
	f := newFixture(t, nil)
	os.WriteFile(filepath.Join(f.quar, "evil.txt"), []byte("ya estaba"), 0o644)
	w := f.upload(t, map[string][]byte{"../../evil.txt": []byte("x")}, true)
	var resp struct {
		OK    bool
		Files []string
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if w.Code != 200 || !resp.OK || len(resp.Files) != 1 || resp.Files[0] != "evil (2).txt" {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if b, _ := os.ReadFile(filepath.Join(f.quar, "evil.txt")); string(b) != "ya estaba" {
		t.Fatal("pisó un archivo existente")
	}
}

func TestUploadEmptyFile(t *testing.T) {
	f := newFixture(t, nil)
	w := f.upload(t, map[string][]byte{"vacio.txt": nil}, false)
	if w.Header().Get("Location") != f.sess.Path()+"?error=err_empty" || len(listDir(t, f.quar)) != 0 {
		t.Fatalf("%q %v", w.Header().Get("Location"), listDir(t, f.quar))
	}
}

type fullDisk struct{ n int }

func (d *fullDisk) Write(p []byte) (int, error) {
	if d.n+len(p) > 10 {
		return 0, &os.PathError{Op: "write", Path: "x", Err: syscall.ENOSPC}
	}
	d.n += len(p)
	return len(p), nil
}
func (d *fullDisk) Close() error { return nil }

func TestUploadDiskFull(t *testing.T) {
	f := newFixture(t, nil)
	f.srv.createPart = func(p string) (io.WriteCloser, error) {
		os.WriteFile(p, nil, 0o644) // que exista, para verificar que se borra
		return &fullDisk{}, nil
	}
	w := f.upload(t, map[string][]byte{"grande.mp4": bytes.Repeat([]byte("x"), 100)}, true)
	if w.Code != http.StatusInsufficientStorage || !strings.Contains(w.Body.String(), "no tiene espacio") {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if len(listDir(t, f.quar)) != 0 {
		t.Fatalf("quedó basura: %v", listDir(t, f.quar))
	}
}

func TestUploadCutMidwayLeavesNoPart(t *testing.T) {
	f := newFixture(t, nil)
	ts := httptest.NewServer(f.srv)
	defer ts.Close()
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		fw, _ := mw.CreateFormFile("f", "cortado.mp4")
		fw.Write(bytes.Repeat([]byte("x"), 1<<20))
		pw.CloseWithError(errors.New("se cortó el WiFi"))
	}()
	req, _ := http.NewRequest("POST", ts.URL+f.sess.Path()+"/up", pr)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	http.DefaultClient.Do(req) // falla del lado cliente: es lo esperado
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(listDir(t, f.quar)) == 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("quedó: %v", listDir(t, f.quar))
}

func TestUploadLargeStreamsWithoutBuffering(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	f := newFixture(t, nil)
	ts := httptest.NewServer(f.srv)
	defer ts.Close()
	const size = 300 << 20
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		fw, _ := mw.CreateFormFile("f", "video.mp4")
		chunk := bytes.Repeat([]byte("v"), 1<<20)
		for i := 0; i < size>>20; i++ {
			fw.Write(chunk)
		}
		mw.Close()
		pw.Close()
	}()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	req, _ := http.NewRequest("POST", ts.URL+f.sess.Path()+"/up", pr)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("%v %v", err, resp)
	}
	runtime.ReadMemStats(&after)
	if fi, _ := os.Stat(filepath.Join(f.quar, "video.mp4")); fi == nil || fi.Size() != size {
		t.Fatal("tamaño incorrecto")
	}
	if int64(after.HeapInuse)-int64(before.HeapInuse) > 50<<20 {
		t.Fatal("la subida se bufferizó en memoria")
	}
}
```

- [ ] **Step 2: Verificar que falla**

Run: `go test -short ./internal/share/ -run Upload`
Expected: FAIL — `f.srv.createPart undefined` / 501.

- [ ] **Step 3: Implementar `internal/share/upload.go`**

```go
package share

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/KonixDev/pasame/internal/files"
	"github.com/KonixDev/pasame/internal/i18n"
	"github.com/KonixDev/pasame/internal/session"
)

const uploadIdle = 60 * time.Second

var errEmpty = errors.New("archivo vacío")

func defaultCreatePart(p string) (io.WriteCloser, error) {
	return os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
}

// upload lee el multipart parte por parte: nunca ParseMultipartForm (bufferiza).
func (s *Server) upload(w http.ResponseWriter, r *http.Request, sess *session.Session, st *session.Stats) {
	rc := http.NewResponseController(w)
	rc.SetReadDeadline(time.Now().Add(uploadIdle)) // en httptest.Recorder devuelve error: se ignora
	mr, err := r.MultipartReader()
	if err != nil {
		s.uploadResult(w, r, sess, nil, "err_generic", http.StatusBadRequest)
		return
	}
	os.MkdirAll(s.d.Quarantine, 0o755)
	var saved []string
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.uploadResult(w, r, sess, saved, "err_generic", http.StatusBadRequest)
			return
		}
		if part.FileName() == "" {
			continue // campos que no son archivos
		}
		name, err := s.savePart(part, rc, st)
		switch {
		case errors.Is(err, syscall.ENOSPC):
			s.uploadResult(w, r, sess, saved, "err_no_space", http.StatusInsufficientStorage)
			return
		case errors.Is(err, errEmpty):
			s.uploadResult(w, r, sess, saved, "err_empty", http.StatusBadRequest)
			return
		case err != nil:
			log.Printf("upload: %v", err)
			s.uploadResult(w, r, sess, saved, "err_generic", http.StatusBadRequest)
			return
		}
		saved = append(saved, name)
	}
	s.uploadResult(w, r, sess, saved, "", http.StatusOK)
}

// savePart escribe en "<nombre>.part" y renombra al terminar: nunca queda un archivo a medias con nombre final.
func (s *Server) savePart(part *multipart.Part, rc *http.ResponseController, st *session.Stats) (string, error) {
	dest, err := files.UniquePath(s.d.Quarantine, part.FileName())
	if err != nil {
		return "", err
	}
	name := filepath.Base(dest)
	tmp := dest + ".part"
	out, err := s.createPart(tmp)
	if err != nil {
		return "", err
	}
	st.UploadStarted(name)
	bp := bufPool.Get().(*[]byte)
	n, err := io.CopyBuffer(out, &idleReader{r: part, rc: rc}, *bp)
	bufPool.Put(bp)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil && n == 0 {
		err = errEmpty
	}
	if err == nil {
		err = os.Rename(tmp, dest)
	}
	if err != nil {
		os.Remove(tmp)
		st.UploadDone(name, false)
		return "", err
	}
	files.MarkDownloaded(dest)
	st.UploadDone(name, true)
	return name, nil
}

// idleReader corre el deadline de lectura en cada Read: corta si pasan 60 s sin bytes.
type idleReader struct {
	r  io.Reader
	rc *http.ResponseController
}

func (i *idleReader) Read(b []byte) (int, error) {
	i.rc.SetReadDeadline(time.Now().Add(uploadIdle))
	return i.r.Read(b)
}

func (s *Server) uploadResult(w http.ResponseWriter, r *http.Request, sess *session.Session, saved []string, errKey string, status int) {
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		resp := map[string]any{"ok": errKey == "", "files": saved}
		if errKey != "" {
			v := s.view(r, sess)
			msg := i18n.T(v.Lang, errKey)
			if strings.Contains(msg, "%s") {
				msg = i18n.T(v.Lang, errKey, v.Sender)
			}
			resp["error"] = msg
		}
		json.NewEncoder(w).Encode(resp)
		return
	}
	loc := sess.Path() + "?subido=" + strconv.Itoa(len(saved))
	if errKey != "" {
		loc = sess.Path() + "?error=" + errKey
	}
	http.Redirect(w, r, loc, http.StatusSeeOther)
}
```

En `server.go`: borrar el stub `upload`, agregar el campo `createPart func(string) (io.WriteCloser, error)` a `Server` e inicializarlo en `New` con `createPart: defaultCreatePart`. Importar `io`.

- [ ] **Step 4: Verificar que pasa**

Run: `go test -race -short ./internal/share/ && go test ./internal/share/ -run UploadLarge`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/share
git commit -m "feat(share): subida inversa en streaming con .part, saneado, disco lleno y watchdog

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 11: Direcciones — `Book` y la interfaz `Provider`

Spec: §4.5 (la costura clave).

**Files:**
- Create: `internal/addr/book.go`
- Test: `internal/addr/book_test.go`

**Interfaces:**
- Consumes: nada.
- Produces:
  ```go
  type Address struct {
      Kind    string `json:"kind"`    // "lan" | "tunnel" | "mdns"
      URL     string `json:"url"`     // lo que va al QR
      Display string `json:"display"` // lo que se muestra grande
      Label   string `json:"label"`
      Primary bool   `json:"primary"`
  }
  type Provider interface {
      Kind() string
      Addresses(ctx context.Context, sessionPath string) ([]Address, error)
  }
  func NewBook() *Book
  func (b *Book) Register(p Provider)          // reemplaza al del mismo Kind y notifica
  func (b *Book) Unregister(kind string)       // notifica
  func (b *Book) Active(ctx context.Context, sessionPath string) []Address // orden tunnel, lan, mdns; el primero Primary
  func (b *Book) Changed() <-chan struct{}     // buffer 1; nunca bloquea al notificar
  func (b *Book) Notify()                      // para proveedores que detectan cambios solos (LAN)
  ```

- [ ] **Step 1: Test que falla**

`internal/addr/book_test.go`:
```go
package addr

import (
	"context"
	"errors"
	"testing"
)

type fake struct {
	kind string
	urls []string
	err  error
}

func (f fake) Kind() string { return f.kind }
func (f fake) Addresses(_ context.Context, p string) ([]Address, error) {
	var out []Address
	for _, u := range f.urls {
		out = append(out, Address{Kind: f.kind, URL: u + p})
	}
	return out, f.err
}

func TestActiveOrderAndPrimary(t *testing.T) {
	b := NewBook()
	b.Register(fake{kind: "mdns", urls: []string{"http://pasame.local:8080"}})
	b.Register(fake{kind: "lan", urls: []string{"http://192.168.1.42:8080"}})
	got := b.Active(context.Background(), "/s/abcde")
	if len(got) != 2 || got[0].Kind != "lan" || !got[0].Primary || got[1].Primary {
		t.Fatalf("%+v", got)
	}
	b.Register(fake{kind: "tunnel", urls: []string{"https://x.trycloudflare.com"}})
	got = b.Active(context.Background(), "/s/abcde")
	if got[0].Kind != "tunnel" || got[0].URL != "https://x.trycloudflare.com/s/abcde" || !got[0].Primary {
		t.Fatalf("%+v", got)
	}
	for _, a := range got[1:] {
		if a.Primary {
			t.Fatalf("más de una primaria: %+v", got)
		}
	}
}

func TestFailingProviderIsSkipped(t *testing.T) {
	b := NewBook()
	b.Register(fake{kind: "tunnel", err: errors.New("caído")})
	b.Register(fake{kind: "lan", urls: []string{"http://10.0.0.2:8080"}})
	got := b.Active(context.Background(), "/s/x")
	if len(got) != 1 || got[0].Kind != "lan" || !got[0].Primary {
		t.Fatalf("%+v", got)
	}
}

func TestChangedNotifications(t *testing.T) {
	b := NewBook()
	b.Register(fake{kind: "lan"})
	b.Register(fake{kind: "lan"}) // dos notificaciones seguidas no bloquean
	select {
	case <-b.Changed():
	default:
		t.Fatal("no notificó Register")
	}
	b.Unregister("lan")
	select {
	case <-b.Changed():
	default:
		t.Fatal("no notificó Unregister")
	}
	if len(b.Active(context.Background(), "/")) != 0 {
		t.Fatal("Unregister no sacó el proveedor")
	}
}
```

- [ ] **Step 2: Verificar que falla**

Run: `go test ./internal/addr/`
Expected: FAIL — `undefined: NewBook`.

- [ ] **Step 3: Implementar `internal/addr/book.go`**

```go
// Package addr decide qué direcciones mostrar. La UI, el servidor y el QR solo conocen Book.
// Agregar un proveedor (ngrok, un VPS) = un archivo nuevo que implemente Provider.
package addr

import (
	"context"
	"log"
	"sync"
)

type Address struct {
	Kind    string `json:"kind"`
	URL     string `json:"url"`
	Display string `json:"display"`
	Label   string `json:"label"`
	Primary bool   `json:"primary"`
}

type Provider interface {
	Kind() string
	Addresses(ctx context.Context, sessionPath string) ([]Address, error)
}

var order = []string{"tunnel", "lan", "mdns"}

type Book struct {
	mu        sync.Mutex
	providers map[string]Provider
	changed   chan struct{}
}

func NewBook() *Book {
	return &Book{providers: map[string]Provider{}, changed: make(chan struct{}, 1)}
}

func (b *Book) Register(p Provider) {
	b.mu.Lock()
	b.providers[p.Kind()] = p
	b.mu.Unlock()
	b.Notify()
}

func (b *Book) Unregister(kind string) {
	b.mu.Lock()
	delete(b.providers, kind)
	b.mu.Unlock()
	b.Notify()
}

func (b *Book) Notify() {
	select {
	case b.changed <- struct{}{}:
	default: // ya hay una notificación pendiente
	}
}

func (b *Book) Changed() <-chan struct{} { return b.changed }

func (b *Book) Active(ctx context.Context, sessionPath string) []Address {
	b.mu.Lock()
	ps := make([]Provider, 0, len(order))
	for _, k := range order {
		if p, ok := b.providers[k]; ok {
			ps = append(ps, p)
		}
	}
	b.mu.Unlock()
	var out []Address
	for _, p := range ps {
		as, err := p.Addresses(ctx, sessionPath)
		if err != nil {
			log.Printf("addr: %s: %v", p.Kind(), err)
			continue
		}
		out = append(out, as...)
	}
	for i := range out {
		out[i].Primary = i == 0
	}
	return out
}
```

- [ ] **Step 4: Verificar que pasa**

Run: `go test -race ./internal/addr/`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/addr
git commit -m "feat(addr): libreta de direcciones con proveedores enchufables

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 12: Proveedor LAN — elegir la interfaz correcta

Spec: §4.5.1 (algoritmo completo y aviso de VPN), §9.1 (fila `addr/lan`: 8 escenarios), §12 (riesgo 3).

**Files:**
- Create: `internal/addr/lan.go`
- Test: `internal/addr/lan_test.go`

**Interfaces:**
- Consumes: `Address`, `Provider` (Tarea 11).
- Produces:
  ```go
  type Iface struct { Name string; IPs []net.IP; Up, Loopback bool }
  type Candidate struct {
      Name    string `json:"name"`    // "en0"
      Human   string `json:"human"`   // "Wi-Fi" | "Ethernet" | "Tailscale" | Name
      IP      string `json:"ip"`
      Virtual bool   `json:"virtual"`
      Score   int    `json:"-"`
  }
  type LAN struct {
      Port  int
      List  func() ([]Iface, error) // por defecto: interfaces del sistema
      Route func() net.IP           // por defecto: IP local de la ruta a 1.1.1.1 (sin enviar nada)
  }
  func NewLAN(port int) *LAN
  func (l *LAN) Kind() string                 // "lan"
  func (l *LAN) Candidates() []Candidate      // ordenados, el primero es el elegido
  func (l *LAN) SetOverride(ip string)        // "" = automático
  func (l *LAN) VPN() bool
  func (l *LAN) Addresses(ctx context.Context, sessionPath string) ([]Address, error)
  ```
  `Label` de la dirección LAN: `"En esta red WiFi"`. `Display`: `"192.168.1.42:8080"`.

- [ ] **Step 1: Test que falla (los 8 escenarios del spec + bordes)**

`internal/addr/lan_test.go`:
```go
package addr

import (
	"context"
	"net"
	"testing"
)

func ifc(name, ip string) Iface {
	return Iface{Name: name, Up: true, IPs: []net.IP{net.ParseIP(ip)}}
}

func lanWith(route string, ifs ...Iface) *LAN {
	l := NewLAN(8080)
	l.List = func() ([]Iface, error) { return ifs, nil }
	l.Route = func() net.IP { return net.ParseIP(route) }
	return l
}

func primary(t *testing.T, l *LAN) string {
	t.Helper()
	as, _ := l.Addresses(context.Background(), "/s/abcde")
	if len(as) == 0 {
		return ""
	}
	return as[0].Display
}

func TestLANScenarios(t *testing.T) {
	lo := Iface{Name: "lo0", Up: true, Loopback: true, IPs: []net.IP{net.ParseIP("127.0.0.1")}}
	cases := []struct {
		name string
		l    *LAN
		want string
		vpn  bool
	}{
		{"solo WiFi", lanWith("192.168.1.42", lo, ifc("en0", "192.168.1.42")), "192.168.1.42:8080", false},
		{"WiFi + Tailscale", lanWith("192.168.1.42", ifc("en0", "192.168.1.42"), ifc("utun3", "100.101.1.2")), "192.168.1.42:8080", true},
		{"WiFi + VPN full-tunnel", lanWith("10.8.0.2", ifc("en0", "192.168.1.42"), ifc("utun4", "10.8.0.2")), "192.168.1.42:8080", true},
		{"Ethernet + WiFi misma red", lanWith("192.168.1.43", ifc("en0", "192.168.1.42"), ifc("en7", "192.168.1.43")), "192.168.1.43:8080", false},
		{"Docker + WiFi", lanWith("192.168.0.10", ifc("docker0", "172.17.0.1"), ifc("br-3f2a", "172.18.0.1"), ifc("veth12", "172.17.0.5"), ifc("wlan0", "192.168.0.10")), "192.168.0.10:8080", false},
		{"solo VPN", lanWith("10.8.0.2", lo, ifc("wg0", "10.8.0.2")), "10.8.0.2:8080", true},
		{"sin red", lanWith("", lo), "", false},
		{"Windows Hyper-V + Wi-Fi", lanWith("192.168.1.9", ifc("vEthernet (WSL)", "172.25.0.1"), ifc("Wi-Fi", "192.168.1.9")), "192.168.1.9:8080", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := primary(t, c.l); got != c.want {
				t.Fatalf("primaria = %q, want %q (candidatas %+v)", got, c.want, c.l.Candidates())
			}
			if c.l.VPN() != c.vpn {
				t.Fatalf("VPN() = %v", c.l.VPN())
			}
		})
	}
}

func TestLANOverride(t *testing.T) {
	l := lanWith("192.168.1.42", ifc("en0", "192.168.1.42"), ifc("tailscale0", "100.101.1.2"))
	l.SetOverride("100.101.1.2")
	if got := primary(t, l); got != "100.101.1.2:8080" {
		t.Fatalf("override ignorado: %q", got)
	}
	l.SetOverride("10.99.99.99") // ya no existe → automático
	if got := primary(t, l); got != "192.168.1.42:8080" {
		t.Fatalf("override inexistente no cayó a automático: %q", got)
	}
}

func TestLANIgnoresUnusable(t *testing.T) {
	l := lanWith("",
		Iface{Name: "en1", Up: false, IPs: []net.IP{net.ParseIP("192.168.5.5")}},
		ifc("en2", "169.254.10.10"),
		ifc("en3", "fe80::1"),
	)
	if c := l.Candidates(); len(c) != 0 {
		t.Fatalf("%+v", c)
	}
}

func TestLANVirtualStillListed(t *testing.T) {
	l := lanWith("192.168.1.42", ifc("en0", "192.168.1.42"), ifc("tailscale0", "100.101.1.2"))
	c := l.Candidates()
	if len(c) != 2 || c[1].Human != "Tailscale" || !c[1].Virtual || c[0].Human != "Wi-Fi" {
		t.Fatalf("%+v", c)
	}
}

func TestLANAddressShape(t *testing.T) {
	l := lanWith("192.168.1.42", ifc("en0", "192.168.1.42"))
	as, _ := l.Addresses(context.Background(), "/s/k3x9m")
	if len(as) != 1 || as[0].URL != "http://192.168.1.42:8080/s/k3x9m" || as[0].Kind != "lan" || as[0].Label != "En esta red WiFi" {
		t.Fatalf("%+v", as)
	}
}
```

- [ ] **Step 2: Verificar que falla**

Run: `go test ./internal/addr/ -run LAN`
Expected: FAIL — `undefined: NewLAN`.

- [ ] **Step 3: Implementar `internal/addr/lan.go`**

```go
package addr

import (
	"context"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Iface struct {
	Name     string
	IPs      []net.IP
	Up       bool
	Loopback bool
}

type Candidate struct {
	Name    string `json:"name"`
	Human   string `json:"human"`
	IP      string `json:"ip"`
	Virtual bool   `json:"virtual"`
	Score   int    `json:"-"`
}

type LAN struct {
	Port  int
	List  func() ([]Iface, error)
	Route func() net.IP

	mu       sync.Mutex
	override string
}

func NewLAN(port int) *LAN { return &LAN{Port: port, List: systemIfaces, Route: routeIP} }

func (l *LAN) Kind() string { return "lan" }

func (l *LAN) SetOverride(ip string) {
	l.mu.Lock()
	l.override = ip
	l.mu.Unlock()
}

// Prefijos de interfaces virtuales: pierden preferencia pero siguen en "Cambiar red".
var virtualPrefixes = []string{"docker", "br-", "veth", "virbr", "vbox", "vmnet", "utun", "tun", "tap",
	"tailscale", "zt", "wg", "ppp", "hyper-v", "vethernet", "wsl"}

// Prefijos que cuentan como VPN para el aviso (subconjunto de los virtuales).
var vpnPrefixes = []string{"utun", "tun", "tailscale", "wg", "ppp"}

func hasPrefix(name string, ps []string) bool {
	n := strings.ToLower(name)
	for _, p := range ps {
		if strings.HasPrefix(n, p) {
			return true
		}
	}
	return false
}

var (
	_, net192, _ = net.ParseCIDR("192.168.0.0/16")
	_, net10, _  = net.ParseCIDR("10.0.0.0/8")
	_, net172, _ = net.ParseCIDR("172.16.0.0/12")
)

func (l *LAN) Candidates() []Candidate {
	ifs, err := l.List()
	if err != nil {
		return nil
	}
	route := l.Route()
	var out []Candidate
	for _, in := range ifs {
		if !in.Up || in.Loopback {
			continue
		}
		for _, ip := range in.IPs {
			v4 := ip.To4()
			if v4 == nil || v4.IsLinkLocalUnicast() || v4.IsLoopback() {
				continue
			}
			c := Candidate{Name: in.Name, Human: human(in.Name), IP: v4.String(), Virtual: hasPrefix(in.Name, virtualPrefixes)}
			switch {
			case net192.Contains(v4):
				c.Score = 3
			case net10.Contains(v4), net172.Contains(v4):
				c.Score = 2
			default:
				c.Score = 1
			}
			if route != nil && route.Equal(v4) {
				c.Score++
			}
			if n := strings.ToLower(in.Name); strings.HasPrefix(n, "wl") || strings.Contains(n, "wi-fi") ||
				strings.Contains(n, "wlan") || n == "en0" || strings.Contains(n, "ethernet") && !c.Virtual {
				c.Score++
			}
			if c.Virtual {
				c.Score -= 10
			}
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		ri, rj := route != nil && route.String() == out[i].IP, route != nil && route.String() == out[j].IP
		if ri != rj {
			return ri
		}
		return out[i].Name < out[j].Name
	})
	l.mu.Lock()
	ov := l.override
	l.mu.Unlock()
	for i, c := range out {
		if c.IP == ov && i > 0 {
			out = append([]Candidate{c}, append(out[:i:i], out[i+1:]...)...)
			break
		}
	}
	return out
}

func human(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.HasPrefix(n, "tailscale"):
		return "Tailscale"
	case strings.HasPrefix(n, "wl"), strings.Contains(n, "wi-fi"), strings.Contains(n, "wlan"), n == "en0":
		return "Wi-Fi"
	case strings.HasPrefix(n, "eth"), strings.HasPrefix(n, "en"), n == "ethernet":
		return "Ethernet"
	}
	return name
}

func (l *LAN) VPN() bool {
	ifs, err := l.List()
	if err != nil {
		return false
	}
	for _, in := range ifs {
		if !in.Up || !hasPrefix(in.Name, vpnPrefixes) {
			continue
		}
		for _, ip := range in.IPs {
			if ip.To4() != nil {
				return true
			}
		}
	}
	return false
}

func (l *LAN) Addresses(_ context.Context, sessionPath string) ([]Address, error) {
	cs := l.Candidates()
	if len(cs) == 0 {
		return nil, nil
	}
	hp := net.JoinHostPort(cs[0].IP, strconv.Itoa(l.Port))
	return []Address{{Kind: "lan", URL: "http://" + hp + sessionPath, Display: hp, Label: "En esta red WiFi"}}, nil
}

func systemIfaces() ([]Iface, error) {
	sys, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var out []Iface
	for _, s := range sys {
		in := Iface{Name: s.Name, Up: s.Flags&net.FlagUp != 0, Loopback: s.Flags&net.FlagLoopback != 0}
		addrs, _ := s.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok {
				in.IPs = append(in.IPs, ipn.IP)
			}
		}
		out = append(out, in)
	}
	return out, nil
}

// routeIP pregunta al OS qué IP usaría para salir a internet. UDP "connect" no manda paquetes.
func routeIP() net.IP {
	c, err := net.Dial("udp", "1.1.1.1:53")
	if err != nil {
		return nil
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).IP
}
```

Cálculo de los escenarios, para quien depure: "Ethernet + WiFi misma red" empata en 4 (`en0`: 3 + nombre; `en7`: 3 + ruta) y desempata la ruta → `en7`. "WiFi + VPN full-tunnel": `utun4` = 2 + 1 (ruta) − 10 = −7; `en0` = 3 + 1 = 4.

- [ ] **Step 4: Verificar que pasa**

Run: `go test -race ./internal/addr/`
Expected: `ok`.

- [ ] **Step 5: Probar contra la máquina real**

Agregar temporalmente (no commitear) `internal/addr/real_test.go`:
```go
package addr

import "testing"

func TestRealMachine(t *testing.T) { t.Logf("%+v VPN=%v", NewLAN(8080).Candidates(), NewLAN(8080).VPN()) }
```
Run: `go test ./internal/addr/ -run RealMachine -v`
Expected: la primera candidata es la IP del WiFi de esta Mac (comparar con Ajustes → Wi-Fi → Detalles). Borrar el archivo.

- [ ] **Step 6: Commit**

```bash
git add internal/addr/lan.go internal/addr/lan_test.go
git commit -m "feat(addr): proveedor LAN con puntaje de interfaces, override y aviso de VPN

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 13: QR como SVG

Spec: §4.7, §9.1 (fila `qr`), §13 (tamaño, quiet zone, corrección `M`).

**Files:**
- Create: `internal/qr/qr.go`
- Test: `internal/qr/qr_test.go`

**Interfaces:**
- Consumes: nada.
- Produces: `qr.SVG(url string) (string, error)` — `<svg>` con `viewBox` en módulos, `shape-rendering="crispEdges"`, fondo blanco incluido (quiet zone de 4 módulos) y sin `width`/`height` fijos (el CSS de la Tarea 18 lo dimensiona entre 280 y 480 px).

- [ ] **Step 1: Agregar dependencias**

```bash
go get github.com/skip2/go-qrcode@latest
go get github.com/makiuchi-d/gozxing@latest   # solo lo importa el _test
```

- [ ] **Step 2: Test que falla**

`internal/qr/qr_test.go`:
```go
package qr

import (
	"image"
	"image/color"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/makiuchi-d/gozxing"
	gzqr "github.com/makiuchi-d/gozxing/qrcode"
)

// rasterize convierte el SVG (solo <rect>) en una imagen para decodificarla.
func rasterize(t *testing.T, svg string, scale int) image.Image {
	t.Helper()
	m := regexp.MustCompile(`viewBox="0 0 (\d+) (\d+)"`).FindStringSubmatch(svg)
	if m == nil {
		t.Fatal("sin viewBox")
	}
	n, _ := strconv.Atoi(m[1])
	img := image.NewGray(image.Rect(0, 0, n*scale, n*scale))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	re := regexp.MustCompile(`<rect x="(\d+)" y="(\d+)" width="(\d+)" height="1"/>`)
	for _, r := range re.FindAllStringSubmatch(svg, -1) {
		x, _ := strconv.Atoi(r[1])
		y, _ := strconv.Atoi(r[2])
		w, _ := strconv.Atoi(r[3])
		for yy := y * scale; yy < (y+1)*scale; yy++ {
			for xx := x * scale; xx < (x+w)*scale; xx++ {
				img.Set(xx, yy, color.Gray{0})
			}
		}
	}
	return img
}

func TestSVGDecodesToURL(t *testing.T) {
	for _, url := range []string{
		"http://192.168.1.42:8080/s/k3x9m",
		"https://amber-cat-dream-longer-name.trycloudflare.com/s/k3x9m?pin=4813",
	} {
		svg, err := SVG(url)
		if err != nil {
			t.Fatal(err)
		}
		// Sin width/height en el <svg>: lo dimensiona el CSS.
		if !strings.HasPrefix(svg, "<svg xmlns") || !strings.Contains(svg, `shape-rendering="crispEdges"`) ||
			strings.HasPrefix(svg, "<svg width") {
			t.Fatalf("SVG mal formado: %.120s", svg)
		}
		bmp, _ := gozxing.NewBinaryBitmapFromImage(rasterize(t, svg, 8))
		res, err := gzqr.NewQRCodeReader().Decode(bmp, nil)
		if err != nil {
			t.Fatalf("no decodifica %q: %v", url, err)
		}
		if res.GetText() != url {
			t.Fatalf("decodificó %q", res.GetText())
		}
	}
}

func TestSVGHasQuietZone(t *testing.T) {
	svg, _ := SVG("http://10.0.0.2:8080/s/abcde")
	// Ningún módulo negro en las primeras 4 filas/columnas.
	re := regexp.MustCompile(`<rect x="(\d+)" y="(\d+)" width="\d+" height="1"/>`)
	for _, r := range re.FindAllStringSubmatch(svg, -1) {
		x, _ := strconv.Atoi(r[1])
		y, _ := strconv.Atoi(r[2])
		if x < 4 || y < 4 {
			t.Fatalf("módulo en (%d,%d): falta quiet zone", x, y)
		}
	}
}
```

- [ ] **Step 3: Verificar que falla**

Run: `go test ./internal/qr/`
Expected: FAIL — `undefined: SVG`.

- [ ] **Step 4: Implementar `internal/qr/qr.go`**

```go
// Package qr dibuja URLs como SVG. Sin red, sin exec.
package qr

import (
	"fmt"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

const quiet = 4

// SVG devuelve un <svg> escalable. Une módulos negros contiguos de cada fila en un solo <rect>
// para que el SVG pese pocos KB aun con URLs largas.
func SVG(url string) (string, error) {
	q, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return "", err
	}
	q.DisableBorder = true
	bm := q.Bitmap()
	n := len(bm) + 2*quiet
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges" role="img" aria-label="Código QR">`, n, n)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#fff"/><g fill="#000">`, n, n)
	for y, row := range bm {
		for x := 0; x < len(row); {
			if !row[x] {
				x++
				continue
			}
			start := x
			for x < len(row) && row[x] {
				x++
			}
			fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="1"/>`, start+quiet, y+quiet, x-start)
		}
	}
	b.WriteString(`</g></svg>`)
	return b.String(), nil
}
```
El `<rect>` del fondo no matchea la regex del test (`height="1"` exige alto 1), así que el test solo ve módulos.

- [ ] **Step 5: Verificar que pasa**

Run: `go mod tidy && go test -race ./internal/qr/`
Expected: `ok`.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/qr
git commit -m "feat(qr): render de URL a SVG con quiet zone y corrección M

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 14: Plataforma — config, abrir carpeta y firewall

Spec: §4.10 (configuración persistente, logs), §6.1 (aviso de firewall en primera ejecución), §6.2 ("Revisar permiso de Windows"), §13 (directorio de configuración).

**Files:**
- Create: `internal/platform/configdir.go`, `internal/platform/config.go`, `internal/platform/openfolder.go`
- Create: `internal/platform/firewall_darwin.go`, `internal/platform/firewall_windows.go`, `internal/platform/firewall_other.go`
- Test: `internal/platform/platform_test.go`

**Interfaces:**
- Consumes: nada.
- Produces:
  ```go
  func ConfigDir() (string, error)                       // os.UserConfigDir()+"/Pasame", creado
  type Config struct {
      Name          string `json:"name,omitempty"`
      IfaceOverride string `json:"iface_override,omitempty"`
      PortOverride  int    `json:"port_override,omitempty"`
      Lang          string `json:"lang,omitempty"`
  }
  func LoadConfig(dir string) (cfg Config, firstRun bool, err error) // firstRun = no existía config.json
  func SaveConfig(dir string, c Config) error                        // escritura atómica (tmp + rename)
  func OpenFolder(path string) error
  func FirewallHint() string   // "windows" | "mac" | "" → qué aviso de permiso mostrar
  func OpenFirewallSettings() error // solo Windows; en el resto devuelve nil
  func openCommand(path string) (name string, args []string)       // testeable
  ```

- [ ] **Step 1: Test que falla**

`internal/platform/platform_test.go`:
```go
package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c, first, err := LoadConfig(dir)
	if err != nil || !first || c != (Config{}) {
		t.Fatalf("%+v %v %v", c, first, err)
	}
	want := Config{Name: "Martín", IfaceOverride: "192.168.1.42", PortOverride: 9000}
	if err := SaveConfig(dir, want); err != nil {
		t.Fatal(err)
	}
	got, first, err := LoadConfig(dir)
	if err != nil || first || got != want {
		t.Fatalf("%+v %v %v", got, first, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json.tmp")); err == nil {
		t.Fatal("quedó el temporal")
	}
}

func TestLoadConfigCorruptIsNotFatal(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.json"), []byte("{no es json"), 0o644)
	c, first, err := LoadConfig(dir)
	if err != nil || first || c != (Config{}) {
		t.Fatalf("un config roto no debe impedir arrancar: %+v %v %v", c, first, err)
	}
}

func TestConfigDirEndsInPasame(t *testing.T) {
	d, err := ConfigDir()
	if err != nil {
		t.Skip("sin UserConfigDir en este entorno:", err)
	}
	if filepath.Base(d) != "Pasame" {
		t.Fatalf("%q", d)
	}
}

func TestOpenCommand(t *testing.T) {
	name, args := openCommand("/tmp/x")
	want := map[string]string{"darwin": "open", "windows": "explorer", "linux": "xdg-open"}[runtime.GOOS]
	if name != want || len(args) != 1 || args[0] != "/tmp/x" {
		t.Fatalf("%s %v", name, args)
	}
}
```

- [ ] **Step 2: Verificar que falla**

Run: `go test ./internal/platform/`
Expected: FAIL — `undefined: LoadConfig`, …

- [ ] **Step 3: Implementar**

`internal/platform/configdir.go`:
```go
// Package platform aísla lo que cambia por sistema operativo.
package platform

import (
	"os"
	"path/filepath"
)

func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(base, "Pasame")
	return d, os.MkdirAll(d, 0o755)
}
```

`internal/platform/config.go`:
```go
package platform

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	Name          string `json:"name,omitempty"`
	IfaceOverride string `json:"iface_override,omitempty"`
	PortOverride  int    `json:"port_override,omitempty"`
	Lang          string `json:"lang,omitempty"`
}

func LoadConfig(dir string) (Config, bool, error) {
	var c Config
	b, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return c, true, nil
	}
	if err != nil {
		return c, false, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		log.Printf("config.json ilegible, uso valores por defecto: %v", err)
		return Config{}, false, nil
	}
	return c, false, nil
}

func SaveConfig(dir string, c Config) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "config.json.tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, "config.json"))
}
```

`internal/platform/openfolder.go`:
```go
package platform

import (
	"os/exec"
	"runtime"
)

func openCommand(path string) (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{path}
	case "windows":
		return "explorer", []string{path}
	default:
		return "xdg-open", []string{path}
	}
}

// OpenFolder abre la carpeta en Finder / Explorador / el gestor de archivos.
func OpenFolder(path string) error {
	name, args := openCommand(path)
	return exec.Command(name, args...).Start()
}
```

`internal/platform/firewall_darwin.go`:
```go
//go:build darwin

package platform

import (
	"os/exec"
	"strings"
)

// FirewallHint devuelve "mac" solo si el firewall de macOS está activado (viene apagado de fábrica).
func FirewallHint() string {
	out, err := exec.Command("defaults", "read", "/Library/Preferences/com.apple.alf", "globalstate").Output()
	if err == nil && strings.TrimSpace(string(out)) != "0" {
		return "mac"
	}
	return ""
}

func OpenFirewallSettings() error { return nil }
```

`internal/platform/firewall_windows.go`:
```go
//go:build windows

package platform

import "os/exec"

// FirewallHint: en Windows siempre puede aparecer el diálogo; core decide mostrarlo solo en la primera ejecución.
func FirewallHint() string { return "windows" }

// OpenFirewallSettings abre "Permitir aplicaciones a través del Firewall de Windows".
func OpenFirewallSettings() error { return exec.Command("control", "firewall.cpl").Start() }
```

`internal/platform/firewall_other.go`:
```go
//go:build !darwin && !windows

package platform

func FirewallHint() string        { return "" }
func OpenFirewallSettings() error { return nil }
```

- [ ] **Step 4: Verificar que pasa**

Run: `go test -race ./internal/platform/ && make cross`
Expected: `ok` y los 7 targets compilan.

- [ ] **Step 5: Commit**

```bash
git add internal/platform
git commit -m "feat(platform): config persistente, abrir carpeta y avisos de firewall por OS

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 15: Diálogo nativo de archivos y apertura del navegador

Spec: §3.2 (cómo se abre el navegador), §4.9 (tres formas de entrada; nota de macOS), §7 (sin navegador, sin diálogo).

**Files:**
- Create: `internal/dialog/dialog.go`, `internal/browser/open.go`
- Test: `internal/browser/open_test.go`, `internal/dialog/dialog_test.go`

**Interfaces:**
- Consumes: nada.
- Produces:
  ```go
  // package dialog
  var ErrCanceled = errors.New("dialog: cancelado")
  var ErrUnsupported = errors.New("dialog: no hay diálogo nativo")
  func PickFiles() ([]string, error)
  func PickFolder() (string, error)
  // package browser
  func Open(url string) error      // si falla, imprime la URL y muestra zenity.Info con ella
  func command(url string) (string, []string)
  ```

- [ ] **Step 1: Agregar dependencia**

Run: `go get github.com/ncruces/zenity@latest`

- [ ] **Step 2: Tests que fallan**

`internal/browser/open_test.go`:
```go
package browser

import (
	"runtime"
	"testing"
)

func TestCommand(t *testing.T) {
	u := "http://127.0.0.1:5555/?t=abc"
	name, args := command(u)
	switch runtime.GOOS {
	case "darwin":
		if name != "open" || args[0] != u {
			t.Fatal(name, args)
		}
	case "windows":
		if name != "rundll32" || args[0] != "url.dll,FileProtocolHandler" || args[1] != u {
			t.Fatal(name, args)
		}
	default:
		if name != "xdg-open" || args[0] != u {
			t.Fatal(name, args)
		}
	}
}
```

`internal/dialog/dialog_test.go` (el diálogo real es prueba manual; acá solo se fija el mapeo de errores):
```go
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
```

- [ ] **Step 3: Verificar que falla**

Run: `go test ./internal/browser/ ./internal/dialog/`
Expected: FAIL — `undefined: command`, `mapErr`.

- [ ] **Step 4: Implementar**

`internal/dialog/dialog.go`:
```go
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
```

`internal/browser/open.go`:
```go
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
		fmt.Fprintln(os.Stderr, "Abrí esta dirección en tu navegador:", url)
		zenity.Info("Abrí esta dirección en tu navegador:\n\n"+url, zenity.Title("Pasame"))
	}
	return err
}
```

- [ ] **Step 5: Verificar que pasa**

Run: `go mod tidy && go test -race ./internal/browser/ ./internal/dialog/ && make cross`
Expected: `ok` y los 7 targets compilan (confirma que `zenity` no arrastra CGO).

- [ ] **Step 6: Prueba manual del diálogo en esta Mac**

`internal/dialog/manual_test.go` temporal (no commitear):
```go
package dialog

import "testing"

func TestManualPick(t *testing.T) { p, err := PickFiles(); t.Log(p, err) }
```
Run: `go test ./internal/dialog/ -run Manual -v`
Expected: aparece el selector de macOS **adelante** de la terminal; elegir 2 archivos → se imprimen las rutas; Cancelar → `dialog: cancelado`. Anotar en `test/manual/matriz.md` (Tarea 22) si apareció detrás. Borrar el archivo.

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum internal/dialog internal/browser
git commit -m "feat(dialog,browser): selector nativo sin CGO y apertura del navegador con fallback

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 16: `core` — orquestación, `State` y regla de inactividad

Spec: §4.1 (core conoce a todos), §4.3 (cambiar archivos = nueva sesión), §4.5.1 (re-evaluación cada 5 s, aviso de cambio de red), §4.9 (tres entradas convergen en `session.New`), §4.10 (regla de 2 min), §6.1–6.2 (qué muestra la UI).

**Decisión de implementación (desvío menor del spec §4.8):** el SVG del QR viaja **dentro** de `State.QR` por SSE en lugar de pedirse a `GET /api/qr.svg?url=`. Motivo: un `<img src>` no puede mandar el header `X-Pasame-Token`, y meter el token en la query de una imagen lo expone en el historial. Resultado: un endpoint menos y cero superficie extra. El SVG pesa 3–6 KB; se reenvía solo cuando cambia el estado.

**Files:**
- Create: `internal/core/state.go`, `internal/core/core.go`, `internal/core/lifecycle.go`
- Test: `internal/core/core_test.go`, `internal/core/lifecycle_test.go`

**Interfaces:**
- Consumes: `session.New/Stats`, `addr.Book/LAN/Candidate`, `qr.SVG`, `platform.Config/SaveConfig/FirewallHint`, `dialog.ErrCanceled/ErrUnsupported`, `i18n.Size`.
- Produces:
  ```go
  type Options struct {
      SharePort  int
      Quarantine string
      ConfigDir  string
      Config     platform.Config
      FirstRun   bool
      LAN        *addr.LAN
      Book       *addr.Book                      // con LAN ya registrado
      Pick       func(kind string) ([]string, error) // "files" | "folder"; por defecto dialog.*
      Version    string
  }
  func New(o Options) *Core
  // Para share.Deps:
  func (c *Core) Current() *session.Session
  func (c *Core) Stats() *session.Stats
  func (c *Core) Sender() string
  func (c *Core) Strict() bool
  // Acciones (las llama control):
  func (c *Core) Share(paths []string)
  func (c *Core) ReceiveOnly()
  func (c *Core) Pick(kind string)          // asíncrono
  func (c *Core) Stop()
  func (c *Core) SetName(name string) error
  func (c *Core) SetIface(ip string) error
  func (c *Core) State() State
  func (c *Core) Subscribe() (<-chan struct{}, func())
  func (c *Core) ClientConnected() (release func()) // una pestaña de control abierta (SSE)
  func (c *Core) Run(ctx context.Context)            // bucle de 5 s + notificaciones del Book
  func (c *Core) RequestQuit()
  func (c *Core) Done() <-chan struct{}              // se cierra cuando la app debe salir
  // Para tests:
  func (c *Core) tick(now time.Time)
  ```
  `State` (JSON que consume `app.js`, Tarea 18):
  ```go
  type State struct {
      Phase          string          `json:"phase"`          // "idle" | "picking" | "sharing"
      Name           string          `json:"name"`
      Files          []FileView      `json:"files"`
      Count          int             `json:"count"`
      Total          string          `json:"total"`          // "1,2 GB"
      ReceiveOnly    bool            `json:"receiveOnly"`
      Unreadable     []string        `json:"unreadable"`
      Addresses      []addr.Address  `json:"addresses"`
      QR             string          `json:"qr"`             // SVG de la primaria
      SharedAt       int64           `json:"sharedAt"`       // unix ms; el JS calcula el hint de 45 s
      Stats          session.Snapshot `json:"stats"`
      VPN            bool            `json:"vpn"`
      Ifaces         []addr.Candidate `json:"ifaces"`
      NetChanged     bool            `json:"netChanged"`
      FirewallHint   string          `json:"firewallHint"`   // "windows" | "mac" | ""
      PickUnsupported bool           `json:"pickUnsupported"`
      Strict         bool            `json:"strict"`
      Quarantine     string          `json:"quarantine"`
      Version        string          `json:"version"`
  }
  type FileView struct { Name string `json:"name"`; Size string `json:"size"` }
  ```

- [ ] **Step 1: Tests de estado y acciones (fallan)**

`internal/core/core_test.go`:
```go
package core

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KonixDev/pasame/internal/addr"
	"github.com/KonixDev/pasame/internal/dialog"
	"github.com/KonixDev/pasame/internal/platform"
)

type env struct {
	c     *Core
	dir   string
	cfg   string
	ifs   []addr.Iface
	pick  func(string) ([]string, error)
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{dir: t.TempDir(), cfg: t.TempDir()}
	e.ifs = []addr.Iface{{Name: "en0", Up: true, IPs: []net.IP{net.ParseIP("192.168.1.42")}}}
	lan := addr.NewLAN(8080)
	lan.List = func() ([]addr.Iface, error) { return e.ifs, nil }
	lan.Route = func() net.IP { return nil }
	book := addr.NewBook()
	book.Register(lan)
	e.c = New(Options{
		SharePort: 8080, Quarantine: t.TempDir(), ConfigDir: e.cfg,
		Config: platform.Config{Name: "Martín"}, LAN: lan, Book: book,
		Pick: func(k string) ([]string, error) { return e.pick(k) },
	})
	return e
}

func (e *env) file(t *testing.T, name string, n int) string {
	p := filepath.Join(e.dir, name)
	os.WriteFile(p, make([]byte, n), 0o644)
	return p
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timeout esperando condición")
}

func TestIdleState(t *testing.T) {
	e := newEnv(t)
	s := e.c.State()
	if s.Phase != "idle" || s.Name != "Martín" || e.c.Current() != nil || e.c.Strict() {
		t.Fatalf("%+v", s)
	}
}

func TestShare(t *testing.T) {
	e := newEnv(t)
	ch, cancel := e.c.Subscribe()
	defer cancel()
	e.c.Share([]string{e.file(t, "a.jpg", 1300), filepath.Join(e.dir, "no-existe.mp4")})
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("no notificó")
	}
	s := e.c.State()
	if s.Phase != "sharing" || s.Count != 1 || s.Total != "1,3 KB" || s.Files[0].Name != "a.jpg" {
		t.Fatalf("%+v", s)
	}
	if len(s.Unreadable) != 1 || s.Unreadable[0] != "no-existe.mp4" {
		t.Fatalf("Unreadable = %v", s.Unreadable)
	}
	tok := e.c.Current().Token
	if len(s.Addresses) != 1 || s.Addresses[0].URL != "http://192.168.1.42:8080/s/"+tok || !strings.HasPrefix(s.QR, "<svg") {
		t.Fatalf("%+v", s.Addresses)
	}
	if s.SharedAt == 0 {
		t.Fatal("SharedAt vacío")
	}
}

func TestShareAgainIsNewSession(t *testing.T) {
	e := newEnv(t)
	e.c.Share([]string{e.file(t, "a", 1)})
	old := e.c.Current().Token
	e.c.Share([]string{e.file(t, "b", 1)})
	if e.c.Current().Token == old {
		t.Fatal("reutilizó el token")
	}
}

func TestReceiveOnlyAndStop(t *testing.T) {
	e := newEnv(t)
	e.c.ReceiveOnly()
	if s := e.c.State(); s.Phase != "sharing" || !s.ReceiveOnly || s.Count != 0 {
		t.Fatalf("%+v", s)
	}
	e.c.Stop()
	if s := e.c.State(); s.Phase != "idle" || e.c.Current() != nil || s.QR != "" {
		t.Fatalf("%+v", s)
	}
}

func TestPickFlows(t *testing.T) {
	e := newEnv(t)
	p := e.file(t, "x.pdf", 5)
	e.pick = func(k string) ([]string, error) {
		if k != "files" {
			t.Errorf("kind = %q", k)
		}
		return []string{p}, nil
	}
	e.c.Pick("files")
	waitFor(t, func() bool { return e.c.State().Phase == "sharing" })

	e.c.Stop()
	e.pick = func(string) ([]string, error) { return nil, dialog.ErrCanceled }
	e.c.Pick("files")
	waitFor(t, func() bool { return e.c.State().Phase == "idle" })

	e.pick = func(string) ([]string, error) { return nil, dialog.ErrUnsupported }
	e.c.Pick("folder")
	waitFor(t, func() bool { return e.c.State().PickUnsupported })
}

func TestPickErrorKeepsCurrentSession(t *testing.T) {
	e := newEnv(t)
	e.c.Share([]string{e.file(t, "a", 1)})
	e.pick = func(string) ([]string, error) { return nil, errors.New("boom") }
	e.c.Pick("files")
	waitFor(t, func() bool { return e.c.State().Phase == "sharing" })
}

func TestSetNamePersists(t *testing.T) {
	e := newEnv(t)
	if err := e.c.SetName("  Abuela Rosa  "); err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := platform.LoadConfig(e.cfg)
	if cfg.Name != "Abuela Rosa" || e.c.Sender() != "Abuela Rosa" {
		t.Fatalf("%+v %q", cfg, e.c.Sender())
	}
	if err := e.c.SetName("   "); err == nil {
		t.Fatal("nombre vacío aceptado")
	}
}

func TestSetIface(t *testing.T) {
	e := newEnv(t)
	e.ifs = append(e.ifs, addr.Iface{Name: "tailscale0", Up: true, IPs: []net.IP{net.ParseIP("100.64.0.9")}})
	e.c.Share([]string{e.file(t, "a", 1)})
	if err := e.c.SetIface("100.64.0.9"); err != nil {
		t.Fatal(err)
	}
	if s := e.c.State(); s.Addresses[0].Display != "100.64.0.9:8080" {
		t.Fatalf("%+v", s.Addresses)
	}
	if err := e.c.SetIface("1.2.3.4"); err == nil {
		t.Fatal("aceptó una IP que no es de esta compu")
	}
	cfg, _, _ := platform.LoadConfig(e.cfg)
	if cfg.IfaceOverride != "100.64.0.9" {
		t.Fatalf("%+v", cfg)
	}
}
```

- [ ] **Step 2: Verificar que falla**

Run: `go test ./internal/core/`
Expected: FAIL — `undefined: New`, `Options`, …

- [ ] **Step 3: Implementar `state.go` y `core.go`**

`internal/core/state.go`:
```go
package core

import (
	"github.com/KonixDev/pasame/internal/addr"
	"github.com/KonixDev/pasame/internal/session"
)

type FileView struct {
	Name string `json:"name"`
	Size string `json:"size"`
}

type State struct {
	Phase           string           `json:"phase"`
	Name            string           `json:"name"`
	Files           []FileView       `json:"files"`
	Count           int              `json:"count"`
	Total           string           `json:"total"`
	ReceiveOnly     bool             `json:"receiveOnly"`
	Unreadable      []string         `json:"unreadable"`
	Addresses       []addr.Address   `json:"addresses"`
	QR              string           `json:"qr"`
	SharedAt        int64            `json:"sharedAt"`
	Stats           session.Snapshot `json:"stats"`
	VPN             bool             `json:"vpn"`
	Ifaces          []addr.Candidate `json:"ifaces"`
	NetChanged      bool             `json:"netChanged"`
	FirewallHint    string           `json:"firewallHint"`
	PickUnsupported bool             `json:"pickUnsupported"`
	Strict          bool             `json:"strict"`
	Quarantine      string           `json:"quarantine"`
	Version         string           `json:"version"`
}
```

`internal/core/core.go`:
```go
// Package core es el único que conoce a todos: sesión, direcciones, stats y (Plan 2) túnel.
package core

import (
	"context"
	"errors"
	"io/fs"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/KonixDev/pasame/internal/addr"
	"github.com/KonixDev/pasame/internal/dialog"
	"github.com/KonixDev/pasame/internal/i18n"
	"github.com/KonixDev/pasame/internal/platform"
	"github.com/KonixDev/pasame/internal/qr"
	"github.com/KonixDev/pasame/internal/session"
)

type Options struct {
	SharePort  int
	Quarantine string
	ConfigDir  string
	Config     platform.Config
	FirstRun   bool
	LAN        *addr.LAN
	Book       *addr.Book
	Pick       func(kind string) ([]string, error)
	Version    string
}

type Core struct {
	o Options

	mu              sync.Mutex
	cfg             platform.Config
	phase           string
	sess            *session.Session
	stats           *session.Stats
	unreadable      []string
	sharedAt        time.Time
	pickUnsupported bool
	strict          bool
	firewallHint    string
	netChangedUntil time.Time
	lastPrimary     string

	subsMu sync.Mutex
	subs   map[chan struct{}]bool

	// ciclo de vida (lifecycle.go)
	clients   int
	idleSince time.Time
	done      chan struct{}
	quitOnce  sync.Once
}

func New(o Options) *Core {
	if o.Pick == nil {
		o.Pick = defaultPick
	}
	c := &Core{o: o, cfg: o.Config, phase: "idle", subs: map[chan struct{}]bool{}, done: make(chan struct{})}
	// Se calcula una vez: en macOS es un exec de `defaults`, no algo para cada push de SSE.
	// En Windows el aviso solo tiene sentido la primera vez (después el permiso ya se dio o se negó).
	if h := platform.FirewallHint(); h == "mac" || (h == "windows" && o.FirstRun) {
		c.firewallHint = h
	}
	if o.Config.IfaceOverride != "" {
		o.LAN.SetOverride(o.Config.IfaceOverride)
	}
	return c
}

func defaultPick(kind string) ([]string, error) {
	if kind == "folder" {
		p, err := dialog.PickFolder()
		return []string{p}, err
	}
	return dialog.PickFiles()
}

// --- lectura para share.Deps ---

func (c *Core) Current() *session.Session { c.mu.Lock(); defer c.mu.Unlock(); return c.sess }
func (c *Core) Stats() *session.Stats     { c.mu.Lock(); defer c.mu.Unlock(); return c.stats }
func (c *Core) Strict() bool              { c.mu.Lock(); defer c.mu.Unlock(); return c.strict }

func (c *Core) Sender() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Name != "" {
		return c.cfg.Name
	}
	return defaultName()
}

func defaultName() string {
	u, err := user.Current()
	if err != nil {
		return "Alguien"
	}
	n := strings.TrimSpace(u.Name)
	if n == "" {
		n = u.Username
	}
	fields := strings.Fields(n) // solo el primer nombre: "Martín", no "Martín Coll"
	if len(fields) == 0 {
		return "Alguien"
	}
	r := []rune(fields[0])
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// --- acciones ---

func (c *Core) Share(paths []string) {
	s, errs := session.New(paths)
	var bad []string
	for _, err := range errs {
		var pe *fs.PathError
		if errors.As(err, &pe) {
			bad = append(bad, filepath.Base(pe.Path))
		} else {
			bad = append(bad, err.Error())
		}
	}
	c.mu.Lock()
	c.sess, c.stats = s, session.NewStats(c.notify)
	c.unreadable, c.sharedAt, c.phase, c.pickUnsupported = bad, time.Now(), "sharing", false
	c.mu.Unlock()
	c.notify()
}

func (c *Core) ReceiveOnly() { c.Share(nil) }

func (c *Core) Stop() {
	c.mu.Lock()
	c.sess, c.stats, c.unreadable, c.phase = nil, nil, nil, "idle"
	c.mu.Unlock()
	c.notify()
}

// Pick abre el diálogo en un goroutine: el request HTTP que lo pidió vuelve enseguida (202).
func (c *Core) Pick(kind string) {
	c.mu.Lock()
	prev := c.phase
	c.phase = "picking"
	c.mu.Unlock()
	c.notify()
	go func() {
		paths, err := c.o.Pick(kind)
		switch {
		case err == nil && len(paths) > 0 && paths[0] != "":
			c.Share(paths)
			return
		case errors.Is(err, dialog.ErrUnsupported):
			c.mu.Lock()
			c.pickUnsupported = true
			c.mu.Unlock()
		}
		c.mu.Lock()
		c.phase = prev
		c.mu.Unlock()
		c.notify()
	}()
}

func (c *Core) SetName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 60 {
		return errors.New("nombre inválido")
	}
	c.mu.Lock()
	c.cfg.Name = name
	cfg := c.cfg
	c.mu.Unlock()
	c.notify()
	return platform.SaveConfig(c.o.ConfigDir, cfg)
}

func (c *Core) SetIface(ip string) error {
	found := ip == ""
	for _, cand := range c.o.LAN.Candidates() {
		found = found || cand.IP == ip
	}
	if !found {
		return errors.New("esa dirección no es de esta computadora")
	}
	c.o.LAN.SetOverride(ip)
	c.mu.Lock()
	c.cfg.IfaceOverride = ip
	cfg := c.cfg
	c.mu.Unlock()
	c.o.Book.Notify()
	c.notify()
	return platform.SaveConfig(c.o.ConfigDir, cfg)
}

// --- estado y suscripciones ---

func (c *Core) State() State {
	c.mu.Lock()
	sess, stats := c.sess, c.stats
	st := State{
		Phase: c.phase, Name: c.nameLocked(), Unreadable: c.unreadable,
		PickUnsupported: c.pickUnsupported, Strict: c.strict, Quarantine: c.o.Quarantine,
		Version: c.o.Version, NetChanged: time.Now().Before(c.netChangedUntil),
		FirewallHint: c.firewallHint,
	}
	if sess != nil {
		st.SharedAt = c.sharedAt.UnixMilli()
	}
	c.mu.Unlock()

	st.VPN = c.o.LAN.VPN()
	st.Ifaces = c.o.LAN.Candidates()
	if sess == nil {
		return st
	}
	st.Count, st.ReceiveOnly = len(sess.Files), len(sess.Files) == 0
	st.Total = i18n.Size(i18n.ES, sess.TotalSize())
	for _, f := range sess.Files {
		st.Files = append(st.Files, FileView{Name: f.Rel, Size: i18n.Size(i18n.ES, f.Size)})
	}
	st.Stats = stats.Snapshot()
	st.Addresses = c.o.Book.Active(context.Background(), c.sessionPath(sess))
	if len(st.Addresses) > 0 {
		st.QR, _ = qr.SVG(st.Addresses[0].URL)
	}
	return st
}

// sessionPath es lo que cada proveedor agrega a su base. En modo estricto (Plan 2) suma "?pin=".
func (c *Core) sessionPath(s *session.Session) string { return s.Path() }

func (c *Core) nameLocked() string {
	if c.cfg.Name != "" {
		return c.cfg.Name
	}
	return defaultName()
}

func (c *Core) Subscribe() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	c.subsMu.Lock()
	c.subs[ch] = true
	c.subsMu.Unlock()
	return ch, func() {
		c.subsMu.Lock()
		delete(c.subs, ch)
		c.subsMu.Unlock()
	}
}

func (c *Core) notify() {
	c.subsMu.Lock()
	defer c.subsMu.Unlock()
	for ch := range c.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
```

- [ ] **Step 4: Verificar que pasa**

Run: `go test -race ./internal/core/`
Expected: `ok`. (`core.go` ya declara los campos `clients`, `idleSince`, `done` y `quitOnce`; sus métodos llegan en el Step 7.)

- [ ] **Step 5: Tests del ciclo de vida (fallan)**

`internal/core/lifecycle_test.go`:
```go
package core

import (
	"net"
	"testing"
	"time"

	"github.com/KonixDev/pasame/internal/addr"
)

func closed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestIdleRuleQuitsAfter120s(t *testing.T) {
	e := newEnv(t)
	t0 := time.Now()
	e.c.tick(t0)
	e.c.tick(t0.Add(119 * time.Second))
	if closed(e.c.Done()) {
		t.Fatal("salió antes de tiempo")
	}
	e.c.tick(t0.Add(121 * time.Second))
	if !closed(e.c.Done()) {
		t.Fatal("no salió a los 120 s")
	}
}

func TestIdleRuleWaitsForTab(t *testing.T) {
	e := newEnv(t)
	release := e.c.ClientConnected()
	t0 := time.Now()
	e.c.tick(t0)
	e.c.tick(t0.Add(10 * time.Minute))
	if closed(e.c.Done()) {
		t.Fatal("salió con la pestaña abierta")
	}
	release()
	e.c.tick(t0.Add(11 * time.Minute))
	e.c.tick(t0.Add(13 * time.Minute))
	if !closed(e.c.Done()) {
		t.Fatal("no salió después de cerrar la pestaña")
	}
}

func TestIdleRuleWaitsForTransfers(t *testing.T) {
	e := newEnv(t)
	e.c.Share([]string{e.file(t, "a", 1)})
	e.c.Stats().DownloadStarted(0)
	t0 := time.Now()
	e.c.tick(t0)
	e.c.tick(t0.Add(10 * time.Minute))
	if closed(e.c.Done()) {
		t.Fatal("salió con una descarga en curso")
	}
	e.c.Stats().DownloadDone(0, true)
	e.c.tick(t0.Add(11 * time.Minute))
	e.c.tick(t0.Add(13*time.Minute + time.Second))
	if !closed(e.c.Done()) {
		t.Fatal("no salió al terminar la descarga")
	}
}

func TestNetworkChangeIsFlagged(t *testing.T) {
	e := newEnv(t)
	e.c.Share([]string{e.file(t, "a", 1)})
	t0 := time.Now()
	e.c.tick(t0)
	e.ifs = []addr.Iface{{Name: "en0", Up: true, IPs: []net.IP{net.ParseIP("10.0.0.7")}}}
	ch, cancel := e.c.Subscribe()
	defer cancel()
	e.c.tick(t0.Add(5 * time.Second))
	if !closed(ch) || !e.c.State().NetChanged || e.c.State().Addresses[0].Display != "10.0.0.7:8080" {
		t.Fatalf("%+v", e.c.State())
	}
}

func TestRequestQuitIsIdempotent(t *testing.T) {
	e := newEnv(t)
	e.c.RequestQuit()
	e.c.RequestQuit()
	if !closed(e.c.Done()) {
		t.Fatal("Done no se cerró")
	}
}
```

- [ ] **Step 6: Verificar que falla**

Run: `go test ./internal/core/ -run 'Idle|Network|Quit'`
Expected: FAIL — `undefined: tick`, `ClientConnected`, `RequestQuit`.

- [ ] **Step 7: Implementar `internal/core/lifecycle.go`**

```go
package core

import (
	"context"
	"time"
)

const (
	idleLimit   = 120 * time.Second
	reevalEvery = 5 * time.Second
	netNotice   = 5 * time.Second
)

func (c *Core) Done() <-chan struct{} { return c.done }

func (c *Core) RequestQuit() { c.quitOnce.Do(func() { close(c.done) }) }

// ClientConnected marca una pestaña de control abierta (una conexión SSE viva).
func (c *Core) ClientConnected() func() {
	c.mu.Lock()
	c.clients++
	c.mu.Unlock()
	var once bool
	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if !once {
			once = true
			c.clients--
		}
	}
}

// Run hace el trabajo periódico hasta que ctx termine o la app deba salir.
func (c *Core) Run(ctx context.Context) {
	t := time.NewTicker(reevalEvery)
	defer t.Stop()
	c.tick(time.Now())
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case <-c.o.Book.Changed():
			c.notify()
		case now := <-t.C:
			c.tick(now)
		}
	}
}

func (c *Core) tick(now time.Time) {
	// 1) ¿Cambió la red? (apagaron la VPN, cambiaron de WiFi)
	primary := ""
	if cs := c.o.LAN.Candidates(); len(cs) > 0 {
		primary = cs[0].IP
	}
	c.mu.Lock()
	changed := c.lastPrimary != "" && primary != c.lastPrimary && c.sess != nil
	c.lastPrimary = primary
	if changed {
		c.netChangedUntil = now.Add(netNotice)
	}
	// 2) Regla de inactividad: sin pestaña y sin transferencias durante 120 s.
	busy := c.clients > 0 || (c.stats != nil && c.stats.InFlight() > 0)
	switch {
	case busy:
		c.idleSince = time.Time{}
	case c.idleSince.IsZero():
		c.idleSince = now
	}
	quit := !busy && now.Sub(c.idleSince) >= idleLimit
	c.mu.Unlock()

	if changed {
		c.o.Book.Notify()
		c.notify()
	}
	if quit {
		c.RequestQuit()
	}
}
```
`NetChanged` en `State()` compara contra `time.Now()`; en el test el `tick` usa `t0 + 5 s` con `t0 = time.Now()`, así que el aviso sigue vigente al leer el estado.

- [ ] **Step 8: Verificar que pasa**

Run: `go test -race ./internal/core/`
Expected: `ok`.

- [ ] **Step 9: Commit**

```bash
git add internal/core
git commit -m "feat(core): orquestación de sesión, direcciones y QR; regla de inactividad y cambio de red

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 17: Listener `control` — API local, token `t`, chequeo de `Host` y SSE

Spec: §4.2 (dos listeners), §4.8 (protección y tabla de API), §4.10 (handoff `POST /api/add`), §9.2 (fila "Listener control").

**Files:**
- Create: `internal/control/server.go`, `internal/control/api.go`, `internal/control/events.go`
- Create: `web/control/index.html` (versión mínima; la Tarea 18 la reemplaza entera)
- Modify: `web/embed.go` (agregar `Control`), `internal/core/core.go` (agregar `OpenFolder`, `OpenFirewall`)
- Test: `internal/control/control_test.go`

**Interfaces:**
- Consumes: todo `core.Core` público (Tarea 16).
- Produces:
  ```go
  // package control
  func New(c *core.Core, token string, port int) http.Handler
  func NewToken() string // 32 bytes aleatorios en hex
  // package web
  var Control embed.FS // control/index.html, control/app.css, control/app.js
  // package core (agregados)
  func (c *Core) OpenFolder() error   // abre la cuarentena
  func (c *Core) OpenFirewall() error // Windows: firewall.cpl
  ```
  API (todas `POST`, header `X-Pasame-Token: <t>`, cuerpo JSON, respuesta `204` o `202`):
  `/api/pick {"kind":"files"|"folder"}` → 202 · `/api/add {"paths":[...]}` · `/api/receive-only` · `/api/stop` · `/api/quit` · `/api/name {"name":"..."}` · `/api/iface {"ip":"..."}` · `/api/open-folder` · `/api/firewall`. Errores de validación → `400` con `{"error":"<texto para mostrar>"}`.
  SSE: `GET /events?t=<t>` → `event: state` + `data: <State JSON>` al conectar y en cada cambio; `event: ping` cada 5 s.

- [ ] **Step 1: Embebido de `control` y página mínima**

`web/control/index.html`:
```html
<!DOCTYPE html><html lang="es"><head><meta charset="utf-8"><title>Pasame</title></head><body>Pasame</body></html>
```
`web/embed.go`, agregar debajo de `Share`:
```go
//go:embed control
var Control embed.FS
```
En `internal/core/core.go`, agregar:
```go
func (c *Core) OpenFolder() error   { return platform.OpenFolder(c.o.Quarantine) }
func (c *Core) OpenFirewall() error { return platform.OpenFirewallSettings() }
```

- [ ] **Step 2: Test que falla**

`internal/control/control_test.go`:
```go
package control

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/KonixDev/pasame/internal/addr"
	"github.com/KonixDev/pasame/internal/core"
	"github.com/KonixDev/pasame/internal/platform"
)

const tok = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func newCore(t *testing.T) *core.Core {
	lan := addr.NewLAN(8080)
	lan.List = func() ([]addr.Iface, error) {
		return []addr.Iface{{Name: "en0", Up: true, IPs: []net.IP{net.ParseIP("192.168.1.42")}}}, nil
	}
	lan.Route = func() net.IP { return nil }
	book := addr.NewBook()
	book.Register(lan)
	return core.New(core.Options{SharePort: 8080, Quarantine: t.TempDir(), ConfigDir: t.TempDir(),
		Config: platform.Config{Name: "Martín"}, LAN: lan, Book: book})
}

func req(method, path, body, host, token string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Host = host
	if token != "" {
		r.Header.Set("X-Pasame-Token", token)
	}
	return r
}

func TestRejectsWithoutToken(t *testing.T) {
	h := New(newCore(t), tok, 5555)
	for _, r := range []*http.Request{
		req("POST", "/api/stop", "", "127.0.0.1:5555", ""),
		req("POST", "/api/stop", "", "127.0.0.1:5555", "malo"),
		req("GET", "/events", "", "127.0.0.1:5555", ""),
		req("GET", "/events?t=malo", "", "127.0.0.1:5555", ""),
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Errorf("%s %s → %d", r.Method, r.URL, w.Code)
		}
	}
}

func TestRejectsForeignHost(t *testing.T) {
	h := New(newCore(t), tok, 5555)
	for _, host := range []string{"evil.com", "192.168.1.42:5555", "127.0.0.1:6666", "localhost.evil.com:5555"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req("POST", "/api/stop", "", host, tok))
		if w.Code != http.StatusForbidden {
			t.Errorf("Host %q → %d", host, w.Code)
		}
		w = httptest.NewRecorder()
		h.ServeHTTP(w, req("GET", "/", "", host, ""))
		if w.Code != http.StatusForbidden {
			t.Errorf("GET / con Host %q → %d", host, w.Code)
		}
	}
}

func TestIndexServed(t *testing.T) {
	h := New(newCore(t), tok, 5555)
	for _, host := range []string{"127.0.0.1:5555", "localhost:5555"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req("GET", "/?t="+tok, "", host, ""))
		if w.Code != 200 || !strings.Contains(w.Body.String(), "Pasame") {
			t.Fatalf("%s → %d", host, w.Code)
		}
	}
}

func TestActions(t *testing.T) {
	c := newCore(t)
	h := New(c, tok, 5555)
	do := func(path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req("POST", path, body, "127.0.0.1:5555", tok))
		return w
	}
	if w := do("/api/receive-only", ""); w.Code != http.StatusNoContent || c.State().Phase != "sharing" {
		t.Fatalf("%d %+v", w.Code, c.State().Phase)
	}
	if w := do("/api/stop", ""); w.Code != http.StatusNoContent || c.Current() != nil {
		t.Fatal("stop")
	}
	if w := do("/api/name", `{"name":"Rosa"}`); w.Code != http.StatusNoContent || c.Sender() != "Rosa" {
		t.Fatal("name")
	}
	if w := do("/api/name", `{"name":"  "}`); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "error") {
		t.Fatalf("name vacío → %d", w.Code)
	}
	if w := do("/api/name", `no-json`); w.Code != http.StatusBadRequest {
		t.Fatal("json roto")
	}
	if w := do("/api/iface", `{"ip":"8.8.8.8"}`); w.Code != http.StatusBadRequest {
		t.Fatal("iface ajena aceptada")
	}
	if w := do("/api/pick", `{"kind":"otra"}`); w.Code != http.StatusBadRequest {
		t.Fatal("kind inválido aceptado")
	}
	if w := do("/api/quit", ""); w.Code != http.StatusNoContent {
		t.Fatal("quit")
	}
	select {
	case <-c.Done():
	case <-time.After(time.Second):
		t.Fatal("quit no cerró Done")
	}
}

func TestEventsStream(t *testing.T) {
	c := newCore(t)
	srv := httptest.NewUnstartedServer(nil)
	port := srv.Listener.Addr().(*net.TCPAddr).Port
	srv.Config.Handler = New(c, tok, port)
	srv.Start()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/events?t=" + tok)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("CT %q", resp.Header.Get("Content-Type"))
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	next := func() core.State {
		t.Helper()
		var ev string
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "event: ") {
				ev = strings.TrimPrefix(line, "event: ")
			}
			if strings.HasPrefix(line, "data: ") && ev == "state" {
				var s core.State
				if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &s); err != nil {
					t.Fatal(err)
				}
				return s
			}
		}
		t.Fatal("se cortó el stream")
		return core.State{}
	}
	if s := next(); s.Phase != "idle" {
		t.Fatalf("primer estado %+v", s.Phase)
	}
	c.ReceiveOnly()
	if s := next(); s.Phase != "sharing" || !strings.HasPrefix(s.QR, "<svg") {
		t.Fatalf("estado tras compartir: %+v", s.Phase)
	}
}
```

- [ ] **Step 3: Verificar que falla**

Run: `go test ./internal/control/`
Expected: FAIL — `undefined: New`.

- [ ] **Step 4: Implementar**

`internal/control/server.go`:
```go
// Package control es la UI local del emisor. Solo escucha en loopback y solo conoce a core.
package control

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"io/fs"
	"net/http"
	"strconv"

	"github.com/KonixDev/pasame/internal/core"
	"github.com/KonixDev/pasame/web"
)

func NewToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

type server struct {
	c     *core.Core
	token string
	hosts map[string]bool
	mux   *http.ServeMux
}

func New(c *core.Core, token string, port int) http.Handler {
	p := strconv.Itoa(port)
	s := &server{c: c, token: token, mux: http.NewServeMux(),
		hosts: map[string]bool{"127.0.0.1:" + p: true, "localhost:" + p: true}}
	static, _ := fs.Sub(web.Control, "control")
	s.mux.Handle("GET /", http.FileServerFS(static))
	s.mux.HandleFunc("GET /events", s.auth(s.events))
	s.mux.HandleFunc("POST /api/pick", s.auth(s.pick))
	s.mux.HandleFunc("POST /api/add", s.auth(s.add))
	s.mux.HandleFunc("POST /api/receive-only", s.auth(s.simple(c.ReceiveOnly)))
	s.mux.HandleFunc("POST /api/stop", s.auth(s.simple(c.Stop)))
	s.mux.HandleFunc("POST /api/quit", s.auth(s.simple(c.RequestQuit)))
	s.mux.HandleFunc("POST /api/name", s.auth(s.name))
	s.mux.HandleFunc("POST /api/iface", s.auth(s.iface))
	s.mux.HandleFunc("POST /api/open-folder", s.auth(s.errOnly(c.OpenFolder)))
	s.mux.HandleFunc("POST /api/firewall", s.auth(s.errOnly(c.OpenFirewall)))
	return s
}

// ServeHTTP rechaza cualquier Host que no sea el nuestro (mitiga DNS rebinding).
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.hosts[r.Host] {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Options", "DENY")
	s.mux.ServeHTTP(w, r)
}

func (s *server) auth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Pasame-Token")
		if r.URL.Path == "/events" {
			got = r.URL.Query().Get("t")
		}
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		h(w, r)
	}
}
```

`internal/control/api.go`:
```go
package control

import (
	"encoding/json"
	"net/http"
)

func bad(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v); err != nil {
		bad(w, "pedido inválido")
		return false
	}
	return true
}

func (s *server) simple(f func()) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { f(); w.WriteHeader(http.StatusNoContent) }
}

func (s *server) errOnly(f func() error) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if err := f(); err != nil {
			bad(w, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *server) pick(w http.ResponseWriter, r *http.Request) {
	var b struct{ Kind string }
	if !decode(w, r, &b) {
		return
	}
	if b.Kind != "files" && b.Kind != "folder" {
		bad(w, "tipo inválido")
		return
	}
	s.c.Pick(b.Kind)
	w.WriteHeader(http.StatusAccepted)
}

// add recibe rutas: de una segunda instancia (arrastrar sobre el ícono) o del campo "pegar ruta" en Linux.
func (s *server) add(w http.ResponseWriter, r *http.Request) {
	var b struct{ Paths []string }
	if !decode(w, r, &b) {
		return
	}
	if len(b.Paths) == 0 {
		bad(w, "no llegó ningún archivo")
		return
	}
	s.c.Share(b.Paths)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) name(w http.ResponseWriter, r *http.Request) {
	var b struct{ Name string }
	if !decode(w, r, &b) {
		return
	}
	if err := s.c.SetName(b.Name); err != nil {
		bad(w, "Escribí un nombre")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) iface(w http.ResponseWriter, r *http.Request) {
	var b struct{ IP string }
	if !decode(w, r, &b) {
		return
	}
	if err := s.c.SetIface(b.IP); err != nil {
		bad(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

`internal/control/events.go`:
```go
package control

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func (s *server) events(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	release := s.c.ClientConnected()
	defer release()
	changes, cancel := s.c.Subscribe()
	defer cancel()
	ping := time.NewTicker(5 * time.Second)
	defer ping.Stop()

	send := func() error {
		b, _ := json.Marshal(s.c.State())
		if _, err := fmt.Fprintf(w, "event: state\ndata: %s\n\n", b); err != nil {
			return err
		}
		return rc.Flush()
	}
	if send() != nil {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.c.Done():
			fmt.Fprint(w, "event: quit\ndata: {}\n\n")
			rc.Flush()
			return
		case <-changes:
			if send() != nil {
				return
			}
		case <-ping.C:
			if _, err := fmt.Fprint(w, "event: ping\ndata: {}\n\n"); err != nil || rc.Flush() != nil {
				return
			}
		}
	}
}
```

- [ ] **Step 5: Verificar que pasa**

Run: `go test -race ./internal/control/ ./internal/core/`
Expected: `ok`.

- [ ] **Step 6: Commit**

```bash
git add web internal/control internal/core
git commit -m "feat(control): API local con token y chequeo de Host, estado en vivo por SSE

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 18: UI del emisor (`web/control`)

Spec: §4.7 (tamaño del QR y pantalla completa), §4.8 (frontend), §4.9 (cartel "fijate detrás", zona de drop), §6 completo (voz, conteo de clicks, pantallas 6.1, 6.2, 6.6), §7 (hint de 45 s, VPN, cambio de red).

**Files:**
- Modify: `web/control/index.html` (reemplazo completo)
- Create: `web/control/app.css`, `web/control/app.js`
- Test: `internal/control/ui_test.go`

**Interfaces:**
- Consumes: SSE `state`/`quit`/`ping` y la API de la Tarea 17; campos de `core.State` (Tarea 16).
- Produces: nada que consuma otro paquete. En el Plan 2 se agregan al objeto `T` y a `render()` los textos y el bloque del túnel.

- [ ] **Step 1: Test que falla (voz de la UI y estructura mínima)**

`internal/control/ui_test.go`:
```go
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
```

- [ ] **Step 2: Verificar que falla**

Run: `go test ./internal/control/ -run UI`
Expected: FAIL — falta `control/app.js`.

- [ ] **Step 3: `web/control/index.html`**

```html
<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Pasame</title>
<link rel="stylesheet" href="app.css">
</head>
<body>
<header>
  <span class="brand">Pasame</span>
  <span id="who"></span>
  <button id="menu" class="link" aria-label="Más opciones">⋯</button>
</header>
<div id="menu-box" hidden><button id="quit" class="link">Cerrar Pasame</button></div>
<main id="app"><p class="muted">Cargando…</p></main>
<div id="toast" hidden></div>
<script src="app.js"></script>
</body>
</html>
```

- [ ] **Step 4: `web/control/app.css`**

```css
*{box-sizing:border-box}
body{margin:0;font-family:-apple-system,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;font-size:19px;line-height:1.45;color:#111;background:#fff}
header{display:flex;align-items:center;gap:16px;padding:14px 24px;border-bottom:1px solid #e5e5e5}
.brand{font-weight:700;font-size:22px;flex:1}
#who{color:#444}
main{max-width:980px;margin:0 auto;padding:28px 24px 60px}
h1{font-size:30px;margin:0 0 8px}
.center{text-align:center}
.primary{display:inline-block;min-width:360px;min-height:64px;padding:16px 28px;font-size:22px;font-weight:700;color:#fff;background:#0a58ca;border:0;border-radius:14px;cursor:pointer}
.primary:focus-visible,.secondary:focus-visible,.link:focus-visible{outline:3px solid #f59f00;outline-offset:2px}
.secondary{padding:10px 18px;font-size:18px;color:#0a58ca;background:#fff;border:2px solid #0a58ca;border-radius:10px;cursor:pointer}
.link{background:none;border:0;padding:4px;color:#0a58ca;font-size:inherit;text-decoration:underline;cursor:pointer}
.muted{color:#555}
.small{font-size:15px;color:#666}
.notice{background:#fff8e1;border:1px solid #f1d38a;border-radius:10px;padding:14px 16px;margin:16px 0}
.ok{background:#e7f6ec;border-color:#9fd3ae}
.row{display:flex;flex-wrap:wrap;gap:32px;align-items:flex-start;margin:24px 0}
.qr{cursor:zoom-in;line-height:0}
.qr svg{width:clamp(280px,36vw,480px);height:auto;border:1px solid #ddd;border-radius:8px}
.qr.full{position:fixed;top:0;left:0;right:0;bottom:0;background:#fff;display:flex;align-items:center;justify-content:center;z-index:10;cursor:zoom-out}
.qr.full svg{width:min(92vw,92vh);border:0}
.addr{font-family:ui-monospace,Menlo,Consolas,monospace;font-size:40px;font-weight:700;padding:10px 18px;border:2px solid #111;border-radius:10px;display:inline-block;user-select:all;word-break:break-all}
.side{flex:1;min-width:300px}
.side p{margin:0 0 10px}
section{margin-top:28px}
section h2{font-size:17px;text-transform:uppercase;letter-spacing:.04em;color:#666;border-bottom:1px solid #e5e5e5;padding-bottom:6px}
ul.plain{list-style:none;padding:0;margin:0}
ul.plain li{padding:6px 0}
.topbar{display:flex;justify-content:space-between;align-items:center;gap:16px;flex-wrap:wrap}
#menu-box{position:absolute;right:24px;top:56px;background:#fff;border:1px solid #ddd;border-radius:10px;padding:8px 12px;box-shadow:0 4px 16px rgba(0,0,0,.1)}
#toast{position:fixed;left:50%;bottom:28px;transform:translateX(-50%);background:#111;color:#fff;padding:12px 20px;border-radius:10px}
input[type=text]{font-size:19px;padding:8px 10px;border:2px solid #999;border-radius:8px}
select{font-size:16px}
@media (max-width:640px){.primary{min-width:0;width:100%}.addr{font-size:28px}}
```

- [ ] **Step 5: `web/control/app.js`**

```js
(function () {
  'use strict';
  var t = new URLSearchParams(location.search).get('t') || '';
  var state = null;
  var closed = false;

  // Todos los textos del emisor (spec §13). Voz rioplatense, sin palabras técnicas.
  var T = {
    who: 'Tu nombre: ',
    edit: '✎',
    headline: 'Pasá archivos a cualquier celular<br>o computadora que esté cerca.',
    pickFiles: '📂  Elegir archivos',
    pickFolder: 'o elegir una carpeta entera',
    receiveOnly: 'Solo quiero recibir archivos',
    // El spec §6.1 dice "No pasan por ningún servidor", pero §6 prohíbe la palabra "servidor": gana la regla de voz.
    direct: 'Los archivos van directo de tu computadora al otro dispositivo, sin pasar por internet.',
    picking: 'Se abrió una ventana para elegir archivos. Si no la ves, fijate detrás de esta.',
    noDialog: 'No se pudo abrir la ventana para elegir. Pegá acá la ubicación del archivo o carpeta:',
    paste: 'Compartir',
    fwWindows: '<b>Windows te va a pedir permiso.</b> Cuando aparezca una ventana que dice "Firewall de Windows Defender", tocá <b>Permitir acceso</b>. Es para que el celular pueda ver tu computadora.',
    fwMac: 'Si tu Mac pregunta si permitís que Pasame acepte conexiones, tocá <b>Permitir</b>.',
    sharingN: function (n, total) { return 'Compartiendo ' + n + (n === 1 ? ' archivo' : ' archivos') + ' (' + total + ')'; },
    receiving: 'Listo para recibir archivos',
    stop: 'Terminar',
    phone: '<b>Desde un celular:</b><br>apuntá la cámara a este código y tocá el aviso que aparece.',
    pc: '<b>Desde una computadora:</b><br>escribí esto en el navegador',
    alsoTry: 'o probá: ',
    activity: 'Actividad',
    files: 'Archivos',
    nobody: '● Nadie entró todavía. Tiene que estar en la misma red WiFi.',
    devices: function (n) { return '● ' + n + (n === 1 ? ' dispositivo conectado' : ' dispositivos conectados'); },
    downloading: function (name, k) { return '⬇ Descargando ' + name + (k > 1 ? ' (' + k + ' personas)' : ''); },
    downloaded: function (name, k) { return '✓ ' + name + ' descargado' + (k > 1 ? ' ' + k + ' veces' : ''); },
    uploading: function (name) { return '⬆ Recibiendo ' + name + '…'; },
    received: function (name) { return '✓ Recibido ' + name; },
    openFolder: 'Abrir carpeta',
    cantEnter: '<b>¿No pueden entrar?</b> Probá en este orden:<ol>' +
      '<li>Los dos dispositivos tienen que estar en <b>la misma red WiFi</b> (fijate el nombre de la red en el celular).</li>' +
      '<li>Si están en un WiFi de un bar, hotel o aeropuerto, esas redes suelen bloquear esto.</li>' +
      '<li>Si tenés una VPN, apagala un momento.</li></ol>',
    checkWindows: 'Revisar permiso de Windows',
    vpn: 'Parece que tenés una VPN activa. Si el celular no puede entrar, apagala un momento o tocá Cambiar red.',
    netChanged: 'Cambió la red. El código se actualizó.',
    unreadable: 'No pude leer: ',
    network: 'Red: ',
    changeNet: 'Cambiar red',
    auto: 'Automática',
    tabHint: 'Si cerrás esta pestaña, Pasame se cierra solo a los 2 minutos.',
    stopped: 'Listo. Los links ya no funcionan.',
    closedMsg: 'Pasame está cerrado. Podés cerrar esta pestaña.',
    dropHint: 'Para elegir archivos usá el botón (o arrastralos sobre el ícono de Pasame).',
    noNetwork: 'Esta computadora no está conectada a ninguna red. Conectate a un WiFi y esperá unos segundos.'
  };
  T.cantEnterPlan2 = ''; // el Plan 2 agrega acá el paso "Tocá Compartir por internet"

  function esc(s) {
    return String(s).replace(/[&<>"']/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
    });
  }

  function api(path, body) {
    return fetch(path, {
      method: 'POST',
      headers: { 'X-Pasame-Token': t, 'Content-Type': 'application/json' },
      body: JSON.stringify(body || {})
    }).then(function (r) {
      if (r.ok) return null;
      return r.json().then(function (j) { toast(j.error || 'No se pudo.'); }, function () { toast('No se pudo.'); });
    });
  }

  function toast(msg) {
    var el = document.getElementById('toast');
    el.textContent = msg;
    el.hidden = false;
    clearTimeout(toast.timer);
    toast.timer = setTimeout(function () { el.hidden = true; }, 3000);
  }

  function renderWho(s) {
    document.getElementById('who').innerHTML = esc(T.who) + '<b>' + esc(s.name) + '</b> <button class="link" id="edit-name" aria-label="Cambiar nombre">' + T.edit + '</button>';
  }

  function renderIdle(s) {
    var h = '<div class="center">';
    h += '<h1 style="margin:40px 0 32px;font-weight:600">' + T.headline + '</h1>';
    if (s.firewallHint === 'windows') h += '<div class="notice">' + T.fwWindows + '</div>';
    if (s.firewallHint === 'mac') h += '<div class="notice">' + T.fwMac + '</div>';
    if (s.phase === 'picking') h += '<div class="notice">' + esc(T.picking) + '</div>';
    h += '<p><button class="primary" data-act="pick-files">' + esc(T.pickFiles) + '</button></p>';
    h += '<p><button class="link" data-act="pick-folder">' + esc(T.pickFolder) + '</button></p>';
    h += '<p><button class="link" data-act="receive-only">' + esc(T.receiveOnly) + '</button></p>';
    if (s.pickUnsupported) {
      h += '<div class="notice"><p>' + esc(T.noDialog) + '</p><input type="text" id="paste" size="40"> ' +
        '<button class="secondary" data-act="paste">' + esc(T.paste) + '</button></div>';
    }
    h += '<p class="muted" style="margin-top:48px">' + esc(T.direct) + '</p>';
    h += '<p class="small" id="drop-hint" hidden>' + esc(T.dropHint) + '</p></div>';
    return h;
  }

  function activity(s) {
    var st = s.stats || {}, lines = [];
    lines.push(st.clients ? T.devices(st.clients) : T.nobody);
    var files = s.files || [];
    Object.keys(st.active || {}).forEach(function (i) {
      if (files[i]) lines.push(esc(T.downloading(files[i].name, st.active[i])));
    });
    Object.keys(st.completed || {}).forEach(function (i) {
      if (files[i]) lines.push(esc(T.downloaded(files[i].name, st.completed[i])));
    });
    (st.uploading || []).forEach(function (n) { lines.push(esc(T.uploading(n))); });
    (st.received || []).forEach(function (n) {
      lines.push(esc(T.received(n)) + ' <button class="link" data-act="open-folder">' + esc(T.openFolder) + '</button>');
    });
    return '<ul class="plain">' + lines.map(function (l) { return '<li>' + l + '</li>'; }).join('') + '</ul>';
  }

  function renderSharing(s) {
    var addrs = s.addresses || [];
    var main = addrs[0];
    var h = '<div class="topbar"><h1>' + esc(s.receiveOnly ? T.receiving : T.sharingN(s.count, s.total)) + '</h1>' +
      '<button class="secondary" data-act="stop">' + esc(T.stop) + '</button></div>';
    if (s.netChanged) h += '<div class="notice ok">' + esc(T.netChanged) + '</div>';
    if (s.vpn) h += '<div class="notice">' + esc(T.vpn) + '</div>';
    if ((s.unreadable || []).length) h += '<div class="notice">' + esc(T.unreadable + s.unreadable.join(', ')) + '</div>';
    if (!main) return h + '<div class="notice">' + esc(T.noNetwork) + '</div>';
    h += '<div class="row"><div class="qr" id="qr" title="Tocá para agrandar">' + s.qr + '</div><div class="side">';
    h += '<p>' + T.phone + '</p><p style="margin-top:24px">' + T.pc + '</p>';
    h += '<div class="addr">' + esc(main.display) + '</div>';
    addrs.slice(1).forEach(function (a) {
      if (a.kind === 'mdns') h += '<p class="small">' + esc(T.alsoTry + a.display) + '</p>';
    });
    h += '<div id="plan2-slot"></div></div></div>';

    h += '<section><h2>' + esc(T.activity) + '</h2>' + activity(s);
    var st = s.stats || {};
    if (!st.clients && s.sharedAt && Date.now() - s.sharedAt > 45000) {
      h += '<div class="notice">' + T.cantEnter + T.cantEnterPlan2;
      if (s.firewallHint === 'windows' || navigator.userAgent.indexOf('Windows') >= 0) {
        h += '<button class="secondary" data-act="firewall">' + esc(T.checkWindows) + '</button>';
      }
      h += '</div>';
    }
    h += '</section>';

    if (!s.receiveOnly) {
      h += '<section><h2>' + esc(T.files) + '</h2><ul class="plain">';
      (s.files || []).forEach(function (f) { h += '<li>' + esc(f.name) + ' <span class="small">' + esc(f.size) + '</span></li>'; });
      h += '</ul></section>';
    }

    var cur = (s.ifaces || [])[0];
    h += '<p class="small" style="margin-top:28px">' + esc(T.network) + (cur ? esc(cur.human + ' (' + cur.ip + ')') : '—') +
      ' · <label>' + esc(T.changeNet) + ' <select id="iface"><option value="">' + esc(T.auto) + '</option>';
    (s.ifaces || []).forEach(function (c) {
      h += '<option value="' + esc(c.ip) + '">' + esc(c.human + ' (' + c.ip + ')') + '</option>';
    });
    h += '</select></label></p><p class="small">' + esc(T.tabHint) + '</p>';
    return h;
  }

  function render() {
    if (!state || closed) return;
    // Re-render cada 5 s: no pisar un control que la persona está usando (se le cerraría el menú de red).
    var f = document.activeElement;
    if (f && (f.id === 'iface' || f.id === 'paste')) return;
    renderWho(state);
    var app = document.getElementById('app');
    var qrFull = document.querySelector('.qr.full') !== null;
    app.innerHTML = state.phase === 'sharing' ? renderSharing(state) : renderIdle(state);
    if (qrFull && document.getElementById('qr')) document.getElementById('qr').classList.add('full');
  }

  function showClosed() {
    closed = true;
    document.getElementById('app').innerHTML = '<h1 class="center" style="margin-top:80px">' + esc(T.closedMsg) + '</h1>';
  }

  document.addEventListener('click', function (e) {
    var el = e.target.closest('[data-act], #qr, #edit-name, #menu, #quit');
    if (!el) return;
    if (el.id === 'qr') return el.classList.toggle('full');
    if (el.id === 'menu') { var m = document.getElementById('menu-box'); m.hidden = !m.hidden; return; }
    if (el.id === 'quit') return api('/api/quit').then(showClosed);
    if (el.id === 'edit-name') {
      var n = prompt('¿Cómo te llamás? (lo ve quien recibe)', state.name); // prompt: único diálogo, lo abre la persona
      if (n !== null) api('/api/name', { name: n });
      return;
    }
    switch (el.getAttribute('data-act')) {
      case 'pick-files': return api('/api/pick', { kind: 'files' });
      case 'pick-folder': return api('/api/pick', { kind: 'folder' });
      case 'receive-only': return api('/api/receive-only');
      case 'stop': return api('/api/stop').then(function () { toast(T.stopped); });
      case 'open-folder': return api('/api/open-folder');
      case 'firewall': return api('/api/firewall');
      case 'paste':
        var v = document.getElementById('paste').value.trim();
        if (v) api('/api/add', { paths: [v] });
        return;
    }
  });

  document.addEventListener('change', function (e) {
    if (e.target.id === 'iface') api('/api/iface', { ip: e.target.value });
  });

  // Soltar archivos sobre la página no puede funcionar (el navegador no da rutas): se explica.
  document.addEventListener('dragover', function (e) { e.preventDefault(); });
  document.addEventListener('drop', function (e) {
    e.preventDefault();
    var hint = document.getElementById('drop-hint');
    if (hint) hint.hidden = false; else toast(T.dropHint);
  });

  var es = new EventSource('/events?t=' + encodeURIComponent(t));
  es.addEventListener('state', function (e) { state = JSON.parse(e.data); render(); });
  es.addEventListener('quit', function () { es.close(); showClosed(); });
  var fails = 0;
  es.onopen = function () { fails = 0; };
  es.onerror = function () { if (++fails > 3) { es.close(); showClosed(); } };

  // El hint de 45 s depende del reloj, no de un evento: se re-evalúa solo.
  setInterval(render, 5000);
})();
```

- [ ] **Step 6: Verificar que pasa**

Run: `go test -race ./internal/control/`
Expected: `ok`.

- [ ] **Step 7: Commit**

```bash
git add web/control internal/control/ui_test.go
git commit -m "feat(web): UI del emisor — inicio, compartiendo, actividad y ayudas de red

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```
La verificación visual y de clicks se hace en la Tarea 20 con el binario real.

---

### Task 19: Instancia única y traspaso a la instancia viva

Spec: §3.2 (doble clic con la app abierta reabre la misma UI), §4.10 (single instance, handoff), §9.1 (fila `lifecycle/lock`).

**Files:**
- Create: `internal/lifecycle/lock.go`, `internal/lifecycle/flock_unix.go`, `internal/lifecycle/flock_windows.go`, `internal/lifecycle/handoff.go`
- Test: `internal/lifecycle/lock_test.go`

**Interfaces:**
- Consumes: nada (habla HTTP con la API de la Tarea 17 sin importar `control`).
- Produces:
  ```go
  var ErrHeld = errors.New("lifecycle: otra instancia está corriendo")
  type Info struct { Port int `json:"port"`; Token string `json:"token"`; PID int `json:"pid"` }
  func Acquire(dir string) (*Lock, error)        // ErrHeld si otra instancia lo tiene
  func (l *Lock) Publish(i Info) error           // escribe <dir>/pasame.json (después de abrir listeners)
  func (l *Lock) Release() error                 // suelta el lock y borra pasame.json
  func ReadInfo(dir string) (Info, error)
  func Handoff(i Info, paths []string) (controlURL string, err error) // con paths: POST /api/add
  ```
  El lock (`pasame.lock`) y los datos (`pasame.json`) son archivos distintos: en Windows `LockFileEx` bloquea también la lectura del rango bloqueado. Si el proceso muere, el OS suelta el lock solo: no hay "lock huérfano" que limpiar a mano.

- [ ] **Step 1: Test que falla (dos procesos reales)**

`internal/lifecycle/lock_test.go`:
```go
package lifecycle

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

// TestMain permite que el binario de test haga de "segunda instancia".
func TestMain(m *testing.M) {
	switch os.Getenv("PASAME_LOCK_CHILD") {
	case "try":
		_, err := Acquire(os.Getenv("PASAME_LOCK_DIR"))
		if errors.Is(err, ErrHeld) {
			os.Exit(3)
		}
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	case "hold-and-die":
		if _, err := Acquire(os.Getenv("PASAME_LOCK_DIR")); err != nil {
			os.Exit(1)
		}
		os.Exit(0) // sin Release: simula un cierre abrupto
	}
	os.Exit(m.Run())
}

func child(t *testing.T, mode, dir string) int {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "PASAME_LOCK_CHILD="+mode, "PASAME_LOCK_DIR="+dir)
	err := cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	if err != nil {
		t.Fatal(err)
	}
	return 0
}

func TestSecondInstanceIsDetected(t *testing.T) {
	dir := t.TempDir()
	l, err := Acquire(dir)
	if err != nil {
		t.Fatal(err)
	}
	if code := child(t, "try", dir); code != 3 {
		t.Fatalf("la segunda instancia no vio el lock (exit %d)", code)
	}
	l.Release()
	if code := child(t, "try", dir); code != 0 {
		t.Fatalf("después de Release no pudo tomarlo (exit %d)", code)
	}
}

func TestOrphanLockIsRecovered(t *testing.T) {
	dir := t.TempDir()
	if code := child(t, "hold-and-die", dir); code != 0 {
		t.Fatalf("exit %d", code)
	}
	l, err := Acquire(dir)
	if err != nil {
		t.Fatalf("lock de un proceso muerto no se recuperó: %v", err)
	}
	l.Release()
}

func TestPublishAndRead(t *testing.T) {
	dir := t.TempDir()
	l, _ := Acquire(dir)
	defer l.Release()
	want := Info{Port: 53211, Token: "abc", PID: os.Getpid()}
	if err := l.Publish(want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadInfo(dir)
	if err != nil || got != want {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestHandoff(t *testing.T) {
	var gotPaths []string
	var gotToken, gotHost string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken, gotHost = r.Header.Get("X-Pasame-Token"), r.Host
		var b struct{ Paths []string }
		json.NewDecoder(r.Body).Decode(&b)
		gotPaths = b.Paths
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	port := srv.Listener.Addr().(*net.TCPAddr).Port
	info := Info{Port: port, Token: "tok"}

	url, err := Handoff(info, []string{"/a.jpg", "/b.mp4"})
	if err != nil || len(gotPaths) != 2 || gotToken != "tok" || gotHost != srv.Listener.Addr().String() {
		t.Fatalf("%v %v %q %q", err, gotPaths, gotToken, gotHost)
	}
	if url != srv.URL+"/?t=tok" {
		t.Fatalf("url = %q", url)
	}
	gotPaths = nil
	if _, err := Handoff(info, nil); err != nil || gotPaths != nil {
		t.Fatal("sin rutas no debería llamar a la API")
	}
}
```

- [ ] **Step 2: Verificar que falla**

Run: `go test ./internal/lifecycle/`
Expected: FAIL — `undefined: Acquire`.

- [ ] **Step 3: Implementar**

`internal/lifecycle/lock.go`:
```go
// Package lifecycle garantiza una sola instancia y le pasa el trabajo a la que ya corre.
package lifecycle

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

var ErrHeld = errors.New("lifecycle: otra instancia está corriendo")

type Info struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
	PID   int    `json:"pid"`
}

type Lock struct {
	f   *os.File
	dir string
}

func Acquire(dir string) (*Lock, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "pasame.lock"), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := tryLock(f); err != nil {
		f.Close()
		return nil, ErrHeld
	}
	return &Lock{f: f, dir: dir}, nil
}

func (l *Lock) Publish(i Info) error {
	b, _ := json.Marshal(i)
	tmp := filepath.Join(l.dir, "pasame.json.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil { // 0600: el token no lo lee otro usuario
		return err
	}
	return os.Rename(tmp, filepath.Join(l.dir, "pasame.json"))
}

func (l *Lock) Release() error {
	os.Remove(filepath.Join(l.dir, "pasame.json"))
	unlock(l.f)
	return l.f.Close()
}

func ReadInfo(dir string) (Info, error) {
	var i Info
	b, err := os.ReadFile(filepath.Join(dir, "pasame.json"))
	if err != nil {
		return i, err
	}
	return i, json.Unmarshal(b, &i)
}
```

`internal/lifecycle/flock_unix.go`:
```go
//go:build !windows

package lifecycle

import (
	"os"
	"syscall"
)

func tryLock(f *os.File) error { return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) }
func unlock(f *os.File)        { syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }
```

`internal/lifecycle/flock_windows.go`:
```go
//go:build windows

package lifecycle

import (
	"os"

	"golang.org/x/sys/windows"
)

func tryLock(f *os.File) error {
	ol := new(windows.Overlapped)
	return windows.LockFileEx(windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, ol)
}

func unlock(f *os.File) {
	windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, new(windows.Overlapped))
}
```

`internal/lifecycle/handoff.go`:
```go
package lifecycle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Handoff le pasa las rutas a la instancia viva y devuelve la URL de su UI.
func Handoff(i Info, paths []string) (string, error) {
	base := fmt.Sprintf("http://127.0.0.1:%d", i.Port)
	url := base + "/?t=" + i.Token
	if len(paths) == 0 {
		return url, nil
	}
	body, _ := json.Marshal(map[string][]string{"paths": paths})
	req, _ := http.NewRequest("POST", base+"/api/add", bytes.NewReader(body))
	req.Header.Set("X-Pasame-Token", i.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return url, err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return url, fmt.Errorf("la instancia abierta respondió %d", resp.StatusCode)
	}
	return url, nil
}
```
(`httptest.NewServer` escucha en `127.0.0.1`, así que `srv.URL` coincide con la URL que arma `Handoff`.)

- [ ] **Step 4: Verificar que pasa**

Run: `go test -race ./internal/lifecycle/ && make cross`
Expected: `ok` y los 7 targets compilan.

- [ ] **Step 5: Commit**

```bash
git add internal/lifecycle
git commit -m "feat(lifecycle): instancia única con lock del OS y traspaso de archivos

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 20: `cmd/pasame` — cableado, señales, apagado limpio y reglas de dependencia

Spec: §3.2 (arranque: listener de control, token, abrir navegador), §4.2, §4.10 (apagado limpio, logs), §8 (reglas de dependencia verificadas con `go list -deps`), §13 (flags de build).

**Files:**
- Create: `cmd/pasame/main.go`, `cmd/pasame/logging.go`
- Create: `internal/deps_test.go`, `internal/doc.go` (una línea: `package internal`, para que `go build ./...` no se queje de un directorio con solo tests)
- Test: `cmd/pasame/main_test.go`

**Interfaces:**
- Consumes: todo lo anterior.
- Produces: el binario `pasame`. `var version = "dev"` (lo pisa `-X main.version=<tag>`). Función `cleanArgs(args []string) []string` (descarta `-psn_*` que macOS agrega a veces y cualquier flag que empiece con `-`).

- [ ] **Step 1: Test de reglas de dependencia (falla hasta que existan todos los paquetes; hoy ya existen)**

`internal/deps_test.go`:
```go
package internal

import (
	"os/exec"
	"strings"
	"testing"
)

const mod = "github.com/KonixDev/pasame/"

func deps(t *testing.T, pkg string) map[string]bool {
	t.Helper()
	out, err := exec.Command("go", "list", "-deps", mod+pkg).Output()
	if err != nil {
		t.Fatalf("go list %s: %v", pkg, err)
	}
	m := map[string]bool{}
	for _, l := range strings.Fields(string(out)) {
		m[l] = true
	}
	return m
}

func imports(t *testing.T, pkg string) []string {
	t.Helper()
	out, err := exec.Command("go", "list", "-f", `{{join .Imports " "}}`, mod+pkg).Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(out))
}

func TestPurePackages(t *testing.T) {
	for _, p := range []string{"internal/session", "internal/files", "internal/qr", "internal/i18n"} {
		d := deps(t, p)
		for _, banned := range []string{"net/http", "os/exec"} {
			if d[banned] {
				t.Errorf("%s depende de %s", p, banned)
			}
		}
	}
}

func TestLayering(t *testing.T) {
	rules := map[string][]string{
		"internal/share": {"internal/control", "internal/core", "internal/addr", "internal/tunnel"},
		"internal/addr":  {"internal/control", "internal/share"},
	}
	for p, banned := range rules {
		d := deps(t, p)
		for _, b := range banned {
			if d[mod+b] {
				t.Errorf("%s no puede depender de %s", p, b)
			}
		}
	}
	for _, imp := range imports(t, "internal/control") {
		if strings.HasPrefix(imp, mod+"internal/") && imp != mod+"internal/core" {
			t.Errorf("control solo puede importar core; importa %s", imp)
		}
	}
}
```
Run: `go test ./internal/`
Expected: `ok` (si falla, hay que corregir el paquete que rompe la regla, no el test).

- [ ] **Step 2: Test de `cleanArgs` (falla)**

`cmd/pasame/main_test.go`:
```go
package main

import (
	"reflect"
	"testing"
)

func TestCleanArgs(t *testing.T) {
	got := cleanArgs([]string{"-psn_0_12345", "/a.jpg", "--algo", "/b c.mp4"})
	if !reflect.DeepEqual(got, []string{"/a.jpg", "/b c.mp4"}) {
		t.Fatalf("%v", got)
	}
	if len(cleanArgs(nil)) != 0 {
		t.Fatal("nil")
	}
}
```
Run: `go test ./cmd/pasame/`
Expected: FAIL — `undefined: cleanArgs`.

- [ ] **Step 3: Implementar `cmd/pasame/logging.go`**

```go
package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

const maxLog = 5 << 20

// setupLogging escribe a stderr y a <config>/pasame.log (se trunca al superar 5 MB).
func setupLogging(dir string) {
	p := filepath.Join(dir, "pasame.log")
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if fi, err := os.Stat(p); err == nil && fi.Size() > maxLog {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(p, flags, 0o644)
	if err != nil {
		return
	}
	log.SetOutput(io.MultiWriter(os.Stderr, f))
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("pasame: ")
}
```

- [ ] **Step 4: Implementar `cmd/pasame/main.go`**

```go
// Pasame: pasá archivos a cualquier celular o computadora que esté cerca.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/KonixDev/pasame/internal/addr"
	"github.com/KonixDev/pasame/internal/browser"
	"github.com/KonixDev/pasame/internal/control"
	"github.com/KonixDev/pasame/internal/core"
	"github.com/KonixDev/pasame/internal/files"
	"github.com/KonixDev/pasame/internal/lifecycle"
	"github.com/KonixDev/pasame/internal/platform"
	"github.com/KonixDev/pasame/internal/share"
)

var version = "dev"

func cleanArgs(args []string) []string {
	var out []string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			out = append(out, a)
		}
	}
	return out
}

func main() {
	if err := run(); err != nil {
		log.Printf("error fatal: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfgDir, err := platform.ConfigDir()
	if err != nil {
		return err
	}
	setupLogging(cfgDir)
	log.Printf("versión %s", version)
	paths := cleanArgs(os.Args[1:])

	lock, err := lifecycle.Acquire(cfgDir)
	if errors.Is(err, lifecycle.ErrHeld) {
		return delegate(cfgDir, paths)
	}
	if err != nil {
		return err
	}
	defer lock.Release()

	cfg, firstRun, err := platform.LoadConfig(cfgDir)
	if err != nil {
		return err
	}
	quarantine, err := files.QuarantineDir()
	if err != nil {
		return err
	}
	cleanParts(quarantine) // .part de una ejecución anterior que se cortó

	port := 8080
	if cfg.PortOverride > 0 {
		port = cfg.PortOverride
	}
	shareL, err := share.Listen("0.0.0.0", port)
	if err != nil {
		return err
	}
	sharePort := shareL.Addr().(*net.TCPAddr).Port

	lan := addr.NewLAN(sharePort)
	book := addr.NewBook()
	book.Register(lan)
	c := core.New(core.Options{
		SharePort: sharePort, Quarantine: quarantine, ConfigDir: cfgDir,
		Config: cfg, FirstRun: firstRun, LAN: lan, Book: book, Version: version,
	})

	shareH, err := share.New(share.Deps{
		Current: c.Current, Stats: c.Stats, Quarantine: quarantine, Sender: c.Sender, Strict: c.Strict,
	})
	if err != nil {
		return err
	}
	shareSrv := share.HTTPServer(shareH)

	ctlL, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	ctlPort := ctlL.Addr().(*net.TCPAddr).Port
	token := control.NewToken()
	ctlSrv := &http.Server{Handler: control.New(c, token, ctlPort), ReadHeaderTimeout: 10 * time.Second}

	go serve(shareSrv, shareL)
	go serve(ctlSrv, ctlL)
	if err := lock.Publish(lifecycle.Info{Port: ctlPort, Token: token, PID: os.Getpid()}); err != nil {
		return err
	}
	log.Printf("compartir en :%d, control en 127.0.0.1:%d", sharePort, ctlPort)

	if len(paths) > 0 {
		c.Share(paths)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go c.Run(ctx)
	browser.Open(fmt.Sprintf("http://127.0.0.1:%d/?t=%s", ctlPort, token))

	select {
	case <-ctx.Done():
	case <-c.Done():
	}
	log.Print("cerrando")
	c.RequestQuit() // avisa a las pestañas abiertas (evento SSE "quit")
	sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ctlSrv.Shutdown(sctx)
	shareSrv.Shutdown(sctx)
	cleanParts(quarantine)
	return nil
}

func serve(s *http.Server, l net.Listener) {
	if err := s.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("servidor: %v", err)
	}
}

// delegate: ya hay una instancia; se le pasan los archivos o se reabre su UI.
func delegate(cfgDir string, paths []string) error {
	info, err := lifecycle.ReadInfo(cfgDir)
	if err != nil {
		return fmt.Errorf("otra instancia corre pero no publicó sus datos: %w", err)
	}
	url, err := lifecycle.Handoff(info, paths)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		browser.Open(url)
	}
	return nil
}

func cleanParts(dir string) {
	ms, _ := filepath.Glob(filepath.Join(dir, "*.part"))
	for _, m := range ms {
		os.Remove(m)
	}
}
```
El `RequestQuit` después de un `SIGINT` es idempotente (`sync.Once`): cierra `Done()` para que el SSE mande `quit` antes del `Shutdown`.

- [ ] **Step 5: Verificar que compila y pasa**

Run: `go test -race ./... && make build && make cross`
Expected: todo `ok`, binario `./pasame`, 7 targets compilan.

- [ ] **Step 6: Humo manual en esta Mac (camino feliz completo)**

1. `./pasame` → se abre una pestaña con "Pasá archivos a cualquier celular…".
2. Click en **Elegir archivos** → aparece el selector → elegir una foto y un video → **en ese mismo momento** aparece el QR (sin botón "Iniciar"). Contar: 3 clicks desde abrir.
3. Con el iPhone en el mismo WiFi: cámara → apuntar → tocar el aviso → se ve "Martín te comparte 2 archivos" → **Descargar todo** → baja el ZIP. En la UI del emisor, Actividad muestra "1 dispositivo conectado" y la descarga.
4. En el iPhone: **Elegir archivos para enviar** → elegir 2 fotos → barra "Enviando 1 de 2 · …" → "✓ Listo". En la Mac: "✓ Recibido IMG_….jpg" y **Abrir carpeta** abre `~/Downloads/Pasame`.
5. Desde otra computadora: tipear la dirección grande (`192.168.x.x:8080`) → redirige a la página del envío.
6. `./pasame` en otra terminal → no abre una segunda app: enfoca/abre la UI de la primera.
7. **Terminar** → "Listo. Los links ya no funcionan." → recargar la página del iPhone → "Este envío ya terminó".
8. Cerrar la pestaña del emisor → a los ~2 min el proceso termina solo (`ps aux | grep pasame`).
9. `./pasame ~/Desktop/algo.pdf` → abre directo compartiendo ese archivo, sin diálogo.

Anotar cualquier desvío en `test/manual/matriz.md` (Tarea 22). Si algo del camino feliz falla, se arregla antes de seguir.

- [ ] **Step 7: Commit**

```bash
git add cmd internal/deps_test.go internal/doc.go
git commit -m "feat(cmd): binario pasame con dos listeners, instancia única y apagado limpio

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 21: Playwright — página del receptor con y sin JavaScript

Spec: §3.3 (compromiso de compatibilidad), §9.2 (fila "Página del receptor sin JS": Chromium y WebKit, 375 × 667 y 1920 × 1080).

**Files:**
- Create: `test/e2e/server/main.go` (servidor de prueba: solo el listener `share` con una sesión fija)
- Create: `test/e2e/package.json`, `test/e2e/playwright.config.ts`, `test/e2e/receiver.spec.ts`
- Modify: `Makefile` (target `e2e`), `.github/workflows/ci.yml` (job `e2e`)

**Interfaces:**
- Consumes: `share.New`, `share.Deps`, `session.New/NewStats` (Tareas 3–10).
- Produces: `make e2e`.

- [ ] **Step 1: Instalar Node (una vez)**

Run: `brew install node && node --version`
Expected: `v20` o mayor.

- [ ] **Step 2: Servidor de prueba**

`test/e2e/server/main.go`:
```go
// Servidor mínimo para Playwright: comparte 3 archivos fijos en 127.0.0.1:18080.
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/KonixDev/pasame/internal/i18n"
	"github.com/KonixDev/pasame/internal/session"
	"github.com/KonixDev/pasame/internal/share"
)

func main() {
	dir, _ := os.MkdirTemp("", "pasame-e2e-")
	quar := filepath.Join(dir, "recibidos")
	os.MkdirAll(quar, 0o755)
	for name, content := range map[string]string{
		"contrato.pdf": "%PDF-1.4 falso", "foto.jpg": "jpg falso", "notas.txt": "hola mundo",
	} {
		os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
	}
	sess, _ := session.New([]string{
		filepath.Join(dir, "contrato.pdf"), filepath.Join(dir, "foto.jpg"), filepath.Join(dir, "notas.txt"),
	})
	stats := session.NewStats(nil)
	h, err := share.New(share.Deps{
		Current: func() *session.Session { return sess }, Stats: func() *session.Stats { return stats },
		Quarantine: quar, Sender: func() string { return "Martín" }, Strict: func() bool { return false },
		Lang: func() i18n.Lang { return i18n.ES },
	})
	if err != nil {
		log.Fatal(err)
	}
	// /__received lista la cuarentena para que el test verifique la subida del lado del emisor.
	mux := http.NewServeMux()
	mux.Handle("/", h)
	mux.HandleFunc("/__received", func(w http.ResponseWriter, _ *http.Request) {
		es, _ := os.ReadDir(quar)
		for _, e := range es {
			w.Write([]byte(e.Name() + "\n"))
		}
	})
	log.Fatal(http.ListenAndServe("127.0.0.1:18080", mux))
}
```

- [ ] **Step 3: Configuración de Playwright**

`test/e2e/package.json`:
```json
{
  "name": "pasame-e2e",
  "private": true,
  "scripts": { "test": "playwright test" },
  "devDependencies": { "@playwright/test": "^1.55.0" }
}
```

`test/e2e/playwright.config.ts`:
```ts
import { defineConfig, devices } from '@playwright/test';

const viewports = [
  { name: 'iphone-se', viewport: { width: 375, height: 667 } },
  { name: 'desktop', viewport: { width: 1920, height: 1080 } },
];
const browsers = [
  { name: 'chromium', use: devices['Desktop Chrome'] },
  { name: 'webkit', use: devices['Desktop Safari'] },
];

export default defineConfig({
  testDir: '.',
  workers: 1, // un solo servidor con estado (cuarentena compartida)
  use: { baseURL: 'http://127.0.0.1:18080', acceptDownloads: true },
  webServer: {
    command: 'go run ./server',
    cwd: '.',
    url: 'http://127.0.0.1:18080/healthz',
    reuseExistingServer: false,
  },
  projects: browsers.flatMap(b => viewports.map(v => ({
    name: `${b.name}-${v.name}`,
    use: { ...b.use, viewport: v.viewport },
  }))),
});
```

- [ ] **Step 4: Tests**

`test/e2e/receiver.spec.ts`:
```ts
import { test, expect } from '@playwright/test';

for (const js of [false, true]) {
  test.describe(js ? 'con JavaScript' : 'sin JavaScript', () => {
    test.use({ javaScriptEnabled: js });

    test('la dirección corta lleva a la página del envío', async ({ page }) => {
      await page.goto('/');
      await expect(page).toHaveURL(/\/s\/[a-z2-9]{5}$/);
      await expect(page.getByRole('heading', { level: 1 })).toHaveText('Martín te comparte 3 archivos');
    });

    test('descargar un archivo', async ({ page }) => {
      await page.goto('/');
      const [dl] = await Promise.all([
        page.waitForEvent('download'),
        page.getByRole('link', { name: 'notas.txt' }).click(),
      ]);
      expect(dl.suggestedFilename()).toBe('notas.txt');
    });

    test('descargar todo como ZIP', async ({ page }) => {
      await page.goto('/');
      const [dl] = await Promise.all([
        page.waitForEvent('download'),
        page.getByRole('link', { name: /Descargar todo/ }).click(),
      ]);
      expect(dl.suggestedFilename()).toMatch(/^Pasame-\d{4}-\d{2}-\d{2}\.zip$/);
    });

    test('los botones grandes entran en pantalla y se pueden tocar', async ({ page }) => {
      await page.goto('/');
      const big = page.getByRole('link', { name: /Descargar todo/ });
      const box = await big.boundingBox();
      const vw = page.viewportSize()!.width;
      expect(box!.height).toBeGreaterThanOrEqual(56);
      expect(box!.x + box!.width).toBeLessThanOrEqual(vw);
      // Sin scroll horizontal
      const sw = await page.evaluate('document.documentElement.scrollWidth');
      expect(sw).toBeLessThanOrEqual(vw);
    });

    test('mandar un archivo de vuelta', async ({ page, request }) => {
      await page.goto('/');
      const name = `subida-${js ? 'js' : 'nojs'}-${test.info().project.name}.txt`;
      await page.locator('#up-input').setInputFiles({ name, mimeType: 'text/plain', buffer: Buffer.from('hola') });
      if (js) {
        await expect(page.locator('#up-status')).toHaveText('✓ Listo. Martín ya los tiene.');
        await expect(page.locator('#up-send')).toBeHidden();
      } else {
        await page.locator('#up-send').click();
        await expect(page).toHaveURL(/\?subido=1$/);
        await expect(page.locator('#up-result')).toHaveText('✓ Enviaste 1 archivo.');
      }
      const received = await (await request.get('/__received')).text();
      expect(received).toContain(name);
    });
  });
}

test('la página pesa menos de 30 KB', async ({ request }) => {
  const res = await request.get('/', { maxRedirects: 5 });
  expect((await res.body()).length).toBeLessThan(30 * 1024);
});
```

- [ ] **Step 5: Makefile y CI**

`Makefile`, agregar:
```make
.PHONY: e2e
e2e:
	cd test/e2e && npm install && npx playwright install --with-deps chromium webkit && npx playwright test
```
`.github/workflows/ci.yml`, agregar el job:
```yaml
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.27.x' }
      - uses: actions/setup-node@v4
        with: { node-version: '20' }
      - run: make e2e
```

- [ ] **Step 6: Correr**

Run: `make e2e`
Expected: 4 proyectos × 11 tests = 44 pasados (el test de peso corre una vez por proyecto). Si falla "con JavaScript / mandar un archivo", revisar el reemplazo de ceros en `page.js` (Tarea 7) y el texto de `sending_js`.

- [ ] **Step 7: Commit**

```bash
git add test/e2e Makefile .github/workflows/ci.yml
git commit -m "test(e2e): página del receptor en Chromium y WebKit, con y sin JavaScript

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 22: Checklist manual del núcleo LAN y README inicial

Spec: §9.3 (matriz manual versionada), §3.4 (frase del README sobre quién necesita la app), §5 ("Qué NO se promete", 3 líneas sin jerga), §6.6 (cerrar pestaña).

**Files:**
- Create: `test/manual/matriz.md`, `README.md`

**Interfaces:**
- Consumes: el binario de la Tarea 20.
- Produces: la matriz que el Plan 2 amplía con túnel, empaquetado y Gatekeeper/SmartScreen.

- [ ] **Step 1: `test/manual/matriz.md`**

```markdown
# Matriz manual — Pasame

Marcar cada celda con fecha (AAAA-MM-DD) y ✓ / ✗ + nota. Sin la matriz completa no se publica una release estable (spec §9.3).

## Emisores × receptores (camino feliz: QR → Descargar todo → mandar 2 fotos de vuelta)

| Emisor ↓ / Receptor → | iPhone iOS actual | iPhone iOS −2 | Android actual | Android 9 | PC (Chrome) |
|---|---|---|---|---|---|
| macOS actual | | | | | |
| Windows 11 | | | | | |
| Ubuntu LTS | | | | | |

## Chequeos por emisor

| Chequeo | macOS | Windows 11 | Ubuntu |
|---|---|---|---|
| Doble clic abre la UI (sin consola visible) | | | |
| El selector de archivos aparece **adelante** del navegador | | | |
| Selector: varios archivos y carpeta | | | |
| Aviso de permiso de red: el texto anticipado coincide con la ventana real | | | |
| "Cancelar" en el permiso → a los 45 s aparece "¿No pueden entrar?" | | | |
| Arrastrar un archivo sobre el ícono abre compartiendo ese archivo | | | |
| Segundo doble clic reabre la misma UI (no dos apps) | | | |
| Cerrar la pestaña → el proceso termina a los ~2 min | | | |
| VPN (WireGuard/Tailscale) activa: elige la red WiFi y muestra el aviso | | | |
| Cambiar de WiFi con la app abierta: el QR se actualiza y avisa | | | |

## Chequeos por receptor

| Chequeo | iOS Safari | Android Chrome | Samsung Internet | Smart TV |
|---|---|---|---|---|
| La cámara lee el QR a 50 cm con brillo al 50 % | | | | n/a |
| "Ver" en una foto permite "Guardar en Fotos" (iOS) | | | | |
| El archivo queda en Archivos → Descargas / carpeta Descargas | | | | n/a |
| Video de 8 GB: se corta el WiFi a mitad y la descarga se reanuda | | | | |
| Subir 5 fotos desde la galería: barra de progreso y "✓ Listo" | | | | n/a |
| "Ver" reproduce un .mp4 | | | | |
```

- [ ] **Step 2: `README.md`**

```markdown
# Pasame

Pasá archivos a cualquier celular o computadora que esté cerca. Sin cable, sin cuentas, sin instalar nada del lado de quien recibe.

1. Abrí Pasame en tu computadora y elegí los archivos.
2. La otra persona apunta la cámara del celular al código que aparece (o escribe la dirección en su navegador).
3. Listo: descarga lo que compartiste y, si quiere, te manda archivos de vuelta.

**Una de las dos personas tiene que tener Pasame en una computadora. La otra, solo un navegador.**

## Lo que conviene saber

- Los dos tienen que estar en la misma red WiFi.
- Si cerrás la pestaña de Pasame, la app se cierra sola a los 2 minutos (si no hay nada bajando).
- Lo que te mandan queda en la carpeta **Descargas → Pasame**.

## Qué no promete

En tu WiFi, cualquiera conectado a esa misma red que tenga el link puede ver lo que compartís mientras la app está abierta. No uses Pasame en una red en la que no confíes. Los archivos no pasan por ningún lado más que por tu red.

## Para desarrolladores

    make test     # tests unitarios y de integración
    make build    # ./pasame
    make cross    # compila los 7 targets
    make e2e      # Playwright sobre la página del receptor

Diseño: `docs/superpowers/specs/2026-09-21-share-now-design.md`. Licencia MIT.
```
(El Plan 2 agrega las secciones "Compartir por internet", descargas por sistema y "Cómo abrirla en Mac/Windows".)

- [ ] **Step 3: Recorrer la matriz para macOS como emisor**

Completar la fila "macOS actual" y la columna de chequeos de macOS con el iPhone y un Android disponibles. Las celdas de Windows y Ubuntu quedan para cuando haya máquinas (o VMs) — se completan antes del primer release del Plan 2.

- [ ] **Step 4: Commit**

```bash
git add test/manual README.md
git commit -m "docs: checklist manual del núcleo LAN y README inicial

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Hito del Plan 1

Al terminar la Tarea 22: `make test` y `make e2e` en verde, `make cross` compila los 7 targets y el humo manual (Tarea 20, Step 6) pasó en macOS con un iPhone. Es una v0.1 funcional por WiFi. Seguir con `docs/superpowers/plans/2026-09-21-pasame-02-tunel-y-distribucion.md`.
