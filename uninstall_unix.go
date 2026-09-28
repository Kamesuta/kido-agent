//go:build !windows

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// removeProgram はインストールの手順が置いた場所の実行ファイルだけを消す。
// 開発中のビルドなど、別の場所から動かしたものまで消さないため。
func removeProgram(out io.Writer) {
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

func pauseIfOwnConsole(io.Writer) {}
