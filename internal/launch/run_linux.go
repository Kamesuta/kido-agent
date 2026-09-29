package launch

import (
	"errors"
	"path/filepath"
	"strings"
)

// Run は .sh を sh に渡し(実行権限が無くても動くように)、
// .desktop はデスクトップ環境と同じ gio に任せ、それ以外は直接動かす。
func Run(dir, file string) error {
	path := filepath.Join(dir, file)
	switch strings.ToLower(filepath.Ext(path)) {
	case ".sh":
		return startDetached(dir, "/bin/sh", path)
	case ".desktop":
		return startDetached(dir, "gio", "launch", path)
	}
	return startDetached(dir, path)
}

// RunWait は終わるまで待って終了コードを返す(kido.toml の wait)。
// .desktop は gio に渡すと終わりが分からないので断る(check でも知らせる)。
func RunWait(dir, file string) (int, error) {
	path := filepath.Join(dir, file)
	switch strings.ToLower(filepath.Ext(path)) {
	case ".sh":
		return runWaited(dir, "/bin/sh", path)
	case ".desktop":
		return 0, errors.New("終わるのを待てない種類です: " + file)
	}
	return runWaited(dir, path)
}
