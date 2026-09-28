package actions

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// IsRunnable は操作フォルダの中のものが「実行できるファイル」かを OS の決まりで判定する。
// goos を引数に取るのは、どの OS の上でも全 OS の決まりをテストできるようにするため。
func IsRunnable(goos, name string, mode fs.FileMode) bool {
	// ._foo.sh(Mac が他のディスクに残す付随ファイル)などを数えないよう、隠しファイルは見ない。
	if strings.HasPrefix(name, ".") || strings.EqualFold(name, TomlName) {
		return false
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch goos {
	case "windows":
		return mode.IsRegular() && inList(ext, ".ps1", ".bat", ".cmd", ".exe", ".lnk")
	case "darwin":
		// .app はフォルダの形をした 1 つのアプリなので、フォルダでも数える。
		if mode.IsDir() {
			return ext == ".app"
		}
		return mode.IsRegular() && inList(ext, ".sh", ".command")
	default:
		if !mode.IsRegular() {
			return false
		}
		return inList(ext, ".sh", ".desktop") || mode.Perm()&0o111 != 0
	}
}

func inList(s string, list ...string) bool {
	for _, v := range list {
		if s == v {
			return true
		}
	}
	return false
}
