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
