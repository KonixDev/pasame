# Pasame — Plan 2: Túnel, PIN y distribución — Plan de implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Sumar a la v0.1 del Plan 1 el botón "Compartir por internet" (quick tunnel de Cloudflare con PIN obligatorio), el bonus `pasame.local`, y todo lo necesario para que una persona no técnica descargue y abra Pasame en Windows, macOS y Linux desde `pasame.com.ar`.

**Architecture:** El túnel es un proveedor más del `addr.Book` (Plan 1, Tarea 11): `internal/tunnel` baja `cloudflared` con versión y SHA-256 fijados, lo ejecuta apuntando **solo** al listener `share`, y `addr/tunnel.go` lo adapta a `Provider`. Mientras el túnel está activo, `core` pone la sesión en modo estricto y `share` exige PIN en todas las rutas mediante una cookie HMAC con clave que rota. La distribución es GoReleaser en GitHub Actions (7 binarios, `.app` universal notarizado, `.dmg`, `.deb`, checksums firmados con cosign) y un sitio estático en GitHub Pages con dominio `pasame.com.ar`.

**Tech Stack:** Todo lo del Plan 1 + `cloudflared` 2026.9.1 (runtime, no compilación), `github.com/hashicorp/mdns` (solo con `-tags mdns`), GoReleaser v2, `codesign`/`notarytool` de Apple en un runner macOS (ver Tarea 8), cosign (Sigstore keyless), `osacompile`, GitHub Pages.

**Spec:** `docs/superpowers/specs/2026-09-21-share-now-design.md`.
**Requisito:** Plan 1 (`docs/superpowers/plans/2026-09-21-pasame-01-nucleo-lan.md`) terminado y en verde.

## Global Constraints

- Todas las restricciones globales del Plan 1 siguen vigentes (módulo `github.com/KonixDev/pasame`, Go 1.27, `CGO_ENABLED=0`, voz de la UI, reglas de dependencia, formato de commits).
- **El túnel apunta al listener `share` (`http://127.0.0.1:<puerto share>`), nunca al de `control`.**
- Túnel activo ⇒ PIN obligatorio en **todas** las rutas `/s/<token>…`, también desde la LAN. Cookie: `pasame_pin=<hex(HMAC-SHA256(clave, token))>`, `HttpOnly`, `SameSite=Lax`, `Path=/s/<token>`, sin `Secure`. La clave (32 bytes, `crypto/rand`) rota cada vez que el túnel se prende o se apaga. 5 PIN incorrectos ⇒ 60 s de bloqueo **global** (a través del túnel todas las IP son loopback).
- `cloudflared`: versión fijada `2026.9.1`, SHA-256 fijado por asset en `internal/tunnel/versions.go`. Flags exactos: `tunnel --url http://127.0.0.1:<port> --no-autoupdate --config <NUL|/dev/null> --loglevel info`. Regex: `https://[a-z0-9-]+\.trycloudflare\.com`. Timeout 30 s. Verificación con `GET <url>/healthz` antes de mostrar. **Los 7 targets tienen túnel** (corrección al spec, verificada con la API de GitHub el 2026-09-21): `linux/arm` usa `cloudflared-linux-armhf` y `windows/arm64` usa `cloudflared-windows-386.exe` por emulación x86. Un OS/arquitectura sin entrada en la tabla ⇒ "No disponible en esta computadora".
- Binario de `cloudflared` en `<config dir>/cloudflared/<versión>/cloudflared[.exe]`.
- Textos del túnel (spec §6.3), literal: *"Preparando… (la primera vez baja un componente de 40 MB, tarda un minuto)"*, *"Conectando…"*, *"y cuando te pida la clave:"*, *"Los archivos pasan por los servidores de Cloudflare (servicio gratuito, puede no estar disponible)."* — excepción documentada a la regla de voz: la palabra "servidores" queda porque es el aviso de privacidad y tiene que ser exacto —, *"Volver a compartir solo por WiFi"*, *"No se pudo conectar por internet. Volvé a intentar en un momento."*, *"Ahora todos necesitan la clave, también en tu WiFi. El código QR ya la lleva."*, *"Se cortó la conexión por internet. Podés volver a activarla."*
- Flags de build: `-trimpath -ldflags="-s -w -X main.version=<tag>"`; Windows además `-H=windowsgui`. **Sin UPX.**
- Targets: `windows/amd64`, `windows/arm64`, `darwin/amd64`+`darwin/arm64` → universal, `linux/amd64`, `linux/arm64`, `linux/arm` (GOARM=7).
- Nombres de assets: `Pasame-Windows.exe`, `Pasame-Windows-arm64.exe`, `Pasame-macOS.dmg`, `pasame-linux-<arch>.tar.gz`, `pasame_<ver>_<arch>.deb`, `checksums.txt`.
- Dominio: `pasame.com.ar` → GitHub Pages del repo `KonixDev/pasame`, carpeta `site/`. La app en ejecución nunca consulta el dominio.

## Mapa de archivos de este plan

| Archivo | Responsabilidad | Tarea |
|---|---|---|
| `internal/share/pin.go`, `web/share/pin.html` | Modo estricto, cookie HMAC, límite de intentos | 1 |
| `internal/tunnel/versions.go`, `download.go`, `scripts/cloudflared-sums.sh` | Asset por OS/arch, descarga verificada | 2 |
| `internal/tunnel/run.go`, `test/fixtures/fake-cloudflared/main.go` | Ejecutar, leer la URL, verificar, apagar | 3 |
| `internal/addr/tunnel.go`, `internal/core/tunnel.go`, `internal/control/api.go`, `web/control/app.js` | Proveedor, modo estricto, API y UI del túnel | 4 |
| `internal/addr/mdns.go`, `mdns_stub.go` | `pasame.local` (bonus recortable) | 5 |
| `packaging/macos/*` | `.app` droplet, `Info.plist`, `.dmg` | 6 |
| `packaging/windows/*`, `packaging/linux/*` | Icono y metadatos del `.exe`; `.desktop` y `.deb` | 7 |
| `.goreleaser.yaml`, `.github/workflows/release.yml` | Releases, notarización, cosign | 8 |
| `site/*` | `pasame.com.ar`: descarga por OS y "cómo abrirla" | 9 |
| `README.md`, `test/manual/*` | Documentación final y matriz ampliada | 10 |

---

### Task 1: PIN y modo estricto en `share`

Spec: §4.4 (fila `GET/POST /s/<token>/pin`, `/` en modo estricto), §5 (fila "Alguien de internet ve los archivos"), §6.5 (textos y códigos 401/429), §13 (cookie de PIN).

**Files:**
- Create: `internal/share/pin.go`, `web/share/pin.html`
- Modify: `internal/share/server.go` (campo `PINKey` en `Deps`, envolver rutas con `gated`), `internal/share/helpers_test.go` (clave en el fixture)
- Test: `internal/share/pin_test.go`

**Interfaces:**
- Consumes: `Deps.Strict` (Plan 1), `session.Session.PIN`.
- Produces:
  ```go
  // Deps gana:
  PINKey func() []byte // clave HMAC vigente; core la rota al prender/apagar el túnel
  // internos:
  func pinCookieValue(key []byte, token string) string
  type limiter struct{ /* 5 fallos → 60 s */ }
  func (l *limiter) locked(now time.Time) bool
  func (l *limiter) fail(now time.Time)
  func (l *limiter) reset()
  ```
  Rutas: `GET /s/{tok}` y cualquier ruta de la sesión con `?pin=NNNN` válido → cookie + `303` a la misma ruta sin `pin`. `POST /s/{tok}/pin` (form `pin`) → cookie + `303` a `/s/{tok}`. Sin cookie válida en modo estricto → `401` con `pin.html` (o JSON `{"ok":false,"error":...}` si `Accept: application/json`).

- [ ] **Step 1: Ajustar el fixture**

En `internal/share/helpers_test.go`, agregar el campo `key []byte` a `fixture`, inicializarlo con `key: []byte("clave-de-prueba-32-bytes-000000")` y pasar `PINKey: func() []byte { return f.key }` en `Deps`.

- [ ] **Step 2: Test que falla**

`internal/share/pin_test.go`:
```go
package share

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func timeZero() time.Time { return time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC) }

func withCookie(r *http.Request, c *http.Cookie) *http.Request {
	if c != nil {
		r.AddCookie(c)
	}
	return r
}

func cookieFrom(w *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == "pasame_pin" {
			return c
		}
	}
	return nil
}

func TestLANModeNeedsNoPIN(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	if w := f.do(httptest.NewRequest("GET", f.sess.Path(), nil)); w.Code != 200 {
		t.Fatalf("%d", w.Code)
	}
}

func TestStrictBlocksEveryRoute(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.strict = true
	for _, r := range []*http.Request{
		httptest.NewRequest("GET", f.sess.Path(), nil),
		httptest.NewRequest("GET", f.sess.Path()+"/f/0", nil),
		httptest.NewRequest("GET", f.sess.Path()+"/zip", nil),
		httptest.NewRequest("POST", f.sess.Path()+"/up", strings.NewReader("")),
	} {
		w := f.do(r)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s → %d", r.Method, r.URL.Path, w.Code)
		}
	}
	w := f.do(httptest.NewRequest("GET", f.sess.Path(), nil))
	if !strings.Contains(w.Body.String(), "Escribí la clave de 4 números que aparece en la pantalla de Martín.") {
		t.Fatalf("body: %s", w.Body)
	}
}

func TestPINFromQRSetsCookieAndCleansURL(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.strict = true
	w := f.do(httptest.NewRequest("GET", f.sess.Path()+"?pin="+f.sess.PIN, nil))
	c := cookieFrom(w)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != f.sess.Path() || c == nil {
		t.Fatalf("%d %q %v", w.Code, w.Header().Get("Location"), c)
	}
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != f.sess.Path() || c.Secure {
		t.Fatalf("cookie %+v", c)
	}
	for _, p := range []string{"", "/f/0", "/zip"} {
		if w := f.do(withCookie(httptest.NewRequest("GET", f.sess.Path()+p, nil), c)); w.Code != 200 {
			t.Errorf("%s con cookie → %d", p, w.Code)
		}
	}
}

func TestPINTyped(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.strict = true
	form := url.Values{"pin": {f.sess.PIN}}.Encode()
	r := httptest.NewRequest("POST", f.sess.Path()+"/pin", strings.NewReader(form))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := f.do(r)
	if w.Code != http.StatusSeeOther || cookieFrom(w) == nil {
		t.Fatalf("%d", w.Code)
	}
}

func TestWrongPINAndLockout(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.strict = true
	wrong := "0000"
	if f.sess.PIN == wrong {
		wrong = "1111"
	}
	for i := 0; i < 5; i++ {
		w := f.do(httptest.NewRequest("GET", f.sess.Path()+"?pin="+wrong, nil))
		if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "La clave no es esa") {
			t.Fatalf("intento %d → %d", i, w.Code)
		}
	}
	// Bloqueado: ni siquiera el PIN correcto entra durante 60 s.
	w := f.do(httptest.NewRequest("GET", f.sess.Path()+"?pin="+f.sess.PIN, nil))
	if w.Code != http.StatusTooManyRequests || !strings.Contains(w.Body.String(), "Demasiados intentos") {
		t.Fatalf("%d", w.Code)
	}
}

func TestKeyRotationInvalidatesCookies(t *testing.T) {
	f := newFixture(t, map[string]int{"a.txt": 1})
	f.strict = true
	c := cookieFrom(f.do(httptest.NewRequest("GET", f.sess.Path()+"?pin="+f.sess.PIN, nil)))
	f.key = []byte("otra-clave-distinta-32-bytes-000")
	if w := f.do(withCookie(httptest.NewRequest("GET", f.sess.Path(), nil), c)); w.Code != http.StatusUnauthorized {
		t.Fatalf("cookie vieja aceptada: %d", w.Code)
	}
}

func TestLimiter(t *testing.T) {
	var l limiter
	now := timeZero()
	for i := 0; i < 4; i++ {
		l.fail(now)
	}
	if l.locked(now) {
		t.Fatal("bloqueó con 4")
	}
	l.fail(now)
	if !l.locked(now) || !l.locked(now.Add(59e9)) || l.locked(now.Add(61e9)) {
		t.Fatal("ventana de 60 s incorrecta")
	}
}
```
- [ ] **Step 3: Verificar que falla**

Run: `go test ./internal/share/ -run 'PIN|Strict|Limiter|Key|LAN'`
Expected: FAIL — `unknown field PINKey`, `undefined: limiter`.

- [ ] **Step 4: Plantilla `web/share/pin.html`**

```html
{{define "pin.html"}}<!DOCTYPE html>
<html lang="{{.Lang}}"><head>{{template "head" .}}</head>
<body class="plain"><main>
<p class="brand">{{template "brand"}}</p>
<h1>{{if .Msg}}{{.Msg}}{{else}}{{t .Lang "pin_prompt" .Sender}}{{end}}</h1>
<form class="pin" method="post" action="{{.Sess.Path}}/pin">
<input name="pin" inputmode="numeric" pattern="[0-9]*" maxlength="4" autocomplete="one-time-code" autofocus required>
<button class="big" type="submit">{{t .Lang "pin_enter"}}</button>
</form>
</main></body></html>{{end}}
```

- [ ] **Step 5: Implementar `internal/share/pin.go`**

```go
package share

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/KonixDev/pasame/internal/i18n"
	"github.com/KonixDev/pasame/internal/session"
)

const (
	maxFails    = 5
	lockoutTime = 60 * time.Second
)

// limiter es global a propósito: por el túnel todas las IP llegan como 127.0.0.1.
type limiter struct {
	mu          sync.Mutex
	fails       int
	lockedUntil time.Time
}

func (l *limiter) locked(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.lockedUntil.IsZero() && !now.Before(l.lockedUntil) {
		l.lockedUntil, l.fails = time.Time{}, 0
	}
	return now.Before(l.lockedUntil)
}

func (l *limiter) fail(now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.fails++; l.fails >= maxFails {
		l.lockedUntil = now.Add(lockoutTime)
	}
}

func (l *limiter) reset() { l.mu.Lock(); l.fails = 0; l.mu.Unlock() }

func pinCookieValue(key []byte, token string) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(token))
	return hex.EncodeToString(m.Sum(nil))
}

func (s *Server) hasPINCookie(r *http.Request, sess *session.Session) bool {
	c, err := r.Cookie("pasame_pin")
	if err != nil {
		return false
	}
	want := pinCookieValue(s.d.PINKey(), sess.Token)
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(want)) == 1
}

// checkPIN valida un PIN recibido; si es correcto deja la cookie. Devuelve el código HTTP a usar si falla.
func (s *Server) checkPIN(w http.ResponseWriter, sess *session.Session, pin string) (ok bool, status int, msgKey string) {
	now := time.Now()
	if s.lim.locked(now) {
		return false, http.StatusTooManyRequests, "pin_locked"
	}
	if subtle.ConstantTimeCompare([]byte(pin), []byte(sess.PIN)) != 1 {
		s.lim.fail(now) // el 5.º fallo todavía responde "La clave no es esa"; el bloqueo se ve en el 6.º intento
		return false, http.StatusUnauthorized, "pin_wrong"
	}
	s.lim.reset()
	http.SetCookie(w, &http.Cookie{
		Name: "pasame_pin", Value: pinCookieValue(s.d.PINKey(), sess.Token),
		Path: sess.Path(), HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	return true, 0, ""
}

// gated agrega el control de PIN a una ruta de sesión. En modo LAN no hace nada.
func (s *Server) gated(h sessHandler) sessHandler {
	return func(w http.ResponseWriter, r *http.Request, sess *session.Session, st *session.Stats) {
		if !s.d.Strict() || s.hasPINCookie(r, sess) {
			h(w, r, sess, st)
			return
		}
		if pin := r.URL.Query().Get("pin"); pin != "" && r.Method == http.MethodGet {
			if ok, status, key := s.checkPIN(w, sess, pin); !ok {
				s.pinPage(w, r, sess, status, key)
				return
			}
			q := r.URL.Query()
			q.Del("pin")
			target := r.URL.Path
			if len(q) > 0 {
				target += "?" + q.Encode()
			}
			http.Redirect(w, r, target, http.StatusSeeOther)
			return
		}
		s.pinPage(w, r, sess, http.StatusUnauthorized, "")
	}
}

func (s *Server) pinPost(w http.ResponseWriter, r *http.Request, sess *session.Session, _ *session.Stats) {
	if ok, status, key := s.checkPIN(w, sess, strings.TrimSpace(r.PostFormValue("pin"))); !ok {
		s.pinPage(w, r, sess, status, key)
		return
	}
	http.Redirect(w, r, sess.Path(), http.StatusSeeOther)
}

func (s *Server) pinPage(w http.ResponseWriter, r *http.Request, sess *session.Session, status int, msgKey string) {
	v := s.view(r, sess)
	if msgKey == "pin_wrong" {
		v.Msg = i18n.T(v.Lang, msgKey, v.Sender)
	} else if msgKey != "" {
		v.Msg = i18n.T(v.Lang, msgKey)
	}
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		msg := v.Msg
		if msg == "" {
			msg = i18n.T(v.Lang, "pin_prompt", v.Sender)
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": msg})
		return
	}
	s.render(w, status, "pin.html", v)
}
```

En `server.go`:
- Agregar `PINKey func() []byte` a `Deps` y `lim limiter` a `Server`.
- Cambiar el registro de rutas de sesión para que pasen por `gated`, y sumar la ruta del PIN:
```go
	s.mux.HandleFunc("GET /s/{tok}", s.withSession(s.gated(s.page)))
	s.mux.HandleFunc("GET /s/{tok}/f/{i}", s.withSession(s.gated(s.file)))
	s.mux.HandleFunc("GET /s/{tok}/zip", s.withSession(s.gated(s.zip)))
	s.mux.HandleFunc("POST /s/{tok}/up", s.withSession(s.gated(s.upload)))
	s.mux.HandleFunc("POST /s/{tok}/pin", s.withSession(s.pinPost))
```
- En `test/e2e/server/main.go` (Plan 1, Tarea 21) agregar `PINKey: func() []byte { return []byte("e2e") }` a `Deps`.

- [ ] **Step 6: Verificar que pasa**

Run: `go test -race ./internal/share/ && make e2e`
Expected: `ok` y Playwright sigue en verde (modo LAN sin cambios).

- [ ] **Step 7: Commit**

```bash
git add internal/share web/share test/e2e/server
git commit -m "feat(share): modo estricto con PIN, cookie HMAC rotable y bloqueo por intentos

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: `cloudflared` — asset por plataforma y descarga verificada

Spec: §4.6 (ciclo del binario, pasos 1–2), §5 (fila "`cloudflared` adulterado"), §7 (fila "`cloudflared` no descarga"), §9.1 (fila `tunnel`: tabla de assets y SHA mismatch).

**Files:**
- Create: `scripts/cloudflared-sums.sh`, `internal/tunnel/versions.go`, `internal/tunnel/download.go`
- Test: `internal/tunnel/download_test.go`

**Interfaces:**
- Consumes: nada.
- Produces:
  ```go
  const Version = "2026.9.1"
  type Asset struct { Name, SHA256 string; Tgz bool }
  var ErrUnavailable = errors.New("tunnel: no hay cloudflared para esta computadora")
  var ErrChecksum    = errors.New("tunnel: el archivo descargado no coincide")
  type Progress func(done, total int64)
  func Available() bool                                                // hay asset para runtime.GOOS/GOARCH
  func Ensure(ctx context.Context, configDir string, p Progress) (string, error) // ruta al ejecutable
  ```

- [ ] **Step 1: Script que obtiene los SHA-256 reales**

`scripts/cloudflared-sums.sh`:
```bash
#!/usr/bin/env bash
# Descarga los assets de cloudflared de la versión fijada y escribe el bloque Go con sus SHA-256.
# Uso: scripts/cloudflared-sums.sh 2026.9.1 > /tmp/sums.go.txt
set -euo pipefail
v="${1:?versión}"
base="https://github.com/cloudflare/cloudflared/releases/download/$v"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
emit() { # clave-go asset tgz
  curl -fsSL -o "$tmp/$2" "$base/$2"
  sum="$(shasum -a 256 "$tmp/$2" | cut -d' ' -f1)"
  printf '\t"%s": {Name: "%s", SHA256: "%s", Tgz: %s},\n' "$1" "$2" "$sum" "$3"
}
emit darwin/amd64  cloudflared-darwin-amd64.tgz  true
emit darwin/arm64  cloudflared-darwin-arm64.tgz  true
emit windows/amd64 cloudflared-windows-amd64.exe false
emit windows/arm64 cloudflared-windows-386.exe   false
emit linux/amd64   cloudflared-linux-amd64       false
emit linux/arm64   cloudflared-linux-arm64       false
emit linux/arm     cloudflared-linux-armhf       false
```
Run: `chmod +x scripts/cloudflared-sums.sh && scripts/cloudflared-sums.sh 2026.9.1`
Expected: exactamente las 7 líneas del mapa de la Tarea 2, Step 4 (ya están en el plan: se verificaron el 2026-09-21 contra el campo `digest` de la API de GitHub). El script queda para cuando se actualice la versión; como verificación cruzada, comparar con:
```bash
curl -s https://api.github.com/repos/cloudflare/cloudflared/releases/tags/2026.9.1 \
  | python3 -c 'import json,sys; [print(a["name"], a["digest"]) for a in json.load(sys.stdin)["assets"]]'
```

- [ ] **Step 2: Test que falla**

`internal/tunnel/download_test.go`:
```go
package tunnel

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"sync/atomic"
	"testing"
)

func TestPinnedAssets(t *testing.T) {
	hex64 := regexp.MustCompile(`^[0-9a-f]{64}$`)
	// Los 7 targets de Pasame tienen túnel.
	for _, k := range []string{"darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm64", "linux/amd64", "linux/arm64", "linux/arm"} {
		a, ok := assets[k]
		if !ok || !hex64.MatchString(a.SHA256) {
			t.Errorf("%s: asset o SHA-256 faltante (%+v)", k, a)
		}
	}
}

func sum(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func tgz(t *testing.T, name string, content []byte) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))})
	tw.Write(content)
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

type fakeRelease struct {
	srv  *httptest.Server
	hits atomic.Int32
}

func serve(t *testing.T, files map[string][]byte) *fakeRelease {
	f := &fakeRelease{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		for name, b := range files {
			if r.URL.Path == "/"+Version+"/"+name {
				w.Header().Set("Content-Length", strconv.Itoa(len(b)))
				w.Write(b)
				return
			}
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func TestEnsurePlainBinary(t *testing.T) {
	bin := []byte("#!/bin/sh\necho falso\n")
	rel := serve(t, map[string][]byte{"cloudflared-linux-amd64": bin})
	tbl := map[string]Asset{"linux/amd64": {Name: "cloudflared-linux-amd64", SHA256: sum(bin)}}
	dir := t.TempDir()
	var calls int
	p, err := ensure(context.Background(), dir, "linux", "amd64", rel.srv.URL+"/", tbl, func(d, tot int64) { calls++ })
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	fi, _ := os.Stat(p)
	if !bytes.Equal(got, bin) || fi.Mode()&0o111 == 0 || calls == 0 {
		t.Fatalf("contenido/permisos/progreso mal: exec=%v calls=%d", fi.Mode(), calls)
	}
	// Segunda vez: ya está, no descarga.
	before := rel.hits.Load()
	if _, err := ensure(context.Background(), dir, "linux", "amd64", rel.srv.URL+"/", tbl, nil); err != nil || rel.hits.Load() != before {
		t.Fatal("volvió a descargar")
	}
}

func TestEnsureTgz(t *testing.T) {
	bin := []byte("binario mac falso")
	archive := tgz(t, "cloudflared", bin)
	rel := serve(t, map[string][]byte{"cloudflared-darwin-arm64.tgz": archive})
	tbl := map[string]Asset{"darwin/arm64": {Name: "cloudflared-darwin-arm64.tgz", SHA256: sum(archive), Tgz: true}}
	p, err := ensure(context.Background(), t.TempDir(), "darwin", "arm64", rel.srv.URL+"/", tbl, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(p); !bytes.Equal(got, bin) {
		t.Fatal("no extrajo el binario del .tgz")
	}
}

func TestEnsureChecksumMismatch(t *testing.T) {
	rel := serve(t, map[string][]byte{"cloudflared-linux-amd64": []byte("adulterado")})
	tbl := map[string]Asset{"linux/amd64": {Name: "cloudflared-linux-amd64", SHA256: sum([]byte("original"))}}
	dir := t.TempDir()
	_, err := ensure(context.Background(), dir, "linux", "amd64", rel.srv.URL+"/", tbl, nil)
	if !errors.Is(err, ErrChecksum) {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(binPath(dir, "linux")); err == nil {
		t.Fatal("dejó el binario adulterado")
	}
	if left, _ := os.ReadDir(binDir(dir)); len(left) != 0 {
		t.Fatalf("dejó temporales: %v", left)
	}
}

func TestEnsureUnavailable(t *testing.T) {
	_, err := ensure(context.Background(), t.TempDir(), "freebsd", "amd64", "http://x/", assets, nil)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v", err)
	}
}

func TestEnsureHTTPError(t *testing.T) {
	rel := serve(t, nil)
	tbl := map[string]Asset{"linux/amd64": {Name: "cloudflared-linux-amd64", SHA256: sum(nil)}}
	if _, err := ensure(context.Background(), t.TempDir(), "linux", "amd64", rel.srv.URL+"/", tbl, nil); err == nil {
		t.Fatal("un 404 no dio error")
	}
}
```
- [ ] **Step 3: Verificar que falla**

Run: `go test ./internal/tunnel/`
Expected: FAIL — `undefined: assets`, `ensure`, …

- [ ] **Step 4: `internal/tunnel/versions.go`**

```go
// Package tunnel maneja cloudflared: bajarlo verificado, correrlo y leer su URL.
package tunnel

// Version se actualiza a mano junto con los SHA-256 (scripts/cloudflared-sums.sh).
// Nunca "latest": el comportamiento del túnel tiene que ser reproducible y testeable.
const Version = "2026.9.1"

type Asset struct {
	Name   string
	SHA256 string
	Tgz    bool
}

// SHA-256 verificados el 2026-09-21 contra el campo "digest" de la API de GitHub.
// windows/arm64 usa el binario x86 (Windows en ARM lo emula); linux/arm usa armhf (armv7).
var assets = map[string]Asset{
	"darwin/amd64":  {Name: "cloudflared-darwin-amd64.tgz", SHA256: "ff0d3b51d5ff70eceef89d6b32145fee985018a2174596a5dbe405e2766e2ac4", Tgz: true},
	"darwin/arm64":  {Name: "cloudflared-darwin-arm64.tgz", SHA256: "c27ab8fd0aa489449e3d201eb02f957ef460a13b613662928b1b23394bf1bcfe", Tgz: true},
	"windows/amd64": {Name: "cloudflared-windows-amd64.exe", SHA256: "2837888cc0f5d58f15b6dc478376de90b4d3ba5241c7947455d1e0a0df429712"},
	"windows/arm64": {Name: "cloudflared-windows-386.exe", SHA256: "11b6e4b2d306950bd87e7caa4deee8e80a32d71ffee555a96237a76651eeae4c"},
	"linux/amd64":   {Name: "cloudflared-linux-amd64", SHA256: "03f1f25d1cc93b9ad6c60569d44060bc4f17ed97075760ed8cfca4b12dcd68cc"},
	"linux/arm64":   {Name: "cloudflared-linux-arm64", SHA256: "3d97437c71848bd8df68041e12436b484a661d95073ea1937f01a845ce88faa3"},
	"linux/arm":     {Name: "cloudflared-linux-armhf", SHA256: "95420507a720fb543122a5d69372fbde8f5c919790e95ddd4374a449e0a6f4dd"},
}
```

- [ ] **Step 5: `internal/tunnel/download.go`**

```go
package tunnel

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var (
	ErrUnavailable = errors.New("tunnel: no hay cloudflared para esta computadora")
	ErrChecksum    = errors.New("tunnel: el archivo descargado no coincide")
)

const releaseBase = "https://github.com/cloudflare/cloudflared/releases/download/"

type Progress func(done, total int64)

func Available() bool {
	_, ok := assets[runtime.GOOS+"/"+runtime.GOARCH]
	return ok
}

func Ensure(ctx context.Context, configDir string, p Progress) (string, error) {
	return ensure(ctx, configDir, runtime.GOOS, runtime.GOARCH, releaseBase, assets, p)
}

func binDir(configDir string) string { return filepath.Join(configDir, "cloudflared", Version) }

func binPath(configDir, goos string) string {
	name := "cloudflared"
	if goos == "windows" {
		name += ".exe"
	}
	return filepath.Join(binDir(configDir), name)
}

func ensure(ctx context.Context, configDir, goos, goarch, base string, table map[string]Asset, p Progress) (string, error) {
	a, ok := table[goos+"/"+goarch]
	if !ok {
		return "", ErrUnavailable
	}
	dst := binPath(configDir, goos)
	if fi, err := os.Stat(dst); err == nil && fi.Size() > 0 {
		return dst, nil
	}
	if err := os.MkdirAll(binDir(configDir), 0o755); err != nil {
		return "", err
	}
	tmp, err := download(ctx, base+Version+"/"+a.Name, binDir(configDir), a.SHA256, p)
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp)
	if a.Tgz {
		err = extract(tmp, dst+".tmp")
	} else {
		err = os.Rename(tmp, dst+".tmp")
	}
	if err == nil {
		err = os.Chmod(dst+".tmp", 0o755)
	}
	if err == nil {
		err = os.Rename(dst+".tmp", dst)
	}
	if err != nil {
		os.Remove(dst + ".tmp")
		return "", err
	}
	if goos == "darwin" {
		exec.Command("xattr", "-d", "com.apple.quarantine", dst).Run() // normalmente no lo tiene; se intenta igual
	}
	return dst, nil
}

// download baja url a un temporal en dir, calculando el SHA-256 mientras copia.
func download(ctx context.Context, url, dir, wantSHA string, p Progress) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tunnel: descarga respondió %s", resp.Status)
	}
	f, err := os.CreateTemp(dir, "dl-*")
	if err != nil {
		return "", err
	}
	h := sha256.New()
	pr := &progressReader{r: resp.Body, total: resp.ContentLength, fn: p}
	_, err = io.Copy(io.MultiWriter(f, h), pr)
	f.Close()
	if err == nil && hex.EncodeToString(h.Sum(nil)) != wantSHA {
		err = ErrChecksum
	}
	if err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func extract(tgzPath, dst string) error {
	f, err := os.Open(tgzPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err != nil {
			return fmt.Errorf("tunnel: el .tgz no trae cloudflared: %w", err)
		}
		if filepath.Base(h.Name) != "cloudflared" || h.Typeflag != tar.TypeReg {
			continue
		}
		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, io.LimitReader(tr, 200<<20))
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		return err
	}
}

type progressReader struct {
	r     io.Reader
	done  int64
	total int64
	fn    Progress
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.done += int64(n)
	if p.fn != nil && n > 0 {
		p.fn(p.done, p.total)
	}
	return n, err
}
```
En `extract` el `Typeflag` de un header escrito sin tipo es `tar.TypeReg` (`'0'`): coincide con lo que genera el test.

- [ ] **Step 6: Verificar que pasa**

Run: `go test -race ./internal/tunnel/`
Expected: `ok`.

- [ ] **Step 7: Commit**

```bash
git add scripts internal/tunnel
git commit -m "feat(tunnel): descarga de cloudflared con versión y SHA-256 fijados

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Ejecutar `cloudflared`, obtener la URL, verificarla y apagarlo

Spec: §4.6 (pasos 3–6, `config.yaml`, límites de TryCloudflare), §7 (filas "Quick tunnel no da URL en 30 s" y "Túnel muere solo"), §9.1 (parseo de logs reales), §9.2 (túnel con `cloudflared` falso).

**Files:**
- Create: `internal/tunnel/run.go`, `internal/tunnel/null_unix.go`, `internal/tunnel/null_windows.go`
- Create: `test/fixtures/fake-cloudflared/main.go`
- Test: `internal/tunnel/run_test.go`

**Interfaces:**
- Consumes: la ruta que devuelve `Ensure` (Tarea 2).
- Produces:
  ```go
  type Options struct {
      Bin     string
      Port    int                                   // puerto del listener share
      Timeout time.Duration                         // 30 s por defecto
      Verify  func(ctx context.Context, url string) error // por defecto: GET <url>/healthz == 200, con reintentos
  }
  type Tunnel struct{ URL string /* + privados */ }
  func Start(ctx context.Context, o Options) (*Tunnel, error)
  func (t *Tunnel) Done() <-chan struct{}   // se cierra si el proceso termina (por Stop o solo)
  func (t *Tunnel) Stop()                   // interrupción → 5 s → kill (Windows: taskkill /T /F)
  func (t *Tunnel) LastLog() string         // últimas líneas de stderr, para "Ver detalle"
  func ParseURL(line string) (string, bool)
  ```

- [ ] **Step 1: `cloudflared` falso**

`test/fixtures/fake-cloudflared/main.go`:
```go
// cloudflared falso para tests: imprime logs parecidos a los reales y una URL.
// FAKE_MODE: "ok" (espera hasta que lo maten), "nourl" (nunca da URL), "die" (da URL y muere al segundo).
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"
)

func main() {
	if os.Getenv("FAKE_ARGS_FILE") != "" {
		os.WriteFile(os.Getenv("FAKE_ARGS_FILE"), []byte(strings.Join(os.Args[1:], " ")), 0o644)
	}
	log := func(s string) { fmt.Fprintf(os.Stderr, "2026-09-21T12:00:00Z INF %s\n", s) }
	log("Thank you for trying Cloudflare Tunnel.")
	log("Requesting new quick Tunnel on trycloudflare.com...")
	fmt.Fprintln(os.Stderr, "2026-09-21T12:00:00Z WRN Cannot determine default configuration path. No file [config.yml config.yaml]")
	mode := os.Getenv("FAKE_MODE")
	if mode != "nourl" {
		log("+--------------------------------------------------------------------------------------------+")
		log("|  Your quick Tunnel has been created! Visit it at (it may take some time to be reachable):  |")
		log("|  https://amber-cat-dream-yellow.trycloudflare.com                                           |")
		log("+--------------------------------------------------------------------------------------------+")
	}
	if mode == "die" {
		time.Sleep(time.Second)
		os.Exit(1)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
```

- [ ] **Step 2: Test que falla**

`internal/tunnel/run_test.go`:
```go
package tunnel

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var fakeBin string

func TestMain(m *testing.M) {
	dir, _ := os.MkdirTemp("", "fakecf")
	fakeBin = filepath.Join(dir, "cloudflared")
	if runtime.GOOS == "windows" {
		fakeBin += ".exe"
	}
	out, err := exec.Command("go", "build", "-o", fakeBin, "../../test/fixtures/fake-cloudflared").CombinedOutput()
	if err != nil {
		panic(string(out))
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func okVerify(context.Context, string) error { return nil }

func TestParseURL(t *testing.T) {
	cases := map[string]string{
		"2026-09-21T12:00:00Z INF |  https://amber-cat-dream-yellow.trycloudflare.com   |": "https://amber-cat-dream-yellow.trycloudflare.com",
		"INF Requesting new quick Tunnel on trycloudflare.com...":                           "",
		"ERR failed to request quick Tunnel: Post \"https://api.trycloudflare.com/tunnel\"": "",
		"https://evil.com/?x=https://a.trycloudflare.com.evil.com":                          "",
		"https://a-b-3.trycloudflare.com":                                                   "https://a-b-3.trycloudflare.com",
	}
	for line, want := range cases {
		got, ok := ParseURL(line)
		if got != want || ok != (want != "") {
			t.Errorf("ParseURL(%q) = %q,%v; want %q", line, got, ok, want)
		}
	}
}

func TestStartOKAndStop(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args")
	t.Setenv("FAKE_ARGS_FILE", argsFile)
	t.Setenv("FAKE_MODE", "ok")
	tun, err := Start(context.Background(), Options{Bin: fakeBin, Port: 8080, Timeout: 5 * time.Second, Verify: okVerify})
	if err != nil {
		t.Fatal(err)
	}
	if tun.URL != "https://amber-cat-dream-yellow.trycloudflare.com" {
		t.Fatalf("URL = %q", tun.URL)
	}
	args, _ := os.ReadFile(argsFile)
	want := "tunnel --url http://127.0.0.1:8080 --no-autoupdate --config " + nullConfig() + " --loglevel info"
	if string(args) != want {
		t.Fatalf("args = %q\nwant  %q", args, want)
	}
	tun.Stop()
	select {
	case <-tun.Done():
	case <-time.After(7 * time.Second):
		t.Fatal("Stop no terminó el proceso")
	}
}

func TestStartTimeoutWithoutURL(t *testing.T) {
	t.Setenv("FAKE_MODE", "nourl")
	start := time.Now()
	_, err := Start(context.Background(), Options{Bin: fakeBin, Port: 8080, Timeout: time.Second, Verify: okVerify})
	if err == nil || time.Since(start) > 4*time.Second {
		t.Fatalf("err=%v en %v", err, time.Since(start))
	}
	if !strings.Contains(err.Error(), "Requesting new quick Tunnel") {
		t.Fatalf("el error no trae el log para 'Ver detalle': %v", err)
	}
}

func TestStartVerifyFails(t *testing.T) {
	t.Setenv("FAKE_MODE", "ok")
	bad := func(context.Context, string) error { return errors.New("todavía no responde") }
	if _, err := Start(context.Background(), Options{Bin: fakeBin, Port: 8080, Timeout: time.Second, Verify: bad}); err == nil {
		t.Fatal("mostró una URL que no verificó")
	}
}

func TestChildDiesOnItsOwn(t *testing.T) {
	t.Setenv("FAKE_MODE", "die")
	tun, err := Start(context.Background(), Options{Bin: fakeBin, Port: 8080, Timeout: 5 * time.Second, Verify: okVerify})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-tun.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("no se enteró de que el proceso murió")
	}
}

func TestIsolatedHome(t *testing.T) {
	// Un ~/.cloudflared/config.yaml del usuario rompe los quick tunnels: el hijo corre con HOME vacío.
	env := childEnv("/tmp/pasame-cf-home")
	var home, profile bool
	for _, e := range env {
		home = home || e == "HOME=/tmp/pasame-cf-home"
		profile = profile || e == "USERPROFILE=/tmp/pasame-cf-home"
	}
	if !home || !profile {
		t.Fatalf("env sin HOME/USERPROFILE aislados")
	}
}
```

- [ ] **Step 3: Verificar que falla**

Run: `go test ./internal/tunnel/ -run 'Parse|Start|Child|Isolated'`
Expected: FAIL — `undefined: ParseURL`, `Start`, …

- [ ] **Step 4: Implementar**

`internal/tunnel/null_unix.go`:
```go
//go:build !windows

package tunnel

import (
	"os"
	"os/exec"
	"time"
)

func nullConfig() string { return "/dev/null" }

func interrupt(cmd *exec.Cmd) { cmd.Process.Signal(os.Interrupt) }

func kill(cmd *exec.Cmd) { cmd.Process.Kill() }

var stopGrace = 5 * time.Second
```

`internal/tunnel/null_windows.go`:
```go
//go:build windows

package tunnel

import (
	"os/exec"
	"strconv"
	"time"
)

func nullConfig() string { return "NUL" }

// En Windows no hay SIGINT para otro proceso: se usa taskkill con el árbol.
func interrupt(cmd *exec.Cmd) {
	exec.Command("taskkill", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
}

func kill(cmd *exec.Cmd) {
	exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
}

var stopGrace = 5 * time.Second
```

`internal/tunnel/run.go`:
```go
package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var urlRe = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com\b`)

// ParseURL busca la URL del quick tunnel en una línea de log. "api." es el endpoint de Cloudflare, no el túnel.
func ParseURL(line string) (string, bool) {
	for _, u := range urlRe.FindAllString(line, -1) {
		if end := strings.Index(line, u) + len(u); end < len(line) && line[end] == '.' {
			continue // "…trycloudflare.com.evil.com"
		}
		if u != "https://api.trycloudflare.com" {
			return u, true
		}
	}
	return "", false
}

type Options struct {
	Bin     string
	Port    int
	Timeout time.Duration
	Verify  func(ctx context.Context, url string) error
}

type Tunnel struct {
	URL  string
	cmd  *exec.Cmd
	done chan struct{}
	home string

	mu  sync.Mutex
	log []string
}

func childEnv(home string) []string {
	return append(os.Environ(), "HOME="+home, "USERPROFILE="+home)
}

func Start(ctx context.Context, o Options) (*Tunnel, error) {
	if o.Timeout == 0 {
		o.Timeout = 30 * time.Second
	}
	if o.Verify == nil {
		o.Verify = verifyHealthz
	}
	home, err := os.MkdirTemp("", "pasame-cf-")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(o.Bin, "tunnel", "--url", "http://127.0.0.1:"+strconv.Itoa(o.Port),
		"--no-autoupdate", "--config", nullConfig(), "--loglevel", "info")
	cmd.Env = childEnv(home)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		os.RemoveAll(home)
		return nil, err
	}
	t := &Tunnel{cmd: cmd, done: make(chan struct{}), home: home}
	found := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			line := sc.Text()
			t.addLog(line)
			if u, ok := ParseURL(line); ok {
				select {
				case found <- u:
				default:
				}
			}
		}
	}()
	go func() {
		cmd.Wait()
		os.RemoveAll(home)
		close(t.done)
	}()

	deadline, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()
	select {
	case t.URL = <-found:
	case <-t.done:
		return nil, fmt.Errorf("cloudflared terminó sin dar una dirección:\n%s", t.LastLog())
	case <-deadline.Done():
		t.Stop()
		return nil, fmt.Errorf("cloudflared no dio una dirección a tiempo:\n%s", t.LastLog())
	}
	// Los quick tunnels tardan unos segundos en propagarse: reintentar hasta el plazo.
	for {
		err := o.Verify(deadline, t.URL)
		if err == nil {
			return t, nil
		}
		select {
		case <-deadline.Done():
			t.Stop()
			return nil, fmt.Errorf("la dirección %s no respondió: %v", t.URL, err)
		case <-t.done:
			return nil, fmt.Errorf("cloudflared se cerró:\n%s", t.LastLog())
		case <-time.After(time.Second):
		}
	}
}

func verifyHealthz(ctx context.Context, url string) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", url+"/healthz", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("respondió %s", resp.Status)
	}
	return nil
}

func (t *Tunnel) Done() <-chan struct{} { return t.done }

func (t *Tunnel) Stop() {
	select {
	case <-t.done:
		return
	default:
	}
	interrupt(t.cmd)
	select {
	case <-t.done:
	case <-time.After(stopGrace):
		kill(t.cmd)
		<-t.done
	}
}

func (t *Tunnel) addLog(line string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.log = append(t.log, line)
	if len(t.log) > 30 {
		t.log = t.log[len(t.log)-30:]
	}
}

func (t *Tunnel) LastLog() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return strings.Join(t.log, "\n")
}
```
El `HOME` aislado resuelve de raíz el problema de `~/.cloudflared/config.yaml` (spec §4.6) sin depender de que `--config /dev/null` lo anule: se hacen las dos cosas. Igual queda como primer chequeo de `test/manual/tunel.md` (Tarea 10).

- [ ] **Step 5: Verificar que pasa**

Run: `go test -race ./internal/tunnel/ && make cross`
Expected: `ok` y los 7 targets compilan.

- [ ] **Step 6: Prueba contra Cloudflare real (una vez, manual)**

`internal/tunnel/real_test.go` temporal (no commitear):
```go
package tunnel

import (
	"context"
	"net/http"
	"testing"
)

func TestRealTunnel(t *testing.T) {
	go http.ListenAndServe("127.0.0.1:18081", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	bin, err := Ensure(context.Background(), t.TempDir(), func(d, tot int64) {})
	if err != nil {
		t.Fatal(err)
	}
	tun, err := Start(context.Background(), Options{Bin: bin, Port: 18081})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("URL:", tun.URL)
	tun.Stop()
}
```
Run: `go test ./internal/tunnel/ -run RealTunnel -v -timeout 3m`
Expected: descarga (~20–55 MB según OS), imprime `URL: https://….trycloudflare.com` en menos de 30 s. Borrar el archivo. Si falla por `config.yaml` o por el formato del log, ajustar `ParseURL` con la línea real y agregarla a `TestParseURL`.

- [ ] **Step 7: Commit**

```bash
git add internal/tunnel test/fixtures
git commit -m "feat(tunnel): ejecutar cloudflared aislado, leer y verificar la URL, apagado ordenado

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: "Compartir por internet" de punta a punta — proveedor, modo estricto, API y UI

Spec: §4.5 (primaria = túnel si está), §4.5.2 (proveedor túnel), §4.6 (pasos 4–6), §5, §6.2 (hint de 45 s con la salida al túnel), §6.3 (pantalla completa del túnel).

**Files:**
- Create: `internal/addr/tunnel.go`, `internal/core/tunnel.go`
- Modify: `internal/core/core.go` (campos, `sessionPath`, `PINKey`, `State.Tunnel`), `internal/core/state.go`, `internal/control/server.go` y `api.go` (`POST /api/tunnel`), `internal/control/ui_test.go` (excepción de voz), `web/control/app.js`, `web/control/app.css`, `cmd/pasame/main.go` (`PINKey` en `share.Deps`, apagar el túnel al salir)
- Test: `internal/addr/tunnel_test.go`, `internal/core/tunnel_test.go`

**Interfaces:**
- Consumes: `tunnel.Ensure`, `tunnel.Start`, `tunnel.Available` (Tareas 2–3); `share.Deps.PINKey` (Tarea 1).
- Produces:
  ```go
  // addr
  type TunnelProvider struct{ URL string }             // "https://x.trycloudflare.com"
  func (p *TunnelProvider) Kind() string                // "tunnel"
  func (p *TunnelProvider) Addresses(ctx context.Context, sessionPath string) ([]Address, error)
  // core
  type TunnelState struct {
      Status   string `json:"status"`   // "off" | "downloading" | "connecting" | "on" | "error" | "unavailable"
      Progress int    `json:"progress"` // 0–100 durante "downloading"
      Host     string `json:"host"`
      PIN      string `json:"pin"`      // "4 8 1 3"
      Detail   string `json:"detail"`   // para "Ver detalle"
      Dropped  bool   `json:"dropped"`  // se cortó sin que la persona lo apagara
  }
  type TunnelStarter func(ctx context.Context, progress func(pct int)) (url string, done <-chan struct{}, stop func(), err error)
  // Options gana: Tunnel TunnelStarter; TunnelAvailable func() bool (por defecto tunnel.Available)
  func (c *Core) EnableTunnel()
  func (c *Core) DisableTunnel()
  func (c *Core) PINKey() []byte
  // State gana: Tunnel TunnelState `json:"tunnel"`
  ```
  Regla de `sessionPath`: en modo estricto **todas** las direcciones llevan `?pin=<PIN>` (también la LAN), porque el modo estricto pide PIN en toda ruta.

- [ ] **Step 1: Test del proveedor (falla)**

`internal/addr/tunnel_test.go`:
```go
package addr

import (
	"context"
	"testing"
)

func TestTunnelProvider(t *testing.T) {
	p := &TunnelProvider{URL: "https://amber-cat.trycloudflare.com"}
	as, err := p.Addresses(context.Background(), "/s/k3x9m?pin=4813")
	if err != nil || len(as) != 1 {
		t.Fatal(err)
	}
	a := as[0]
	if a.Kind != "tunnel" || a.URL != "https://amber-cat.trycloudflare.com/s/k3x9m?pin=4813" ||
		a.Display != "amber-cat.trycloudflare.com" || a.Label != "Por internet" {
		t.Fatalf("%+v", a)
	}
}
```

- [ ] **Step 2: Implementar `internal/addr/tunnel.go`**

```go
package addr

import (
	"context"
	"strings"
)

// TunnelProvider publica la URL pública del túnel. No sabe nada de cloudflared: core le pasa la URL.
type TunnelProvider struct{ URL string }

func (p *TunnelProvider) Kind() string { return "tunnel" }

func (p *TunnelProvider) Addresses(_ context.Context, sessionPath string) ([]Address, error) {
	return []Address{{
		Kind: "tunnel", URL: p.URL + sessionPath,
		Display: strings.TrimPrefix(p.URL, "https://"), Label: "Por internet",
	}}, nil
}
```
Run: `go test -race ./internal/addr/`
Expected: `ok`.

- [ ] **Step 3: Tests de core (fallan)**

`internal/core/tunnel_test.go`:
```go
package core

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

type fakeTunnel struct {
	err     error
	done    chan struct{}
	stopped atomic.Bool
}

func (f *fakeTunnel) start(_ context.Context, progress func(int)) (string, <-chan struct{}, func(), error) {
	progress(50)
	if f.err != nil {
		return "", nil, nil, f.err
	}
	f.done = make(chan struct{})
	return "https://amber-cat.trycloudflare.com", f.done, func() {
		if f.stopped.CompareAndSwap(false, true) {
			close(f.done)
		}
	}, nil
}

func withTunnel(t *testing.T, ft *fakeTunnel, available bool) *env {
	e := newEnv(t)
	e.c.o.Tunnel = ft.start
	e.c.o.TunnelAvailable = func() bool { return available }
	return e
}

func TestEnableTunnel(t *testing.T) {
	ft := &fakeTunnel{}
	e := withTunnel(t, ft, true)
	e.c.Share([]string{e.file(t, "a", 1)})
	keyBefore := append([]byte(nil), e.c.PINKey()...)
	e.c.EnableTunnel()
	waitFor(t, func() bool { return e.c.State().Tunnel.Status == "on" })
	s := e.c.State()
	pin := e.c.Current().PIN
	if !s.Strict || !e.c.Strict() || s.Tunnel.Host != "amber-cat.trycloudflare.com" {
		t.Fatalf("%+v", s.Tunnel)
	}
	if s.Tunnel.PIN != strings.Join(strings.Split(pin, ""), " ") {
		t.Fatalf("PIN mostrado %q", s.Tunnel.PIN)
	}
	if s.Addresses[0].Kind != "tunnel" || !strings.HasSuffix(s.Addresses[0].URL, "?pin="+pin) {
		t.Fatalf("primaria %+v", s.Addresses[0])
	}
	if !strings.HasSuffix(s.Addresses[1].URL, "?pin="+pin) {
		t.Fatal("la dirección LAN también necesita el PIN en modo estricto")
	}
	if bytes.Equal(keyBefore, e.c.PINKey()) {
		t.Fatal("la clave no rotó al prender")
	}
}

func TestDisableTunnel(t *testing.T) {
	ft := &fakeTunnel{}
	e := withTunnel(t, ft, true)
	e.c.Share([]string{e.file(t, "a", 1)})
	e.c.EnableTunnel()
	waitFor(t, func() bool { return e.c.State().Tunnel.Status == "on" })
	key := append([]byte(nil), e.c.PINKey()...)
	e.c.DisableTunnel()
	s := e.c.State()
	if s.Tunnel.Status != "off" || s.Strict || !ft.stopped.Load() || s.Addresses[0].Kind != "lan" {
		t.Fatalf("%+v", s)
	}
	if strings.Contains(s.Addresses[0].URL, "?pin=") {
		t.Fatal("quedó el PIN en la URL LAN")
	}
	if bytes.Equal(key, e.c.PINKey()) {
		t.Fatal("la clave no rotó al apagar")
	}
}

func TestTunnelDropsOnItsOwn(t *testing.T) {
	ft := &fakeTunnel{}
	e := withTunnel(t, ft, true)
	e.c.EnableTunnel()
	waitFor(t, func() bool { return e.c.State().Tunnel.Status == "on" })
	close(ft.done) // Cloudflare cortó
	ft.stopped.Store(true)
	waitFor(t, func() bool { s := e.c.State(); return s.Tunnel.Status == "off" && s.Tunnel.Dropped && !s.Strict })
}

func TestTunnelError(t *testing.T) {
	e := withTunnel(t, &fakeTunnel{err: errors.New("línea 1\nlínea 2")}, true)
	e.c.EnableTunnel()
	waitFor(t, func() bool { return e.c.State().Tunnel.Status == "error" })
	if s := e.c.State(); s.Strict || !strings.Contains(s.Tunnel.Detail, "línea 2") {
		t.Fatalf("%+v", s.Tunnel)
	}
}

func TestTunnelUnavailable(t *testing.T) {
	e := withTunnel(t, &fakeTunnel{}, false)
	if s := e.c.State(); s.Tunnel.Status != "unavailable" {
		t.Fatalf("%+v", s.Tunnel)
	}
	e.c.EnableTunnel()
	if s := e.c.State(); s.Tunnel.Status != "unavailable" || s.Strict {
		t.Fatalf("%+v", s.Tunnel)
	}
}
```

- [ ] **Step 4: Verificar que falla**

Run: `go test ./internal/core/ -run Tunnel`
Expected: FAIL — `e.c.o.Tunnel undefined`, …

- [ ] **Step 5: Implementar `internal/core/tunnel.go`**

```go
package core

import (
	"context"
	"crypto/rand"
	"strings"

	"github.com/KonixDev/pasame/internal/addr"
	"github.com/KonixDev/pasame/internal/tunnel"
)

type TunnelState struct {
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Host     string `json:"host"`
	PIN      string `json:"pin"`
	Detail   string `json:"detail"`
	Dropped  bool   `json:"dropped"`
}

type TunnelStarter func(ctx context.Context, progress func(pct int)) (string, <-chan struct{}, func(), error)

// defaultTunnel baja cloudflared si hace falta y lo arranca apuntando al listener share.
func defaultTunnel(configDir string, sharePort int) TunnelStarter {
	return func(ctx context.Context, progress func(int)) (string, <-chan struct{}, func(), error) {
		bin, err := tunnel.Ensure(ctx, configDir, func(done, total int64) {
			if total > 0 {
				progress(int(done * 100 / total))
			}
		})
		if err != nil {
			return "", nil, nil, err
		}
		progress(100)
		t, err := tunnel.Start(ctx, tunnel.Options{Bin: bin, Port: sharePort})
		if err != nil {
			return "", nil, nil, err
		}
		return t.URL, t.Done(), t.Stop, nil
	}
}

func newPINKey() []byte {
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		panic(err)
	}
	return k
}

func (c *Core) PINKey() []byte { c.mu.Lock(); defer c.mu.Unlock(); return c.pinKey }

func (c *Core) EnableTunnel() {
	c.mu.Lock()
	if !c.o.TunnelAvailable() || c.tun.Status == "downloading" || c.tun.Status == "connecting" || c.tun.Status == "on" {
		c.mu.Unlock()
		return
	}
	c.tun = TunnelState{Status: "downloading"}
	c.mu.Unlock()
	c.notify()

	go func() {
		url, done, stop, err := c.o.Tunnel(context.Background(), func(pct int) {
			c.mu.Lock()
			c.tun.Progress = pct
			if pct >= 100 {
				c.tun.Status = "connecting"
			}
			c.mu.Unlock()
			c.notify()
		})
		if err != nil {
			c.mu.Lock()
			c.tun = TunnelState{Status: "error", Detail: err.Error()}
			c.mu.Unlock()
			c.notify()
			return
		}
		c.mu.Lock()
		c.tun = TunnelState{Status: "on", Host: strings.TrimPrefix(url, "https://")}
		c.tunStop, c.strict, c.pinKey = stop, true, newPINKey()
		c.mu.Unlock()
		c.o.Book.Register(&addr.TunnelProvider{URL: url})
		c.notify()

		<-done
		c.mu.Lock()
		dropped := c.tun.Status == "on" // si lo apagó la persona, DisableTunnel ya cambió el estado
		c.mu.Unlock()
		if dropped {
			c.turnOff(true)
		}
	}()
}

func (c *Core) DisableTunnel() {
	c.mu.Lock()
	stop := c.tunStop
	c.mu.Unlock()
	c.turnOff(false)
	if stop != nil {
		stop()
	}
}

func (c *Core) turnOff(dropped bool) {
	c.o.Book.Unregister("tunnel")
	c.mu.Lock()
	c.tun = TunnelState{Status: "off", Dropped: dropped}
	c.tunStop, c.strict, c.pinKey = nil, false, newPINKey()
	c.mu.Unlock()
	c.notify()
}

// tunnelView completa lo que depende de la sesión (el PIN, con espacios para leerlo en voz alta).
func (c *Core) tunnelView(pin string) TunnelState {
	t := c.tun
	if !c.o.TunnelAvailable() {
		t.Status = "unavailable"
	}
	if t.Status == "on" && pin != "" {
		t.PIN = strings.Join(strings.Split(pin, ""), " ")
	}
	return t
}
```
`turnOff` marca `Status: "off"` **antes** de llamar a `stop()`: así, cuando el `<-done` del goroutine se desbloquea, ve que no fue un corte y no pisa el estado con `Dropped`.

- [ ] **Step 6: Cambios en `core.go`, `state.go` y `main.go`**

`internal/core/state.go`: agregar a `State` el campo `Tunnel TunnelState `json:"tunnel"``.

`internal/core/core.go`:
- En `Options`: `Tunnel TunnelStarter` y `TunnelAvailable func() bool`.
- En `Core`: `pinKey []byte`, `tun TunnelState`, `tunStop func()`.
- En `New`, antes del `return`:
```go
	if c.o.Tunnel == nil {
		c.o.Tunnel = defaultTunnel(o.ConfigDir, o.SharePort)
	}
	if c.o.TunnelAvailable == nil {
		c.o.TunnelAvailable = tunnel.Available
	}
	c.pinKey = newPINKey()
	c.tun = TunnelState{Status: "off"}
```
(importar `github.com/KonixDev/pasame/internal/tunnel`; como `c.o` es copia de `o`, asignar sobre `c.o`.)
- `sessionPath`:
```go
func (c *Core) sessionPath(s *session.Session) string {
	if c.Strict() {
		return s.Path() + "?pin=" + s.PIN
	}
	return s.Path()
}
```
- En `State()`, dentro del bloque bloqueado, después de armar `st`: `pin := ""; if sess != nil { pin = sess.PIN }; st.Tunnel = c.tunnelView(pin)`.

`cmd/pasame/main.go`:
- En `share.Deps`: `PINKey: c.PINKey,`.
- Antes de `ctlSrv.Shutdown`: `c.DisableTunnel()`.

- [ ] **Step 7: API `POST /api/tunnel`**

En `internal/control/server.go`, registrar `s.mux.HandleFunc("POST /api/tunnel", s.auth(s.tunnel))`. En `api.go`:
```go
func (s *server) tunnel(w http.ResponseWriter, r *http.Request) {
	var b struct{ On bool }
	if !decode(w, r, &b) {
		return
	}
	if b.On {
		s.c.EnableTunnel()
	} else {
		s.c.DisableTunnel()
	}
	w.WriteHeader(http.StatusAccepted)
}
```

- [ ] **Step 8: UI del túnel en `web/control/app.js`**

Agregar al objeto `T`:
```js
    far: '¿Están lejos o en otra red?',
    tunnelOn: 'Compartir por internet',
    tunnelOff: 'Volver a compartir solo por WiFi',
    preparing: 'Preparando… (la primera vez baja un componente de 40 MB, tarda un minuto)',
    connecting: 'Conectando…',
    askKey: 'y cuando te pida la clave:',
    privacy: 'Los archivos pasan por los servidores de Cloudflare (servicio gratuito, puede no estar disponible).',
    strictNote: 'Ahora todos necesitan la clave, también en tu WiFi. El código QR ya la lleva.',
    tunnelError: 'No se pudo conectar por internet. Volvé a intentar en un momento.',
    detail: 'Ver detalle',
    dropped: 'Se cortó la conexión por internet. Podés volver a activarla.',
    noTunnel: 'Compartir por internet no está disponible en esta computadora.',
```
Reemplazar `T.cantEnterPlan2 = '';` por:
```js
  T.cantEnterPlan2 = '<p>Si están en un bar, hotel o aeropuerto, o en redes distintas: <button class="secondary" data-act="tunnel-on">' + T.tunnelOn + '</button></p>';
```
Agregar la función y llamarla desde `renderSharing` reemplazando `h += '<div id="plan2-slot"></div></div></div>';` por `h += tunnelBlock(s) + '</div></div>';`:
```js
  function tunnelBlock(s) {
    var tn = s.tunnel || {}, h = '<div style="margin-top:24px">';
    switch (tn.status) {
      case 'on':
        h += '<p>' + esc(T.askKey) + '</p><div class="addr pin">' + esc(tn.pin) + '</div>';
        h += '<p class="small">' + esc(T.privacy) + '</p><p class="small">' + esc(T.strictNote) + '</p>';
        h += '<button class="secondary" data-act="tunnel-off">' + esc(T.tunnelOff) + '</button>';
        break;
      case 'downloading':
        h += '<p>' + esc(T.preparing) + '</p><div class="bar"><div style="width:' + (tn.progress | 0) + '%"></div></div>';
        break;
      case 'connecting':
        h += '<p>' + esc(T.connecting) + '</p>';
        break;
      case 'unavailable':
        h += '<p class="small">' + esc(T.noTunnel) + '</p>';
        break;
      case 'error':
        h += '<div class="notice">' + esc(T.tunnelError) + ' <button class="link" data-act="detail">' + esc(T.detail) +
          '</button><pre id="detail" hidden>' + esc(tn.detail) + '</pre></div>';
        /* falls through */
      default:
        if (tn.dropped) h += '<div class="notice">' + esc(T.dropped) + '</div>';
        h += '<p>' + esc(T.far) + ' <button class="secondary" data-act="tunnel-on">' + esc(T.tunnelOn) + '</button></p>';
    }
    return h + '</div>';
  }
```
En el `switch` de clicks, agregar:
```js
      case 'tunnel-on': return api('/api/tunnel', { on: true });
      case 'tunnel-off': return api('/api/tunnel', { on: false });
      case 'detail': var d = document.getElementById('detail'); d.hidden = !d.hidden; return;
```
Y como `render()` cada 5 s cerraría el detalle abierto, sumar `|| (f && f.getAttribute('data-act') === 'detail')` a la condición de foco al principio de `render()`.

`web/control/app.css`, agregar:
```css
.addr.pin{letter-spacing:.2em}
.bar{height:14px;background:#eee;border-radius:7px;overflow:hidden;margin:8px 0;max-width:420px}
.bar div{height:100%;background:#0a58ca}
pre{white-space:pre-wrap;font-size:13px;max-height:200px;overflow:auto}
```

`internal/control/ui_test.go`, en `TestUIVoice`, debajo del `ReplaceAll` de Windows Defender:
```go
	// Aviso de privacidad del túnel: tiene que decir exactamente por dónde pasan los archivos.
	texts = strings.ReplaceAll(texts, "los servidores de cloudflare", "")
```

- [ ] **Step 9: Verificar todo**

Run: `go test -race ./... && make build`
Expected: todo `ok`.

- [ ] **Step 10: Humo manual con Cloudflare real**

1. `./pasame` → elegir un archivo → **Compartir por internet**.
2. La primera vez: "Preparando…" con barra; después "Conectando…"; en < 30 s el QR cambia y aparece `xxx.trycloudflare.com` + la clave en grande.
3. Con el celular **en datos móviles (WiFi apagado)**: escanear → baja sin pedir clave (va en el QR).
4. Desde otra compu, tipear la dirección → pide la clave → tipearla → entra. 5 claves mal → "Demasiados intentos".
5. Desde el WiFi, tipear `192.168.x.x:8080` → "Para entrar necesitás el código QR o el link de quien comparte."
6. **Volver a compartir solo por WiFi** → QR de LAN; el link de internet deja de andar; en la LAN ya no pide clave.
7. Con la app compartiendo por internet, matar `cloudflared` (`pkill cloudflared`) → "Se cortó la conexión por internet. Podés volver a activarla."

- [ ] **Step 11: Commit**

```bash
git add internal cmd web
git commit -m "feat: compartir por internet con quick tunnel, PIN en el QR y modo estricto

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: `pasame.local` (bonus, detrás de `-tags mdns`)

Spec: §4.5.3 (reglas: tercera línea en letra chica, nunca al QR, silencioso si falla, recortable), §12 (riesgo 11).

**Regla de corte:** si en la prueba manual del Step 5 aparece **cualquier** diálogo de firewall extra o un comportamiento raro de multicast, esta tarea se revierte (`git revert`) y la v1 sale sin `mdns`. No se depura: el costo/beneficio no lo justifica.

**Files:**
- Create: `internal/addr/mdns.go` (`//go:build mdns`), `internal/addr/mdns_stub.go` (`//go:build !mdns`)
- Modify: `cmd/pasame/main.go` (registrar el proveedor si arrancó)
- Test: `internal/addr/mdns_test.go` (`//go:build mdns`)

**Interfaces:**
- Consumes: `LAN.Candidates()` (Plan 1, Tarea 12), `Book.Register`.
- Produces:
  ```go
  var ErrMDNSDisabled = errors.New("addr: compilado sin mdns")
  func NewMDNS(lan *LAN, port int) (*MDNS, error) // stub: ErrMDNSDisabled
  func (m *MDNS) Kind() string                   // "mdns"
  func (m *MDNS) Addresses(ctx context.Context, sessionPath string) ([]Address, error)
  func (m *MDNS) Close()
  ```
  `Addresses` devuelve `Display: "pasame.local:8080"`, `Label: "Nombre fácil (puede no funcionar)"`. Si la IP primaria cambió desde el último anuncio, re-anuncia; si el anuncio falla, devuelve `nil, nil` (no se muestra nada).

- [ ] **Step 1: Dependencia (solo se compila con el tag)**

Run: `go get github.com/hashicorp/mdns@latest`

- [ ] **Step 2: Test que falla**

`internal/addr/mdns_test.go`:
```go
//go:build mdns

package addr

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestMDNSAddressAndReannounce(t *testing.T) {
	ip := "192.168.1.42"
	lan := NewLAN(8080)
	lan.List = func() ([]Iface, error) { return []Iface{{Name: "en0", Up: true, IPs: []net.IP{net.ParseIP(ip)}}}, nil }
	lan.Route = func() net.IP { return nil }
	var announced []string
	m := &MDNS{lan: lan, port: 8080, announce: func(ip string) (func(), error) {
		announced = append(announced, ip)
		return func() {}, nil
	}}
	as, _ := m.Addresses(context.Background(), "/s/abcde")
	if len(as) != 1 || as[0].Display != "pasame.local:8080" || as[0].Kind != "mdns" || as[0].URL != "http://pasame.local:8080/s/abcde" {
		t.Fatalf("%+v", as)
	}
	m.Addresses(context.Background(), "/s/abcde")
	ip = "10.0.0.7"
	m.Addresses(context.Background(), "/s/abcde")
	if len(announced) != 2 || announced[1] != "10.0.0.7" {
		t.Fatalf("anuncios: %v", announced)
	}
}

func TestMDNSFailureIsSilent(t *testing.T) {
	lan := NewLAN(8080)
	lan.List = func() ([]Iface, error) { return []Iface{{Name: "en0", Up: true, IPs: []net.IP{net.ParseIP("192.168.1.42")}}}, nil }
	lan.Route = func() net.IP { return nil }
	m := &MDNS{lan: lan, port: 8080, announce: func(string) (func(), error) { return nil, errors.New("5353 ocupado") }}
	if as, err := m.Addresses(context.Background(), "/s/x"); as != nil || err != nil {
		t.Fatalf("%v %v", as, err)
	}
}
```

- [ ] **Step 3: Verificar que falla**

Run: `go test -tags mdns ./internal/addr/ -run MDNS`
Expected: FAIL — `undefined: MDNS`.

- [ ] **Step 4: Implementar**

`internal/addr/mdns_stub.go`:
```go
//go:build !mdns

package addr

import (
	"context"
	"errors"
)

var ErrMDNSDisabled = errors.New("addr: compilado sin mdns")

type MDNS struct{}

func NewMDNS(*LAN, int) (*MDNS, error)                                { return nil, ErrMDNSDisabled }
func (*MDNS) Kind() string                                            { return "mdns" }
func (*MDNS) Addresses(context.Context, string) ([]Address, error)    { return nil, nil }
func (*MDNS) Close()                                                  {}
```

`internal/addr/mdns.go`:
```go
//go:build mdns

package addr

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/hashicorp/mdns"
)

var ErrMDNSDisabled = errors.New("addr: compilado sin mdns")

const mdnsHost = "pasame.local"

type MDNS struct {
	lan      *LAN
	port     int
	announce func(ip string) (stop func(), error)

	mu   sync.Mutex
	ip   string
	stop func()
	ok   bool
}

func NewMDNS(lan *LAN, port int) (*MDNS, error) {
	return &MDNS{lan: lan, port: port, announce: realAnnounce(port)}, nil
}

func realAnnounce(port int) func(string) (func(), error) {
	return func(ip string) (func(), error) {
		svc, err := mdns.NewMDNSService("Pasame", "_http._tcp", "", mdnsHost+".", port, []net.IP{net.ParseIP(ip)}, nil)
		if err != nil {
			return nil, err
		}
		srv, err := mdns.NewServer(&mdns.Config{Zone: svc, Logger: log.New(io.Discard, "", 0)})
		if err != nil {
			return nil, err
		}
		return func() { srv.Shutdown() }, nil
	}
}

func (m *MDNS) Kind() string { return "mdns" }

func (m *MDNS) Addresses(_ context.Context, sessionPath string) ([]Address, error) {
	cs := m.lan.Candidates()
	if len(cs) == 0 {
		return nil, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if cs[0].IP != m.ip {
		if m.stop != nil {
			m.stop()
		}
		m.ip, m.stop, m.ok = cs[0].IP, nil, false
		if stop, err := m.announce(cs[0].IP); err == nil {
			m.stop, m.ok = stop, true
		} else {
			log.Printf("mdns: %v (no se muestra pasame.local)", err)
		}
	}
	if !m.ok {
		return nil, nil
	}
	hp := mdnsHost + ":" + strconv.Itoa(m.port)
	return []Address{{Kind: "mdns", URL: "http://" + hp + sessionPath, Display: hp, Label: "Nombre fácil (puede no funcionar)"}}, nil
}

func (m *MDNS) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stop != nil {
		m.stop()
	}
}
```
En `cmd/pasame/main.go`, después de `book.Register(lan)`:
```go
	if m, err := addr.NewMDNS(lan, sharePort); err == nil {
		book.Register(m)
		defer m.Close()
	}
```
La UI ya muestra `kind == "mdns"` como "o probá: pasame.local:8080" (Plan 1, Tarea 18) y el `Book` nunca lo pone primero porque LAN siempre está registrado.

- [ ] **Step 5: Verificar y probar a mano**

Run: `go test -race ./internal/addr/ && go test -race -tags mdns ./internal/addr/ && go build -tags mdns -o pasame ./cmd/pasame && ./pasame`
Expected: tests `ok`. En la UI aparece "o probá: pasame.local:8080". Desde el iPhone (Safari) y otra Mac, `http://pasame.local:8080` abre el envío. Anotar en la matriz: Windows 11, Android 12+ y Ubuntu. **Aplicar la regla de corte** de arriba si aparece un diálogo extra.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/addr cmd/pasame
git commit -m "feat(addr): pasame.local por mDNS detrás del build tag mdns

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Empaquetado macOS — `.app` droplet, `Info.plist` e icono

Spec: §3.2 (consola parásita: `LSUIElement`; arrastrar sobre el ícono: droplet AppleScript recortable), §3.3 (universal), §10 (`.dmg`).

**Files:**
- Create: `packaging/macos/Info.plist`, `packaging/macos/droplet.applescript`, `packaging/macos/build-app.sh`, `packaging/icon/pasame.svg`, `packaging/icon/render.sh`
- Generated (commiteados): `packaging/macos/icon.icns`, `packaging/windows/icon.ico`, `packaging/linux/icon.png`

**Interfaces:**
- Consumes: binario universal `dist/pasame_darwin_all/pasame` (lo genera GoReleaser en la Tarea 8; para probar local se arma con `lipo`).
- Produces: `build-app.sh <binario-universal> <versión> <salida-dir>` → `<salida-dir>/Pasame.app`.

- [ ] **Step 1: Icono fuente y render**

`packaging/icon/pasame.svg`: es el isotipo de la marca elegida (Ronda, ver `docs/marca/`). Copiarlo tal cual:
```bash
cp docs/marca/isotipo.svg packaging/icon/pasame.svg
```
Para los tamaños de 16 y 32 px del `.ico`/`.icns`, `render.sh` usa `docs/marca/isotipo-reducido.svg` (por debajo de 24 px el isotipo completo se empasta).
`packaging/icon/render.sh`:
```bash
#!/usr/bin/env bash
# Genera .icns (macOS), .ico (Windows) y .png (Linux) desde el SVG. Requiere: brew install librsvg imagemagick
set -euo pipefail
cd "$(dirname "$0")"
set_dir=pasame.iconset
rm -rf "$set_dir" && mkdir "$set_dir"
for s in 16 32 64 128 256 512 1024; do
  src=pasame.svg; [ $s -lt 24 ] && src=../../docs/marca/isotipo-reducido.svg
  rsvg-convert -w $s -h $s "$src" -o "png-$s.png"
done
for s in 16 32 128 256 512; do
  cp "png-$s.png" "$set_dir/icon_${s}x${s}.png"
  cp "png-$((s*2)).png" "$set_dir/icon_${s}x${s}@2x.png"
done
iconutil -c icns "$set_dir" -o ../macos/icon.icns
magick png-16.png png-32.png png-64.png png-128.png png-256.png ../windows/icon.ico
mkdir -p ../linux && cp png-256.png ../linux/icon.png
rm -rf "$set_dir" png-*.png
```
Run: `brew install librsvg imagemagick && mkdir -p packaging/windows packaging/linux && chmod +x packaging/icon/render.sh && packaging/icon/render.sh`
Expected: existen `packaging/macos/icon.icns`, `packaging/windows/icon.ico`, `packaging/linux/icon.png`. Abrir el `.icns` con Vista Previa y verificar que se ve bien a 16 px.

- [ ] **Step 2: `packaging/macos/Info.plist`**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>Pasame</string>
  <key>CFBundleDisplayName</key><string>Pasame</string>
  <key>CFBundleIdentifier</key><string>ar.com.pasame</string>
  <key>CFBundleVersion</key><string>__VERSION__</string>
  <key>CFBundleShortVersionString</key><string>__VERSION__</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleExecutable</key><string>droplet</string>
  <key>CFBundleIconFile</key><string>icon</string>
  <key>LSMinimumSystemVersion</key><string>12.0</string>
  <key>LSUIElement</key><true/>
  <key>NSHighResolutionCapable</key><true/>
  <key>CFBundleDocumentTypes</key>
  <array>
    <dict>
      <key>CFBundleTypeName</key><string>Cualquier archivo</string>
      <key>CFBundleTypeRole</key><string>Viewer</string>
      <key>LSItemContentTypes</key><array><string>public.item</string></array>
    </dict>
  </array>
</dict>
</plist>
```
`CFBundleIdentifier` usa el dominio `pasame.com.ar` invertido. `__VERSION__` lo reemplaza `build-app.sh`. `CFBundleDocumentTypes` con `public.item` es lo que habilita soltar cualquier archivo o carpeta sobre el ícono.

- [ ] **Step 3: `packaging/macos/droplet.applescript`**

```applescript
-- Doble clic: abre Pasame. Soltar archivos encima: abre Pasame compartiéndolos.
on run
	launchPasame({})
end run

on open droppedItems
	launchPasame(droppedItems)
end open

on launchPasame(items)
	set bin to quoted form of (POSIX path of (path to me) & "Contents/Resources/pasame")
	set args to ""
	repeat with f in items
		set args to args & " " & quoted form of (POSIX path of f)
	end repeat
	do shell script bin & args & " > /dev/null 2>&1 &"
end launchPasame
```

- [ ] **Step 4: `packaging/macos/build-app.sh`**

```bash
#!/usr/bin/env bash
# Uso: build-app.sh <binario-universal> <versión> <dir-salida>
set -euo pipefail
bin="$1"; ver="${2#v}"; out="$3"
here="$(cd "$(dirname "$0")" && pwd)"
app="$out/Pasame.app"
rm -rf "$app"
osacompile -o "$app" "$here/droplet.applescript"
# osacompile genera su propio Info.plist y ejecutable "droplet": se reemplaza el plist por el nuestro.
sed "s/__VERSION__/$ver/g" "$here/Info.plist" > "$app/Contents/Info.plist"
cp "$here/icon.icns" "$app/Contents/Resources/icon.icns"
rm -f "$app/Contents/Resources/droplet.icns"
cp "$bin" "$app/Contents/Resources/pasame"
chmod 755 "$app/Contents/Resources/pasame"
echo "$app"
```

- [ ] **Step 5: Probar local**

```bash
chmod +x packaging/macos/build-app.sh
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /tmp/pasame-arm64 ./cmd/pasame
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /tmp/pasame-amd64 ./cmd/pasame
lipo -create -output /tmp/pasame-universal /tmp/pasame-arm64 /tmp/pasame-amd64
packaging/macos/build-app.sh /tmp/pasame-universal 0.0.0 /tmp
open /tmp/Pasame.app
```
Expected: se abre la pestaña de Pasame; **no** aparece ícono en el Dock ni ventana de Terminal. Arrastrar una foto sobre `/tmp/Pasame.app` en Finder → abre (o delega a la instancia abierta) compartiendo esa foto. Si arrastrar no funciona tras dos intentos de ajuste, aplicar la salida del spec: el droplet se deja solo con `on run` y se anota en la matriz.

- [ ] **Step 6: Commit**

```bash
git add packaging
git commit -m "build(macos): Pasame.app como droplet sin Dock, icono y metadatos

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Empaquetado Windows y Linux

Spec: §3.2 (sin consola en Windows: `-H=windowsgui`; Linux `.desktop` con `Terminal=false`), §3.3, §10 (`.exe` portable con icono y metadatos; `.tar.gz` y `.deb`).

**Files:**
- Create: `packaging/windows/versioninfo.json`, `cmd/pasame/gen.go`
- Create: `packaging/linux/pasame.desktop`
- Generated (commiteados): `cmd/pasame/resource_windows_amd64.syso`, `cmd/pasame/resource_windows_arm64.syso`

**Interfaces:**
- Consumes: `packaging/windows/icon.ico`, `packaging/linux/icon.png` (Tarea 6).
- Produces: recursos `.syso` que `go build` enlaza solos en Windows (icono y metadatos del `.exe`); `.desktop` que usa la Tarea 8 para el `.deb` y el `.tar.gz`.

- [ ] **Step 1: Metadatos del `.exe`**

`packaging/windows/versioninfo.json`:
```json
{
  "FixedFileInfo": { "FileVersion": { "Major": 1 }, "ProductVersion": { "Major": 1 } },
  "StringFileInfo": {
    "CompanyName": "Pasame",
    "FileDescription": "Pasame — pasá archivos a cualquier celular o computadora que esté cerca",
    "InternalName": "pasame",
    "LegalCopyright": "MIT — Martin Coll",
    "OriginalFilename": "Pasame.exe",
    "ProductName": "Pasame"
  },
  "VarFileInfo": { "Translation": { "LangID": "0C0A", "CharsetID": "04B0" } },
  "IconPath": "../../packaging/windows/icon.ico"
}
```
`cmd/pasame/gen.go`:
```go
package main

// Genera el icono y los metadatos del .exe para ambas arquitecturas de Windows.
//go:generate go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.4.1 -64 -o resource_windows_amd64.syso ../../packaging/windows/versioninfo.json
//go:generate go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.4.1 -arm -64 -o resource_windows_arm64.syso ../../packaging/windows/versioninfo.json
```
(`goversioninfo` corre con `go run` y versión fijada: no entra en `go.mod`, la regla de "dos dependencias" se mantiene.)

Run: `go generate ./cmd/pasame/ && ls cmd/pasame/*.syso`
Expected: los dos `.syso`. El sufijo `_windows_<arch>` hace que Go solo los enlace en ese target.

- [ ] **Step 2: Verificar el `.exe`**

Run:
```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -H=windowsgui" -o /tmp/Pasame.exe ./cmd/pasame
file /tmp/Pasame.exe
```
Expected: `PE32+ executable (GUI) x86-64` — la palabra **GUI** confirma que no abre consola. En una PC con Windows 11: el ícono se ve en el Explorador y en Propiedades → Detalles aparece "Pasame".

- [ ] **Step 3: `packaging/linux/pasame.desktop`**

```ini
[Desktop Entry]
Type=Application
Name=Pasame
GenericName=Pasar archivos
Comment=Pasá archivos a cualquier celular o computadora que esté cerca
Exec=pasame %F
Icon=pasame
Terminal=false
Categories=Network;FileTransfer;
MimeType=application/octet-stream;inode/directory;
StartupNotify=false
```
`%F` pasa las rutas soltadas o elegidas con "Abrir con" como argumentos (Plan 1, Tarea 20: `cleanArgs`).

Run: `desktop-file-validate packaging/linux/pasame.desktop` (en Ubuntu: `sudo apt install desktop-file-utils`)
Expected: sin salida (válido). En macOS se puede saltear: CI lo corre en la Tarea 8.

- [ ] **Step 4: Commit**

```bash
git add packaging cmd/pasame/gen.go cmd/pasame/*.syso
git commit -m "build(windows,linux): icono y metadatos del .exe, lanzador .desktop

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Releases — GoReleaser para binarios y paquetes, script propio para el `.dmg` notarizado

Spec: §10 completo (tabla de assets, cosign keyless, notarización), §12 (riesgos 1 y 9: sin UPX), §13 (flags y targets).

**Decisión de implementación (desvío menor del spec §10):** el spec proponía notarizar con `quill` desde Linux. Pero `osacompile` (el droplet de la Tarea 6) solo existe en macOS, así que el job de release corre en `macos-latest` (gratis en repos públicos). Estando en macOS, se usan las herramientas nativas de Apple (`codesign`, `xcrun notarytool`, `xcrun stapler`): son las documentadas por Apple y no suman una herramienta más. GoReleaser queda para lo que hace bien (binarios, `.tar.gz`, `.deb`, checksums, cosign) y el `.app`/`.dmg` lo arma un script aparte, en orden claro: armar → firmar → notarizar → grapar → subir.

**Files:**
- Create: `.goreleaser.yaml`, `packaging/macos/release-mac.sh`, `packaging/macos/make-dmg.sh`, `packaging/macos/entitlements.plist`, `.github/workflows/release.yml`
- Modify: `.github/workflows/ci.yml` (job que valida la config)

**Interfaces:**
- Consumes: Tareas 6–7.
- Produces: al pushear un tag `v*`, una GitHub Release con `Pasame-Windows.exe`, `Pasame-Windows-arm64.exe`, `Pasame-macOS.dmg`, `pasame-linux-{amd64,arm64,armv7}.tar.gz`, `pasame_<ver>_{amd64,arm64,armhf}.deb`, `checksums.txt` (+ `.sig` y `.pem` de cosign) y `latest.json` (`{"version":"1.2.3","macNotarized":true}`), que consume el sitio (Tarea 9).
- Secrets opcionales del repo (sin ellos la release sale igual, marcada pre-release y "beta en Mac", spec §10): `MACOS_CERT_P12` (Developer ID Application en base64), `MACOS_CERT_PASSWORD`, `MACOS_SIGN_IDENTITY` (ej. `Developer ID Application: Martin Coll (ABCDE12345)`), `MACOS_NOTARY_KEY` (API key `.p8` en base64), `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_ISSUER_ID`.

- [ ] **Step 1: Instalar GoReleaser**

Run: `brew install goreleaser && goreleaser --version`
Expected: v2.x.

- [ ] **Step 2: `.goreleaser.yaml`**

```yaml
version: 2
project_name: pasame

before:
  hooks:
    - go mod tidy
    - go test ./...

builds:
  - id: unix
    main: ./cmd/pasame
    binary: pasame
    env: [CGO_ENABLED=0]
    flags: [-trimpath]
    ldflags: ["-s -w -X main.version={{ .Version }}"]
    goos: [linux, darwin]
    goarch: [amd64, arm64, arm]
    goarm: ["7"]
    ignore:
      - { goos: darwin, goarch: arm }
  - id: windows
    main: ./cmd/pasame
    binary: pasame
    env: [CGO_ENABLED=0]
    flags: [-trimpath]
    ldflags: ["-s -w -H=windowsgui -X main.version={{ .Version }}"]
    goos: [windows]
    goarch: [amd64, arm64]

universal_binaries:
  - id: macos
    ids: [unix]
    replace: true   # queda un solo binario darwin (Intel + Apple Silicon)

archives:
  - id: linux
    ids: [unix]
    formats: [tar.gz]
    builds_info: { mode: 0755 }
    name_template: 'pasame-linux-{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}'
    files:
      - src: packaging/linux/pasame.desktop
        strip_parent: true
      - src: packaging/linux/icon.png
        strip_parent: true
      - README.md
      - LICENSE
  - id: windows
    ids: [windows]
    formats: [binary]
    name_template: 'Pasame-Windows{{ if eq .Arch "arm64" }}-arm64{{ end }}'

nfpms:
  - id: deb
    ids: [unix]
    package_name: pasame
    file_name_template: 'pasame_{{ .Version }}_{{ if eq .Arch "arm" }}armhf{{ else }}{{ .Arch }}{{ end }}'
    formats: [deb]
    maintainer: Martin Coll <martin@commercy.com.ar>
    homepage: https://pasame.com.ar
    description: Pasá archivos a cualquier celular o computadora que esté cerca.
    license: MIT
    recommends: [zenity, xdg-utils]
    contents:
      - src: packaging/linux/pasame.desktop
        dst: /usr/share/applications/pasame.desktop
      - src: packaging/linux/icon.png
        dst: /usr/share/icons/hicolor/256x256/apps/pasame.png

checksum:
  name_template: checksums.txt

signs:
  - cmd: cosign
    artifacts: checksum
    signature: '${artifact}.sig'
    certificate: '${artifact}.pem'
    args: [sign-blob, --yes, '--output-signature=${signature}', '--output-certificate=${certificate}', '${artifact}']

release:
  github: { owner: KonixDev, name: pasame }
  prerelease: auto
  draft: true   # release-mac.sh sube el .dmg y latest.json, y recién ahí la publica
```
`archives` no incluye el binario darwin: en macOS la gente descarga el `.dmg` (Step 3). Tampoco sale un `.tar.gz` de darwin, porque un binario suelto en Mac no es para no técnicos (se puede usar `go install`).

Run: `goreleaser check && goreleaser release --snapshot --clean --skip=sign`
Expected: `dist/` con 2 `.exe`, 3 `.tar.gz`, 3 `.deb`, `checksums.txt` y el universal en `dist/macos_darwin_all/pasame`.

- [ ] **Step 3: Script del `.dmg`**

`packaging/macos/entitlements.plist` (el binario Go necesita escuchar en la red; con hardened runtime no hace falta ningún entitlement especial para eso, así que queda vacío a propósito):
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict/></plist>
```

`packaging/macos/make-dmg.sh`:
```bash
#!/usr/bin/env bash
# Uso: make-dmg.sh <Pasame.app> <salida.dmg>
set -euo pipefail
app="$1"; out="$2"
stage="$(mktemp -d)"
cp -R "$app" "$stage/"
ln -s /Applications "$stage/Aplicaciones"
rm -f "$out"
hdiutil create -volname Pasame -srcfolder "$stage" -ov -format UDZO "$out"
rm -rf "$stage"
```

`packaging/macos/release-mac.sh`:
```bash
#!/usr/bin/env bash
# Arma Pasame.app y el .dmg desde el binario universal de GoReleaser; firma y notariza si hay certificado.
# Uso: release-mac.sh <versión>   (ej. 1.2.3). Corre en macOS.
set -euo pipefail
ver="${1#v}"
here="$(cd "$(dirname "$0")" && pwd)"
out=dist/mac
mkdir -p "$out"
"$here/build-app.sh" dist/macos_darwin_all/pasame "$ver" "$out" >/dev/null
app="$out/Pasame.app"
dmg=dist/Pasame-macOS.dmg
notarized=false

if [[ -n "${MACOS_CERT_P12:-}" ]]; then
  kc="$RUNNER_TEMP/pasame.keychain-db"; kcpass="$(uuidgen)"
  security create-keychain -p "$kcpass" "$kc"
  security set-keychain-settings -lut 3600 "$kc"
  security unlock-keychain -p "$kcpass" "$kc"
  echo "$MACOS_CERT_P12" | base64 --decode > "$RUNNER_TEMP/cert.p12"
  security import "$RUNNER_TEMP/cert.p12" -k "$kc" -P "$MACOS_CERT_PASSWORD" -T /usr/bin/codesign
  security set-key-partition-list -S apple-tool:,apple: -s -k "$kcpass" "$kc"
  security list-keychains -d user -s "$kc" login.keychain-db

  sign() { codesign --force --options runtime --timestamp --entitlements "$here/entitlements.plist" --sign "$MACOS_SIGN_IDENTITY" "$1"; }
  # De adentro hacia afuera: el binario Go, el ejecutable del droplet y por último el bundle.
  sign "$app/Contents/Resources/pasame"
  sign "$app/Contents/MacOS/droplet"
  sign "$app"
  codesign --verify --deep --strict --verbose=2 "$app"

  "$here/make-dmg.sh" "$app" "$dmg"
  sign "$dmg"
  echo "$MACOS_NOTARY_KEY" | base64 --decode > "$RUNNER_TEMP/notary.p8"
  xcrun notarytool submit "$dmg" --key "$RUNNER_TEMP/notary.p8" \
    --key-id "$MACOS_NOTARY_KEY_ID" --issuer "$MACOS_NOTARY_ISSUER_ID" --wait
  xcrun stapler staple "$dmg"
  spctl -a -t open --context context:primary-signature -vv "$dmg"   # Expected: accepted, source=Notarized Developer ID
  notarized=true
else
  echo "Sin certificado de Apple: el .dmg sale sin firmar (beta en Mac)."
  "$here/make-dmg.sh" "$app" "$dmg"
fi

printf '{"version":"%s","macNotarized":%s}\n' "$ver" "$notarized" > dist/latest.json
```
`codesign` sobre `Contents/MacOS/droplet`: es el ejecutable que genera `osacompile`; si no se firma aparte, `--deep` lo firma igual, pero hacerlo explícito deja el error en la línea correcta si algo falla.

Run local (sin certificado): `chmod +x packaging/macos/*.sh && RUNNER_TEMP=/tmp packaging/macos/release-mac.sh 0.0.0 && open dist/Pasame-macOS.dmg`
Expected: se monta un disco "Pasame" con `Pasame.app` y el acceso directo `Aplicaciones`; `dist/latest.json` = `{"version":"0.0.0","macNotarized":false}`.

- [ ] **Step 4: `.github/workflows/release.yml`**

```yaml
name: release
on:
  push:
    tags: ['v*']
permissions:
  contents: write
  id-token: write   # cosign keyless (Sigstore)
jobs:
  release:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version: '1.27.x' }
      - uses: sigstore/cosign-installer@v3
      - uses: goreleaser/goreleaser-action@v6
        with: { version: '~> v2', args: release --clean }
        env: { GITHUB_TOKEN: '${{ secrets.GITHUB_TOKEN }}' }
      - name: .dmg (firmado y notarizado si hay certificado)
        run: packaging/macos/release-mac.sh "$GITHUB_REF_NAME"
        env:
          MACOS_CERT_P12: ${{ secrets.MACOS_CERT_P12 }}
          MACOS_CERT_PASSWORD: ${{ secrets.MACOS_CERT_PASSWORD }}
          MACOS_SIGN_IDENTITY: ${{ secrets.MACOS_SIGN_IDENTITY }}
          MACOS_NOTARY_KEY: ${{ secrets.MACOS_NOTARY_KEY }}
          MACOS_NOTARY_KEY_ID: ${{ secrets.MACOS_NOTARY_KEY_ID }}
          MACOS_NOTARY_ISSUER_ID: ${{ secrets.MACOS_NOTARY_ISSUER_ID }}
      - name: Subir .dmg y latest.json, agregarlos a los checksums y publicar
        env: { GH_TOKEN: '${{ secrets.GITHUB_TOKEN }}' }
        run: |
          cd dist
          shasum -a 256 Pasame-macOS.dmg >> checksums.txt
          cosign sign-blob --yes --output-signature checksums.txt.sig --output-certificate checksums.txt.pem checksums.txt
          gh release upload "$GITHUB_REF_NAME" Pasame-macOS.dmg latest.json checksums.txt checksums.txt.sig checksums.txt.pem --clobber
          pre=true; grep -q '"macNotarized":true' latest.json && [[ "$GITHUB_REF_NAME" != *-* ]] && pre=false
          gh release edit "$GITHUB_REF_NAME" --draft=false --prerelease=$pre
```
Una release sin notarizar queda `prerelease` aunque el tag sea `v1.0.0`: el spec (§10) dice que sin notarización Mac es beta y la v1.0 no se considera estable.

- [ ] **Step 5: CI valida la config en cada PR**

En `.github/workflows/ci.yml`, agregar:
```yaml
  release-dry:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version: '1.27.x' }
      - uses: goreleaser/goreleaser-action@v6
        with: { version: '~> v2', args: release --snapshot --clean --skip=sign,publish }
      - run: RUNNER_TEMP=$RUNNER_TEMP packaging/macos/release-mac.sh 0.0.0-dev
```

- [ ] **Step 6: Primera release de prueba**

**Crear el repo y pushear lo hace público: pedirle confirmación a Martín antes de este paso.**
```bash
gh repo create KonixDev/pasame --public --source . --push
git tag v0.1.0-rc1 && git push origin v0.1.0-rc1
gh run watch
```
Expected: en Releases aparece `v0.1.0-rc1` como Pre-release con los 11 archivos del bloque *Produces*. Verificar la firma:
```bash
gh release download v0.1.0-rc1 -p 'checksums.txt*'
cosign verify-blob --certificate checksums.txt.pem --signature checksums.txt.sig \
  --certificate-identity-regexp '^https://github.com/KonixDev/pasame/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com checksums.txt
```
Expected: `Verified OK`.

- [ ] **Step 7: Commit**

```bash
git add .goreleaser.yaml .github packaging/macos
git commit -m "build: releases con GoReleaser y .dmg firmado y notarizado cuando hay certificado

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: Sitio `pasame.com.ar` — descarga según el sistema y "cómo abrirla"

Spec: §3.5 (dominio decidido), §10 ("Sitio de descarga": botón gigante por OS, capturas de SmartScreen y Gatekeeper; **esta página es parte del producto**), §12 (riesgo 1).

**Files:**
- Create: `site/index.html`, `site/como-abrir.html`, `site/site.css`, `site/CNAME`, `site/img/.gitkeep`, `.github/workflows/pages.yml`
- Test: `site/site_test.go` y `site/doc.go` (una línea: `package site`, para que `go build ./...` no se queje de un directorio con solo tests)

**Interfaces:**
- Consumes: la API pública de GitHub `GET https://api.github.com/repos/KonixDev/pasame/releases?per_page=1` (incluye pre-releases; CORS habilitado; 60 pedidos/hora por IP alcanza para un sitio chico) y los nombres de assets de la Tarea 8.
- Produces: `https://pasame.com.ar` y `https://pasame.com.ar/como-abrir.html`.

**Lo que hace Martín a mano (no lo puede hacer un agente):**
1. Registrar `pasame.com.ar` en `nic.ar` (requiere Clave Fiscal de ARCA nivel 2 o superior; ~ARS según tarifa vigente de NIC.ar, anual).
2. Crear una zona gratuita en Cloudflare DNS para `pasame.com.ar` y, en NIC.ar → Delegaciones, poner los dos nameservers que da Cloudflare.
3. En Cloudflare DNS: cuatro registros `A` para `pasame.com.ar` → `185.199.108.153`, `185.199.109.153`, `185.199.110.153`, `185.199.111.153` (GitHub Pages) y un `CNAME` `www` → `konixdev.github.io`. **Proxy desactivado (nube gris)** para que GitHub pueda emitir el certificado.
4. En GitHub → repo → Settings → Pages: *Source* = GitHub Actions; *Custom domain* = `pasame.com.ar`; tildar *Enforce HTTPS* cuando aparezca disponible (minutos a horas).

- [ ] **Step 1: Test que falla (lógica de detección y textos)**

`site/site_test.go`:
```go
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

func TestIndex(t *testing.T) {
	h := read(t, "index.html")
	for _, must := range []string{
		"Descargar Pasame",
		"Pasame-Windows.exe", "Pasame-Windows-arm64.exe", "Pasame-macOS.dmg",
		"pasame-linux-amd64.tar.gz", "pasame-linux-arm64.tar.gz", "pasame-linux-armv7.tar.gz",
		"api.github.com/repos/KonixDev/pasame/releases?per_page=1",
		"https://github.com/KonixDev/pasame/releases", // sin JS: link a todas las descargas
		"como-abrir.html",
		"Desde el celular no hace falta instalar nada",
	} {
		if !strings.Contains(h, must) {
			t.Errorf("index.html sin %q", must)
		}
	}
	if regexp.MustCompile(`(?i)<script[^>]+src=`).MatchString(h) {
		t.Error("el sitio no carga scripts externos")
	}
}

func TestComoAbrir(t *testing.T) {
	h := read(t, "como-abrir.html")
	for _, must := range []string{`id="windows"`, `id="mac"`, "Más información", "Ejecutar de todas formas", "Abrir de todos modos", "Privacidad y seguridad"} {
		if !strings.Contains(h, must) {
			t.Errorf("como-abrir.html sin %q", must)
		}
	}
	for _, img := range regexp.MustCompile(`src="(img/[^"]+)"`).FindAllStringSubmatch(h, -1) {
		if _, err := os.Stat(img[1]); err != nil {
			t.Errorf("falta la captura %s", img[1])
		}
	}
}

func TestCNAME(t *testing.T) {
	if strings.TrimSpace(read(t, "CNAME")) != "pasame.com.ar" {
		t.Fatal("CNAME incorrecto")
	}
}
```
Run: `go test ./site/`
Expected: FAIL — no existen los archivos.

- [ ] **Step 2: `site/site.css`**

```css
/* Marca Ronda (docs/marca). El sitio sí carga la tipografía de marca: no tiene la restricción offline de la app. */
@import url("https://fonts.googleapis.com/css2?family=Bricolage+Grotesque:opsz,wght@12..96,800&family=Atkinson+Hyperlegible:wght@400;700&display=swap");
body{margin:0;font-family:"Atkinson Hyperlegible",-apple-system,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;font-size:19px;line-height:1.5;color:#1B2A1A;background:#FAFAF6}
main{max-width:760px;margin:0 auto;padding:32px 16px 64px}
h1,h2{font-family:"Bricolage Grotesque",-apple-system,"Segoe UI",sans-serif;font-weight:800;letter-spacing:-.02em}
h1{font-size:44px;margin:0 0 8px;color:#2F5D2E}
.lead{font-size:22px;color:#1B2A1A;margin:0 0 32px}
.dl{display:block;text-align:center;padding:22px;font-size:24px;font-weight:700;color:#1B2A1A;background:#A7C957;border-radius:16px;text-decoration:none}
.sub{text-align:center;color:#5E675F;font-size:16px;margin:10px 0 32px}
.box{background:#fff;border:1px solid #E1E4DA;border-radius:16px;padding:18px 20px;margin:20px 0}
.beta{background:#FBF3D5;border:1px solid #F2C94C}
ol li{margin:0 0 14px}
img{max-width:100%;border:1px solid #E1E4DA;border-radius:10px;margin:8px 0}
a{color:#2F5D2E}
```

- [ ] **Step 3: `site/index.html`**

```html
<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Pasame — pasá archivos a cualquier celular o computadora</title>
<meta name="description" content="Pasá fotos, videos y documentos entre dispositivos cercanos. Sin cable, sin cuentas, sin instalar nada en el celular.">
<link rel="stylesheet" href="site.css">
<link rel="icon" href="img/isotipo-reducido.svg" type="image/svg+xml">
</head>
<body><main>
<h1><img src="img/logo.svg" alt="Pasame" width="220" height="50" style="border:0;margin:0"></h1>
<p class="lead">Pasá archivos a cualquier celular o computadora que esté cerca. Sin cable, sin cuentas.</p>

<a class="dl" id="dl" href="https://github.com/KonixDev/pasame/releases">Descargar Pasame</a>
<p class="sub" id="dl-sub">Para Windows, Mac y Linux</p>
<div class="box beta" id="beta" hidden>En Mac, esta versión todavía no está verificada por Apple. <a href="como-abrir.html#mac">Mirá cómo abrirla</a>.</div>
<div class="box" id="phone" hidden><b>Desde el celular no hace falta instalar nada.</b> Pasame se instala en una computadora; el celular solo escanea el código que aparece en la pantalla.</div>

<div class="box">
<ol>
<li>Abrí Pasame en tu computadora y elegí los archivos.</li>
<li>La otra persona apunta la cámara del celular al código que aparece.</li>
<li>Listo: se descargan. Y te puede mandar archivos de vuelta.</li>
</ol>
</div>
<p>¿Windows o Mac te avisan algo al abrirla? <a href="como-abrir.html">Mirá cómo abrirla</a>.</p>
<p class="sub"><a href="https://github.com/KonixDev/pasame/releases">Todas las descargas</a> · <a href="https://github.com/KonixDev/pasame">Código</a> · Gratis y de código abierto</p>

<script>
(function () {
  var ua = navigator.userAgent, asset = null, label = '';
  var arm = /arm|aarch64/i.test(ua) || (navigator.userAgentData && navigator.userAgentData.platform === 'Windows' && /arm/i.test(navigator.platform));
  if (/iPhone|iPad|iPod|Android/i.test(ua)) {
    document.getElementById('phone').hidden = false;
    document.getElementById('dl').hidden = true;
    document.getElementById('dl-sub').hidden = true;
    return;
  }
  if (/Windows/i.test(ua)) { asset = arm ? 'Pasame-Windows-arm64.exe' : 'Pasame-Windows.exe'; label = 'para Windows'; }
  else if (/Mac OS X/i.test(ua)) { asset = 'Pasame-macOS.dmg'; label = 'para Mac'; }
  else if (/Linux/i.test(ua)) {
    asset = /armv7|armv8l/i.test(ua) ? 'pasame-linux-armv7.tar.gz' : /aarch64|arm64/i.test(ua) ? 'pasame-linux-arm64.tar.gz' : 'pasame-linux-amd64.tar.gz';
    label = 'para Linux';
  }
  if (!asset) return;
  var x = new XMLHttpRequest();
  x.open('GET', 'https://api.github.com/repos/KonixDev/pasame/releases?per_page=1');
  x.onload = function () {
    try {
      var rel = JSON.parse(x.responseText)[0];
      for (var i = 0; i < rel.assets.length; i++) {
        if (rel.assets[i].name === asset) {
          document.getElementById('dl').href = rel.assets[i].browser_download_url;
          document.getElementById('dl').textContent = 'Descargar Pasame ' + label;
          document.getElementById('dl-sub').textContent = 'Versión ' + rel.tag_name.replace(/^v/, '');
        }
      }
      if (asset === 'Pasame-macOS.dmg' && rel.prerelease) document.getElementById('beta').hidden = false;
    } catch (e) { /* queda el link a todas las descargas */ }
  };
  x.send();
})();
</script>
</main></body>
</html>
```
Sin JavaScript o si la API falla, el botón sigue funcionando: lleva a la página de releases.

- [ ] **Step 4: `site/como-abrir.html` y capturas**

```html
<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Cómo abrir Pasame</title>
<link rel="stylesheet" href="site.css">
</head>
<body><main>
<p><a href="./">← Pasame</a></p>
<h1>Cómo abrir Pasame</h1>
<p class="lead">La primera vez, tu computadora puede avisarte porque Pasame es nueva y gratuita. Es normal: se hace una sola vez.</p>

<h2 id="windows">En Windows</h2>
<ol>
<li>Hacé doble clic en <b>Pasame-Windows.exe</b> (está en Descargas).</li>
<li>Si aparece “Windows protegió tu PC”, tocá <b>Más información</b>.<br><img src="img/win-smartscreen-1.png" alt="Aviso azul de Windows con el link Más información"></li>
<li>Tocá <b>Ejecutar de todas formas</b>.<br><img src="img/win-smartscreen-2.png" alt="Botón Ejecutar de todas formas"></li>
<li>Si aparece una ventana del “Firewall de Windows Defender”, tocá <b>Permitir acceso</b>. Es para que el celular pueda ver tu computadora.<br><img src="img/win-firewall.png" alt="Ventana del firewall con el botón Permitir acceso"></li>
</ol>

<h2 id="mac">En Mac</h2>
<ol>
<li>Abrí <b>Pasame-macOS.dmg</b> y arrastrá <b>Pasame</b> a <b>Aplicaciones</b>.</li>
<li>Abrí Pasame desde Aplicaciones. Si dice que no se puede abrir, tocá <b>Aceptar</b> (o <b>Listo</b>).<br><img src="img/mac-blocked.png" alt="Aviso de macOS que dice que no se puede abrir"></li>
<li>Abrí <b>Ajustes del Sistema</b> → <b>Privacidad y seguridad</b>, bajá hasta el final y tocá <b>Abrir de todos modos</b>. Ese botón aparece solo durante un rato: si no lo ves, volvé a intentar el paso 2.<br><img src="img/mac-open-anyway.png" alt="Botón Abrir de todos modos en Privacidad y seguridad"></li>
<li>Confirmá con tu contraseña o huella.</li>
</ol>
<p class="box">Cuando Pasame esté verificada por Apple, en Mac no vas a ver ningún aviso.</p>

<h2>En el celular</h2>
<p>No hay que instalar nada. Apuntá la cámara al código que muestra la computadora y tocá el aviso que aparece.</p>
</main></body>
</html>
```
Capturas (a mano, en máquinas reales, con Pasame v0.1.0-rc1 de la Tarea 8): `site/img/win-smartscreen-1.png`, `win-smartscreen-2.png`, `win-firewall.png` en Windows 11; `mac-blocked.png`, `mac-open-anyway.png` en macOS 15. Recortar a la ventana del aviso, 1200 px de ancho máximo, y pasar por `pngquant` o `oxipng`. Copiar además los logos de la marca: `cp docs/marca/logo.svg docs/marca/isotipo-reducido.svg site/img/`. `TestComoAbrir` falla mientras falte alguna captura: es el recordatorio.

- [ ] **Step 5: `site/CNAME` y deploy**

`site/CNAME`:
```
pasame.com.ar
```
`.github/workflows/pages.yml`:
```yaml
name: pages
on:
  push:
    branches: [main]
    paths: ['site/**']
  workflow_dispatch:
permissions:
  contents: read
  pages: write
  id-token: write
concurrency: { group: pages, cancel-in-progress: true }
jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: { name: github-pages, url: '${{ steps.d.outputs.page_url }}' }
    steps:
      - uses: actions/checkout@v4
      - uses: actions/configure-pages@v5
      - uses: actions/upload-pages-artifact@v3
        with: { path: site }
      - id: d
        uses: actions/deploy-pages@v4
```

- [ ] **Step 6: Verificar**

Run: `go test ./site/ && python3 -m http.server -d site 8000` y abrir `http://localhost:8000` en Safari, en Chrome con el modo "dispositivo móvil" de DevTools y con JavaScript desactivado.
Expected: tests `ok`; en la Mac el botón dice "Descargar Pasame para Mac" y apunta al `.dmg`; en "celular" se ve el cartel de que no hace falta instalar nada; sin JS el botón lleva a Releases. Después del deploy y de que Martín complete los 4 pasos de DNS: `curl -sI https://pasame.com.ar | head -1` → `HTTP/2 200`.

- [ ] **Step 7: Commit**

```bash
git add site .github/workflows/pages.yml
git commit -m "feat(site): pasame.com.ar con descarga según el sistema y guía para abrirla

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: README final, checklist del túnel y matriz ampliada

Spec: §5 ("Qué NO se promete"), §9.3 (matriz mínima por release, "sin la matriz completa no se publica una estable"), §10, §12.

**Files:**
- Modify: `README.md`, `test/manual/matriz.md`
- Create: `test/manual/tunel.md`

**Interfaces:**
- Consumes: todo lo anterior.
- Produces: la documentación con la que se decide publicar `v1.0.0`.

- [ ] **Step 1: `test/manual/tunel.md`**

```markdown
# Checklist del túnel real (cloudflared 2026.9.1)

Fecha: ____  Emisor: ____  Receptor: ____

| # | Chequeo | Resultado |
|---|---|---|
| 1 | Con `~/.cloudflared/config.yaml` presente (crear uno de prueba), "Compartir por internet" igual funciona (HOME aislado + `--config`). | |
| 2 | Primera vez: se ve "Preparando…" con barra y termina en menos de 2 min con conexión normal. | |
| 3 | Segunda vez: no vuelve a descargar (pasa directo a "Conectando…"). | |
| 4 | La URL aparece en < 30 s y el QR cambia solo cuando `/healthz` responde. | |
| 5 | Celular en datos móviles escanea el QR y entra sin tipear la clave. | |
| 6 | Tipear la dirección en otra PC pide la clave; 5 errores → "Demasiados intentos"; al minuto se puede de nuevo. | |
| 7 | Descarga de un video de 2 GB por el túnel completa (anotar velocidad). | |
| 8 | Subida de 3 fotos desde el celular por el túnel llega a `Descargas/Pasame`. | |
| 9 | "Volver a compartir solo por WiFi": el link de internet deja de responder; en la LAN ya no pide clave. | |
| 10 | `pkill cloudflared` con el túnel activo → aviso "Se cortó la conexión por internet". | |
| 11 | Cerrar Pasame con el túnel activo no deja un `cloudflared` vivo (`ps aux | grep cloudflared`). | |
| 12 | Windows ARM: el túnel funciona con `cloudflared-windows-386.exe` por emulación. | |
| 13 | Raspberry Pi 32 bits: el túnel funciona con `cloudflared-linux-armhf`. | |
```

- [ ] **Step 2: Ampliar `test/manual/matriz.md`**

Agregar al final:
```markdown
## Instalación y primer arranque (con los assets de la release, no con `make build`)

| Chequeo | Windows 11 x64 | Windows 11 ARM | macOS 15 | macOS 12 | Ubuntu LTS (.deb) | Raspberry Pi OS |
|---|---|---|---|---|---|---|
| El botón de pasame.com.ar baja el archivo correcto | | | | | | |
| Los pasos de como-abrir.html coinciden con lo que se ve | | | | | | |
| Sin notarizar: "Abrir de todos modos" aparece y funciona | n/a | n/a | | | n/a | n/a |
| Notarizado: abre sin ningún aviso | n/a | n/a | | | n/a | n/a |
| El icono se ve bien (Dock/Finder, Explorador, lanzador) | | | | | | |
| Antivirus (Defender) no lo marca | | | n/a | n/a | n/a | n/a |
| `pasame.local` (solo builds con `-tags mdns`) | | | | | | |
```

- [ ] **Step 3: README final**

Reemplazar la sección "Para desarrolladores" del README del Plan 1 y agregar, en este orden, antes de ella:
```markdown
## Descargar

**[pasame.com.ar](https://pasame.com.ar)** — el botón elige la versión para tu computadora.

La primera vez, Windows o Mac pueden avisarte porque la app es nueva: [cómo abrirla](https://pasame.com.ar/como-abrir.html).

## Si están lejos o en otra red

Tocá **Compartir por internet**. Aparece un código nuevo y una clave de 4 números. El código del celular ya trae la clave; desde una computadora, se escribe la dirección y después la clave.

Mientras está activo, todos necesitan la clave, también en tu WiFi. Los archivos pasan por los servidores de Cloudflare (un servicio gratuito que puede no estar disponible).

## Qué no promete

- En tu WiFi, cualquiera conectado a esa misma red que tenga el link puede ver lo que compartís mientras la app está abierta.
- Por internet, los archivos van cifrados hasta Cloudflare, pero Cloudflare podría verlos.
- No uses Pasame en una red en la que no confíes.

## Para desarrolladores

    make test     # unitarios e integración
    make e2e      # Playwright sobre la página del receptor
    make build    # ./pasame
    make cross    # los 7 targets
    go build -tags mdns ./cmd/pasame   # con pasame.local

Releases: pushear un tag `vX.Y.Z` → `.github/workflows/release.yml`. Para que Mac salga notarizado, cargar los secrets `MACOS_*` (ver `docs/superpowers/plans/2026-09-21-pasame-02-tunel-y-distribucion.md`, Tarea 8). Sin ellos, la release sale como pre-release.

Diseño: `docs/superpowers/specs/2026-09-21-share-now-design.md`. Licencia MIT.
```
(El párrafo "Qué no promete" del Plan 1 se reemplaza por esta versión de tres puntos.)

- [ ] **Step 4: Completar las matrices y decidir la v1.0**

Completar `test/manual/matriz.md` y `test/manual/tunel.md` con la release `v0.1.0-rcN` más reciente. Criterio para taggear `v1.0.0` (spec §9.3 y §10):
- Todas las celdas de la matriz mínima en ✓ (Windows 11 + macOS actual + Ubuntu LTS × iPhone actual e iOS−2 + Android actual y Android 9 + una PC).
- El `.dmg` notarizado (secrets de Apple cargados). Sin eso, se puede publicar `v1.0.0` para Windows y Linux, pero sale como pre-release por el workflow y el sitio muestra "beta en Mac".

- [ ] **Step 5: Commit**

```bash
git add README.md test/manual
git commit -m "docs: README final, checklist del túnel y matriz de instalación

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Hito del Plan 2

Plan 2 terminado: `make test`, `make e2e` y el job `release-dry` en verde; `v0.1.0-rc1` publicada con los 11 archivos; `pasame.com.ar` sirve el sitio con HTTPS; `test/manual/tunel.md` completo. La `v1.0.0` estable queda sujeta a que la matriz esté completa y a la notarización de Mac (Tarea 10, Step 4).
