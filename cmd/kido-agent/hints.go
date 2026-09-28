package main

import (
	"os"
	"path/filepath"
	"runtime"
)

// startHint は常駐アプリを手で起こすコマンド。自動起動の仕組みが OS ごとに違う。
func startHint() string {
	switch runtime.GOOS {
	case "windows":
		exe, _ := os.Executable()
		return `PowerShell で: Start-Process "` + filepath.Join(filepath.Dir(exe), "kido-agentd.exe") + `"`
	case "darwin":
		return "ターミナルで: launchctl kickstart -k gui/$(id -u)/page.kido.agent"
	}
	return "端末で: systemctl --user restart kido-agent"
}

// firewallHint は「許可」を押し損ねたときに直す場所。
func firewallHint() string {
	switch runtime.GOOS {
	case "windows":
		return "(押し損ねたときは「Windows セキュリティ」→「ファイアウォールとネットワーク保護」→「ファイアウォールによるアプリケーションの許可」で kido-agentd を許可する)"
	case "darwin":
		return "(ファイアウォールを入れている場合は「システム設定」→「ネットワーク」→「ファイアウォール」→「オプション」で kido-agent を許可)"
	}
	return "(ufw を使っている場合は: sudo ufw allow 47821/tcp)"
}
