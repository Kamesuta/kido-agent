//go:build !windows

// Package uninstall は kido-agent uninstall の OS ごとの片付け。
package uninstall

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// RemoveProgram はインストールの手順が置いた場所の実行ファイルだけを消す。
// 開発中のビルドなど、別の場所から動かしたものまで消さないため。
func RemoveProgram(out io.Writer) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	home, _ := os.UserHomeDir()
	installed := filepath.Join(home, ".local", "bin", "kido-agent")
	if exe != installed {
		fmt.Fprintln(out, "! このプログラムはインストール先の外にあるので残します:", exe)
		return
	}
	if err := os.Remove(exe); err != nil {
		fmt.Fprintln(out, "! プログラムを消せませんでした:", err)
	}
}

// PauseIfOwnConsole は Windows でだけ意味がある(端末は結果を読む前に閉じない)。
func PauseIfOwnConsole(io.Writer) {}
