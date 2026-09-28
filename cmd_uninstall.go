package main

import (
	"fmt"
	"io"
	"os"
	"time"
)

func cmdUninstall(out io.Writer) int {
	defer pauseIfOwnConsole(out)
	fmt.Fprintln(out, "起動丸エージェントを取り除きます。")
	// 自動起動を先に外す。先に止めると、OS が「落ちた」と見て起こし直すことがある。
	removeAutostart(out)
	stopDaemon()
	if dir, err := configDir(); err == nil {
		// 鍵は秘密なので残さない。ログも一緒に消える。
		if err := os.RemoveAll(dir); err != nil {
			fmt.Fprintln(out, "! 鍵とログを消せませんでした:", dir, err)
		}
	}
	removeProgram(out)
	if dir, err := actionsDir(); err == nil {
		fmt.Fprintf(out, "操作フォルダ %s は残しています。要らなければ手で消してください。\n", dir)
	}
	fmt.Fprintln(out, "✓ 取り除きました")
	return 0
}

// stopDaemon は動いている常駐アプリに止まってもらい、窓口が閉じるまで少し待つ。
// 止まる前に鍵やログを消すと、Windows ではファイルが使用中で消せない。
func stopDaemon() {
	if _, err := callControl("POST", "/control/stop"); err != nil {
		return
	}
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if _, err := callControl("GET", "/control/status"); err != nil {
			return
		}
	}
}
