package paths

import (
	"os"
	"path/filepath"
)

// 待ち受けの番号は本体との取り決め(PROTOCOL.md)で固定。
// 操作用の窓口は同じ PC の CLI からしか使わないので 127.0.0.1 に絞る。
// 環境変数は、動いている常駐アプリの隣で開発版を試すためだけのもの。
var (
	APIAddr     = envOr("KIDO_AGENT_API_ADDR", "0.0.0.0:47821")
	ControlAddr = envOr("KIDO_AGENT_CONTROL_ADDR", "127.0.0.1:47822")
)

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

// ActionsDir は利用者が操作を置くフォルダ。
// 利用者が自分で開いて触る場所なので、隠しフォルダではなくホーム直下に置く。
// 名前は「ここに置いたものがスマホのボタンになる」がそのまま伝わるものにした。
func ActionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "KidoButtons"), nil
}

// ConfigDir は鍵とログの置き場。鍵は秘密なので、利用者が触る ~/KidoButtons とは分けて、
// ドット付きのフォルダにする(Mac・Linux では隠れ、Windows は PrepareConfigDir が隠す)。
// どの OS も同じ場所にそろえる。Windows の %APPDATA% は移動プロファイルで別の PC に
// 同期されうるが、鍵はこの PC 専用のもの。
func ConfigDir() (string, error) {
	if dir := os.Getenv("KIDO_AGENT_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kido-agent"), nil
}

// PrepareConfigDir は置き場を作り、Windows では隠す。
// Windows はドット付きの名前でも隠さないので、ホームに並んで利用者を迷わせる。
func PrepareConfigDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return hide(dir)
}

func KeyFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "key.json"), nil
}
