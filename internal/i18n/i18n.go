// Package i18n tiene todos los textos que ve el receptor. Sin red, sin exec.
package i18n

import (
	"fmt"
	"strconv"
	"strings"
)

type Lang string

const (
	ES Lang = "es"
	EN Lang = "en"
)

var tables = map[Lang]map[string]string{ES: es, EN: en}

// Pick elige entre es y en según Accept-Language; ante empate o nada conocido, es.
func Pick(accept string) Lang {
	best, bestQ := ES, -1.0
	for _, part := range strings.Split(accept, ",") {
		fields := strings.Split(strings.TrimSpace(part), ";")
		tag := strings.ToLower(strings.SplitN(fields[0], "-", 2)[0])
		q := 1.0
		for _, f := range fields[1:] {
			if v, ok := strings.CutPrefix(strings.TrimSpace(f), "q="); ok {
				q, _ = strconv.ParseFloat(v, 64)
			}
		}
		if l := Lang(tag); (l == ES || l == EN) && q > bestQ {
			best, bestQ = l, q
		}
	}
	return best
}

func T(l Lang, key string, args ...any) string {
	s, ok := tables[l][key]
	if !ok {
		if s, ok = es[key]; !ok {
			return "[" + key + "]"
		}
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

func Size(l Lang, n int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	f, u := float64(n), 0
	for f >= 1000 && u < len(units)-1 {
		f /= 1000
		u++
	}
	var s string
	switch {
	case u == 0:
		s = strconv.FormatInt(n, 10)
	case f >= 100:
		s = strconv.FormatFloat(f, 'f', 0, 64)
	default:
		s = strconv.FormatFloat(f, 'f', 1, 64)
	}
	if l == ES {
		s = strings.Replace(s, ".", ",", 1)
	}
	return s + " " + units[u]
}
