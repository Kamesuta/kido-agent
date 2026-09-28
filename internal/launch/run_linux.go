package launch

import (
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
