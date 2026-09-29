package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"kido-agent/internal/control"
	"kido-agent/internal/paths"
	"kido-agent/internal/uninstall"
)

func cmdUninstall(out io.Writer) int {
	defer uninstall.PauseIfOwnConsole(out)
	fmt.Fprintln(out, "起動丸エージェントを取り除きます。")
	// 自動起動を先に外す。先に止めると、OS が「落ちた」と見て起こし直すことがある。
	uninstall.RemoveAutostart(out)
	stopDaemon()
	// 待ち受け役が止まっても、手足役が窓口を取り直して残る(Windows)。自分の分を全部止める。
	uninstall.StopDaemons(out)
	if dir, err := paths.ConfigDir(); err == nil {
		// 鍵は秘密なので残さない。ログも一緒に消える。
		if err := os.RemoveAll(dir); err != nil {
			fmt.Fprintln(out, "! 鍵とログを消せませんでした:", dir, err)
		}
	}
	uninstall.RemoveProgram(out)
	if dir, err := paths.ActionsDir(); err == nil {
		fmt.Fprintf(out, "操作フォルダ %s は残しています。要らなければ手で消してください。\n", dir)
	}
	fmt.Fprintln(out, "✓ 取り除きました")
	return 0
}

// stopDaemon は動いている常駐アプリに止まってもらい、窓口が閉じるまで少し待つ。
// 止まる前に鍵やログを消すと、Windows ではファイルが使用中で消せない。
func stopDaemon() {
	if _, err := control.Call("POST", "/control/stop"); err != nil {
		return
	}
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if _, err := control.Call("GET", "/control/status"); err != nil {
			return
		}
	}
}
