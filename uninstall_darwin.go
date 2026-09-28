package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

const launchLabel = "page.kido.agent"

func removeAutostart(out io.Writer) {
	home, _ := os.UserHomeDir()
	plist := filepath.Join(home, "Library", "LaunchAgents", launchLabel+".plist")
	// bootout で止めてから外す。KeepAlive のため、先に止めると起こし直される。
	exec.Command("launchctl", "bootout", "gui/"+strconv.Itoa(os.Getuid())+"/"+launchLabel).Run()
	if err := os.Remove(plist); err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(out, "! 自動起動の設定を消せませんでした:", err)
	}
	// 管理者の権限が要るので、ここでは消さずにコマンドを案内する。
	if _, err := os.Stat("/etc/sudoers.d/kido-agent"); err == nil {
		fmt.Fprintln(out, "! 再起動・シャットダウン用の許可が残っています。消すには:")
		fmt.Fprintln(out, "    sudo rm /etc/sudoers.d/kido-agent")
	}
}
