package core

import (
	"context"
	"crypto/rand"
	"strings"

	"github.com/KonixDev/pasame/internal/addr"
	"github.com/KonixDev/pasame/internal/tunnel"
)

type TunnelState struct {
	Status   string `json:"status"`   // "off" | "downloading" | "connecting" | "on" | "error" | "unavailable"
	Progress int    `json:"progress"` // 0–100 durante "downloading"
	Host     string `json:"host"`
	PIN      string `json:"pin"`     // "4 8 1 3": con espacios, para dictarlo
	Detail   string `json:"detail"`  // para "Ver detalle"
	Dropped  bool   `json:"dropped"` // se cortó sin que la persona lo apagara
}

// TunnelStarter prende el túnel. Tiene que cortar (y no dejar procesos) si se cancela ctx.
type TunnelStarter func(ctx context.Context, progress func(pct int)) (url string, done <-chan struct{}, stop func(), err error)

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
	ctx, cancel := context.WithCancel(context.Background())
	c.tun, c.tunCancel = TunnelState{Status: "downloading"}, cancel
	c.mu.Unlock()
	c.notify()

	go func() {
		url, done, stop, err := c.o.Tunnel(ctx, func(pct int) {
			c.mu.Lock()
			if ctx.Err() == nil {
				c.tun.Progress = pct
				if pct >= 100 {
					c.tun.Status = "connecting"
				}
			}
			c.mu.Unlock()
			c.notify()
		})
		c.mu.Lock()
		switch {
		case ctx.Err() != nil: // lo apagaron mientras conectaba: DisableTunnel ya dejó el estado en "off"
			c.mu.Unlock()
			if stop != nil {
				stop()
			}
			return
		case err != nil:
			c.tun = TunnelState{Status: "error", Detail: err.Error()}
			c.mu.Unlock()
			c.notify()
			return
		}
		c.tun = TunnelState{Status: "on", Host: strings.TrimPrefix(url, "https://")}
		c.tunStop, c.strict, c.pinKey = stop, true, newPINKey()
		c.mu.Unlock()
		c.o.Book.Register(&addr.TunnelProvider{URL: url})
		c.notify()

		<-done
		c.mu.Lock()
		dropped := c.tun.Status == "on" && ctx.Err() == nil // si lo apagó la persona, DisableTunnel ya cambió el estado
		c.mu.Unlock()
		if dropped {
			c.turnOff(true) // el proceso ya terminó: no hay nada que parar
		}
	}()
}

func (c *Core) DisableTunnel() {
	c.mu.Lock()
	cancel := c.tunCancel
	c.mu.Unlock()
	if cancel == nil {
		return // no está prendido ni conectándose
	}
	cancel() // primero: un túnel que termina de conectarse ahora ve ctx cancelado y se para solo
	if stop := c.turnOff(false); stop != nil {
		stop()
	}
}

// turnOff marca "off" y devuelve el stop vigente, tomado bajo el mismo lock que lo guardó. Cuando se
// desbloquea <-done, el goroutine ve que no fue un corte y no pisa el estado con Dropped.
func (c *Core) turnOff(dropped bool) (stop func()) {
	c.o.Book.Unregister("tunnel")
	c.mu.Lock()
	stop = c.tunStop
	c.tun = TunnelState{Status: "off", Dropped: dropped}
	c.tunStop, c.tunCancel, c.strict, c.pinKey = nil, nil, false, newPINKey()
	c.mu.Unlock()
	c.notify()
	return stop
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
