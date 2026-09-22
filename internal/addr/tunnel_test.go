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
