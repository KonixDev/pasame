package share

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

// Listen prueba preferred..preferred+9 y, si están todos ocupados, uno aleatorio.
func Listen(host string, preferred int) (net.Listener, error) {
	for p := preferred; p < preferred+10; p++ {
		if l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, p)); err == nil {
			return l, nil
		}
	}
	return net.Listen("tcp", net.JoinHostPort(host, "0"))
}

// HTTPServer aplica los timeouts del spec: nunca Read/WriteTimeout (matan descargas largas).
func HTTPServer(h http.Handler) *http.Server {
	return &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 120 * time.Second}
}
