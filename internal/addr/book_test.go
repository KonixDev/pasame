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
