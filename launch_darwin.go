package main

import (
	"path/filepath"
	"strings"
)

// launchAction は Finder でダブルクリックしたときに近い形で動かす。
// .sh は Terminal の窓を出さずに裏で流し、.command と .app は open に任せる。
func launchAction(a action) error {
	path := filepath.Join(a.Dir, a.Run)
	if strings.EqualFold(filepath.Ext(path), ".sh") {
		return startDetached(a.Dir, "/bin/sh", path)
	}
	return startDetached(a.Dir, "/usr/bin/open", path)
}
