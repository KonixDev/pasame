package control

import (
	"encoding/json"
	"net/http"
)

func bad(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v); err != nil {
		bad(w, "pedido inválido")
		return false
	}
	return true
}

func (s *server) simple(f func()) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { f(); w.WriteHeader(http.StatusNoContent) }
}

func (s *server) errOnly(f func() error) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if err := f(); err != nil {
			bad(w, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *server) pick(w http.ResponseWriter, r *http.Request) {
	var b struct{ Kind string }
	if !decode(w, r, &b) {
		return
	}
	if b.Kind != "files" && b.Kind != "folder" {
		bad(w, "tipo inválido")
		return
	}
	s.c.Pick(b.Kind)
	w.WriteHeader(http.StatusAccepted)
}

// add recibe rutas: de una segunda instancia (arrastrar sobre el ícono) o del campo "pegar ruta" en Linux.
func (s *server) add(w http.ResponseWriter, r *http.Request) {
	var b struct{ Paths []string }
	if !decode(w, r, &b) {
		return
	}
	if len(b.Paths) == 0 {
		bad(w, "no llegó ningún archivo")
		return
	}
	s.c.Share(b.Paths)
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) name(w http.ResponseWriter, r *http.Request) {
	var b struct{ Name string }
	if !decode(w, r, &b) {
		return
	}
	if err := s.c.SetName(b.Name); err != nil {
		bad(w, "Escribí un nombre")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) iface(w http.ResponseWriter, r *http.Request) {
	var b struct{ IP string }
	if !decode(w, r, &b) {
		return
	}
	if err := s.c.SetIface(b.IP); err != nil {
		bad(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
