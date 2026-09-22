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
