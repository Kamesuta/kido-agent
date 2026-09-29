// kido-agent は起動丸の本体から届く操作を、この PC で実行する常駐アプリ。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kido-agent/internal/boot"
	"kido-agent/internal/server"
)

// version はリリースのビルドで -ldflags -X main.version=... により埋め込む。
var version = "dev"

const usage = `起動丸エージェント

使い方: kido-agent <コマンド>

  serve      常駐する(普段はログイン時に自動で起動します)
  pair       起動丸の本体と組む(10 分間だけ受け付けます)
  check      操作フォルダの中身と、組めているかを確かめる
  open       操作フォルダ(~/KidoButtons)を開く
  boot       ログイン前(電源を入れただけ)でも使えるようにする(on/off で切り替え)
  uninstall  このアプリを取り除く(~/KidoButtons は残します)
  version    版を表示する
`

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	cmd := defaultCommand(args[0])
	if len(args) > 1 {
		cmd = args[1]
	}
	switch cmd {
	case "serve":
		return server.Serve(version)
	case "pair":
		return cmdPair(os.Stdout)
	case "check":
		return cmdCheck(os.Stdout)
	case "open":
		return cmdOpen(os.Stdout)
	case "boot":
		arg := ""
		if len(args) > 2 {
			arg = args[2]
		}
		return boot.Run(os.Stdout, arg)
	case "uninstall":
		return cmdUninstall(os.Stdout)
	case "version", "--version", "-v":
		fmt.Println("kido-agent", version)
		return 0
	case "", "help", "--help", "-h":
		fmt.Print(usage)
		return 0
	}
	fmt.Fprintf(os.Stderr, "知らないコマンドです: %s\n\n%s", cmd, usage)
	return 2
}

// defaultCommand は引数なしで起動されたときの動きを実行ファイルの名前で決める。
// Windows ではログオン時の自動起動に GUI 版(kido-agentd.exe)を使い、
// 引数を付けずに登録しても常駐するようにしておく。
func defaultCommand(argv0 string) string {
	base := strings.ToLower(filepath.Base(argv0))
	if strings.HasPrefix(base, "kido-agentd") {
		return "serve"
	}
	return ""
}
