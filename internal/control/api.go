package control

import (
	"encoding/json"
	"net/http"
)

// bad responde 400 con un código estable; la UI muestra el texto en el idioma de la persona.
func bad(w http.ResponseWriter, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"code": code, "error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v); err != nil {
		bad(w, "bad_request", "pedido inválido")
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
			bad(w, "action_failed", err.Error())
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
		bad(w, "bad_request", "tipo inválido")
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
		bad(w, "no_paths", "no llegó ningún archivo")
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
		bad(w, "bad_name", "Escribí un nombre")
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
		bad(w, "bad_iface", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// lang: auto=true cuando la UI detecta el idioma del navegador; false cuando la persona lo elige.
func (s *server) lang(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Lang string
		Auto bool
	}
	if !decode(w, r, &b) {
		return
	}
	if err := s.c.SetLang(b.Lang, !b.Auto); err != nil {
		bad(w, "bad_lang", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) tunnel(w http.ResponseWriter, r *http.Request) {
	var b struct{ On bool }
	if !decode(w, r, &b) {
		return
	}
	if b.On {
		s.c.EnableTunnel()
	} else {
		s.c.DisableTunnel()
	}
	w.WriteHeader(http.StatusAccepted)
}
