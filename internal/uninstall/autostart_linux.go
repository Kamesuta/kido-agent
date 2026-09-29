package uninstall

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// RemoveAutostart はログイン時の自動起動(systemd --user)を外す。
func RemoveAutostart(out io.Writer) {
	exec.Command("systemctl", "--user", "disable", "--now", "kido-agent.service").Run()
	dir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	unit := filepath.Join(dir, "systemd", "user", "kido-agent.service")
	if err := os.Remove(unit); err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(out, "! 自動起動の設定を消せませんでした:", err)
	}
	exec.Command("systemctl", "--user", "daemon-reload").Run()
	// ログイン前にも動くようにしていた(linger)なら戻す。
	if user, err := exec.Command("id", "-un").Output(); err == nil {
		exec.Command("loginctl", "disable-linger", string(user[:len(user)-1])).Run()
	}
}
