package main

import (
	"path/filepath"
	"strings"
)

// launchAction は .sh を sh に渡し(実行権限が無くても動くように)、
// .desktop はデスクトップ環境と同じ gio に任せ、それ以外は直接動かす。
func launchAction(a action) error {
	path := filepath.Join(a.Dir, a.Run)
	switch strings.ToLower(filepath.Ext(path)) {
	case ".sh":
		return startDetached(a.Dir, "/bin/sh", path)
	case ".desktop":
		return startDetached(a.Dir, "gio", "launch", path)
	}
	return startDetached(a.Dir, path)
}
