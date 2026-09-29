package boot

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Linux は linger(ログインしていなくても user のサービスを動かし続ける設定)で、
// ログイン前でも待ち受け役を動かせる。root 権限が要るときは案内する。
func enable(out io.Writer) int  { return setLinger(out, true) }
func disable(out io.Writer) int { return setLinger(out, false) }

func setLinger(out io.Writer, on bool) int {
	verb := "enable-linger"
	if !on {
		verb = "disable-linger"
	}
	user := currentUser()
	if err := exec.Command("loginctl", verb, user).Run(); err != nil {
		// 普通は本人のぶんは権限なしでできるが、環境によっては root が要る
		fmt.Fprintf(out, "✗ 設定できませんでした。次を試してください:\n    sudo loginctl %s %s\n", verb, user)
		return 1
	}
	if on {
		fmt.Fprintln(out, "✓ ログイン前でも動くようにしました(linger)")
	} else {
		fmt.Fprintln(out, "✓ ログイン前には動かさないようにしました")
	}
	return 0
}

func status(out io.Writer) int {
	cmd := exec.Command("loginctl", "show-user", currentUser(), "-p", "Linger", "--value")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Run()
	if strings.TrimSpace(buf.String()) == "yes" {
		fmt.Fprintln(out, "ログイン前: 動きます(linger 有効)")
	} else {
		fmt.Fprintln(out, "ログイン前: 動きません(kido-agent boot on で有効にできます)")
	}
	return 0
}

func usage(out io.Writer) int {
	fmt.Fprintln(out, "使い方: kido-agent boot [on|off]")
	return 2
}

func currentUser() string {
	out, err := exec.Command("id", "-un").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
