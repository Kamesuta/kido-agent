package defaults

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// 同梱の操作(スリープ・ロック・再起動・シャットダウン)。OS ごとに中身が違う。
//
//go:embed all:data
var defaultsFS embed.FS

// Write は ~/Kido そのものが無いときだけ同梱の操作を置く。
// 利用者が消した操作を勝手に戻さないよう、フォルダが 1 度でもあれば触らない。
func Write(dir, goos string) (bool, error) {
	if _, err := os.Lstat(dir); !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	// 途中で失敗しても半端な ~/Kido が残らない(次の起動でやり直せる)よう、
	// 隣に作ってから名前を付け替える。
	tmp, err := os.MkdirTemp(filepath.Dir(dir), ".Kido-init-")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(tmp)
	root := path.Join("data", goos)
	err = fs.WalkDir(defaultsFS, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == root {
			return err
		}
		dst := filepath.Join(tmp, filepath.FromSlash(strings.TrimPrefix(p, root+"/")))
		if d.IsDir() {
			return os.Mkdir(dst, 0o755)
		}
		data, err := defaultsFS.ReadFile(p)
		if err != nil {
			return err
		}
		mode := fs.FileMode(0o644)
		if strings.HasSuffix(p, ".sh") {
			mode = 0o755
		}
		return os.WriteFile(dst, data, mode)
	})
	if err != nil {
		return false, err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		return false, err
	}
	return true, os.Rename(tmp, dir)
}
