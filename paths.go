package main

import (
	"os"
	"path/filepath"
	"runtime"
)

// 待ち受けの番号は本体との取り決め(PROTOCOL.md)で固定。
// 操作用の窓口は同じ PC の CLI からしか使わないので 127.0.0.1 に絞る。
// 環境変数は、動いている常駐アプリの隣で開発版を試すためだけのもの。
var (
	apiAddr     = envOr("KIDO_AGENT_API_ADDR", "0.0.0.0:47821")
	controlAddr = envOr("KIDO_AGENT_CONTROL_ADDR", "127.0.0.1:47822")
)

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

// actionsDir は利用者が操作を置くフォルダ。
// 利用者が自分で開いて触る場所なので、隠しフォルダではなくホーム直下に置く。
func actionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Kido"), nil
}

// configDir は鍵とログの置き場。鍵は秘密なので、利用者が触る ~/Kido とは分ける。
// Windows は %APPDATA%、Linux は ~/.config(XDG_CONFIG_HOME)。macOS も ~/.config に
// そろえる(既定の ~/Library/Application Support は空白入りで、手で確かめにくい)。
func configDir() (string, error) {
	if dir := os.Getenv("KIDO_AGENT_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	if runtime.GOOS != "darwin" {
		base, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(base, "kido-agent"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "kido-agent"), nil
}

func keyPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "key.json"), nil
}
