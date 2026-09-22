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
	"github.com/KonixDev/pasame/internal/tunnel"
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

	Tunnel          TunnelStarter // por defecto baja y corre cloudflared
	TunnelAvailable func() bool   // por defecto tunnel.Available
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
	detectedLang    string // del navegador de la pestaña de control; no se guarda (es una suposición)
	netChangedUntil time.Time
	lastPrimary     string
	pinKey          []byte // clave HMAC de la cookie del PIN; rota al prender y apagar el túnel
	tun             TunnelState
	tunStop         func()
	tunCancel       context.CancelFunc // corta un túnel que todavía se está conectando

	subsMu sync.Mutex
	subs   map[chan struct{}]bool

	// ciclo de vida (lifecycle.go)
	clients   int
	idleSince time.Time
	done      chan struct{}
	quitOnce  sync.Once
}

func New(o Options) *Core {
	c := &Core{o: o, cfg: o.Config, phase: "idle", subs: map[chan struct{}]bool{}, done: make(chan struct{})}
	// Se calcula una vez: en macOS es un exec de `defaults`, no algo para cada push de SSE.
	// En Windows el aviso solo tiene sentido la primera vez (después el permiso ya se dio o se negó).
	if h := platform.FirewallHint(); h == "mac" || (h == "windows" && o.FirstRun) {
		c.firewallHint = h
	}
	if c.o.Pick == nil {
		c.o.Pick = c.nativePick
	}
	if o.Config.IfaceOverride != "" {
		o.LAN.SetOverride(o.Config.IfaceOverride)
	}
	if c.o.Tunnel == nil {
		c.o.Tunnel = defaultTunnel(o.ConfigDir, o.SharePort)
	}
	if c.o.TunnelAvailable == nil {
		c.o.TunnelAvailable = tunnel.Available
	}
	c.pinKey = newPINKey()
	c.tun = TunnelState{Status: "off"}
	return c
}

var pickTitles = map[i18n.Lang][2]string{
	i18n.ES: {"Elegí los archivos para pasar", "Elegí la carpeta para pasar"},
	i18n.EN: {"Choose the files to send", "Choose the folder to send"},
}

func (c *Core) nativePick(kind string) ([]string, error) {
	t := pickTitles[c.Lang()]
	if kind == "folder" {
		p, err := dialog.PickFolder(t[1])
		return []string{p}, err
	}
	return dialog.PickFiles(t[0])
}

// --- lectura para share.Deps ---

func (c *Core) Current() *session.Session { c.mu.Lock(); defer c.mu.Unlock(); return c.sess }
func (c *Core) Stats() *session.Stats     { c.mu.Lock(); defer c.mu.Unlock(); return c.stats }

// Lang es el idioma de quien comparte: el elegido en el menú (se guarda), si no el detectado del
// navegador de la pestaña de control, si no español. También es el idioma por defecto del receptor.
func (c *Core) Lang() i18n.Lang {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.langLocked()
}

func (c *Core) langLocked() i18n.Lang {
	switch {
	case c.cfg.Lang == "en" || c.cfg.Lang == "es":
		return i18n.Lang(c.cfg.Lang)
	case c.detectedLang == "en":
		return i18n.EN
	}
	return i18n.ES
}

// SetLang cambia el idioma. explicit=true es una elección de la persona (se guarda y gana siempre);
// explicit=false es lo detectado del navegador (no se guarda y no pisa una elección).
func (c *Core) SetLang(lang string, explicit bool) error {
	if lang != "es" && lang != "en" {
		return errors.New("idioma no soportado")
	}
	c.mu.Lock()
	if !explicit {
		c.detectedLang = lang
		c.mu.Unlock()
		c.notify()
		return nil
	}
	c.cfg.Lang = lang
	cfg := c.cfg
	c.mu.Unlock()
	c.notify()
	return platform.SaveConfig(c.o.ConfigDir, cfg)
}

func (c *Core) Strict() bool { c.mu.Lock(); defer c.mu.Unlock(); return c.strict }

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
	c.DisableTunnel() // terminar de compartir también corta internet
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
		Lang:         string(c.langLocked()), LangExplicit: c.cfg.Lang != "",
	}
	pin := ""
	if sess != nil {
		st.SharedAt, pin = c.sharedAt.UnixMilli(), sess.PIN
	}
	st.Tunnel = c.tunnelView(pin)
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
	if st.Strict {
		// En modo estricto "/" no lleva a la sesión: lo que se dicta o se tipea tiene que traer la ruta.
		// El PIN no va: se pide aparte ("y cuando te pida la clave:").
		for i := range st.Addresses {
			st.Addresses[i].Display += sess.Path()
		}
	}
	if len(st.Addresses) > 0 {
		st.QR, _ = qr.SVG(st.Addresses[0].URL)
	}
	return st
}

// sessionPath es lo que cada proveedor agrega a su base. En modo estricto todas las direcciones llevan
// el PIN (también la de la red WiFi), porque el modo estricto lo pide en todas las rutas.
func (c *Core) sessionPath(s *session.Session) string {
	if c.Strict() {
		return s.Path() + "?pin=" + s.PIN
	}
	return s.Path()
}

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

func (c *Core) OpenFolder() error   { return platform.OpenFolder(c.o.Quarantine) }
func (c *Core) OpenFirewall() error { return platform.OpenFirewallSettings() }
