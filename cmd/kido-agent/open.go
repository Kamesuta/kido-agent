package main

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"

	"github.com/Kamesuta/kido-agent/internal/defaults"
	"github.com/Kamesuta/kido-agent/internal/paths"
)

func cmdOpen(out io.Writer) int {
	dir, err := paths.ActionsDir()
	if err != nil {
		fmt.Fprintln(out, "✗ ホームフォルダが分かりません:", err)
		return 1
	}
	// 常駐アプリより先に開かれても空のフォルダにならないよう、同梱の操作を置いておく。
	if _, err := defaults.Write(dir, runtime.GOOS); err != nil {
		fmt.Fprintln(out, "✗ 操作フォルダを作れません:", err)
		return 1
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	// explorer.exe は開けても終了コード 1 を返すので、待たずに起動の成否だけ見る。
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(out, "✗ 開けませんでした:", err)
		fmt.Fprintln(out, "  場所:", dir)
		return 1
	}
	fmt.Fprintln(out, "操作フォルダを開きました:", dir)
	return 0
}
