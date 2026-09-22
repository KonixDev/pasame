package session

import (
	"io/fs"
	"path/filepath"
	"strings"
)

var junk = map[string]bool{"thumbs.db": true, "desktop.ini": true}

func hidden(name string) bool {
	return strings.HasPrefix(name, ".") || junk[strings.ToLower(name)]
}

// walk recorre root una sola vez. No sigue symlinks. Rel usa "/" siempre (va al ZIP).
func walk(root string) ([]File, error) {
	parent := filepath.Dir(root)
	var out []File
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // un subdirectorio ilegible no arruina el resto
		}
		if p != root && hidden(d.Name()) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil // directorios, symlinks, sockets
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(parent, p)
		out = append(out, File{
			Name: d.Name(), Rel: filepath.ToSlash(rel), Abs: p,
			Size: info.Size(), ModTime: info.ModTime(),
		})
		return nil
	})
	return out, err
}
