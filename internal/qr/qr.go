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
