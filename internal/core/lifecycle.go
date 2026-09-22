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
