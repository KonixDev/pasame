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
