package launch

import (
	"path/filepath"
	"strings"
)

// Run は Finder でダブルクリックしたときに近い形で動かす。
// .sh は Terminal の窓を出さずに裏で流し、.command と .app は open に任せる。
func Run(dir, file string) error {
	path := filepath.Join(dir, file)
	if strings.EqualFold(filepath.Ext(path), ".sh") {
		return startDetached(dir, "/bin/sh", path)
	}
	return startDetached(dir, "/usr/bin/open", path)
}
