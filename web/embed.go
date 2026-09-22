// Package web contiene los archivos estáticos embebidos en el binario.
package web

import "embed"

//go:embed share
var Share embed.FS
