package launch

import (
	"errors"
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

// RunWait は終わるまで待って終了コードを返す(kido.toml の wait)。.command は
// Terminal に渡すと終わりが分からないので、中身のシェルスクリプトを窓なしで流す。
func RunWait(dir, file string) (int, error) {
	switch strings.ToLower(filepath.Ext(file)) {
	case ".sh", ".command":
		return runWaited(dir, "/bin/sh", filepath.Join(dir, file))
	}
	return 0, errors.New("終わるのを待てない種類です: " + file)
}
