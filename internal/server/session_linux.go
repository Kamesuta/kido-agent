package server

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// loginSession は、この利用者がいま画面の前にログインしているかを logind に尋ねる。
// Linux の常駐アプリは systemd --user で動き、linger を入れるとログイン前から
// 動くので、待ち受け役がいることはログインの印にならない。手足役も生まれない
// (プロセスは 1 つ)。logind は PAM を通ったログインをセッションとして数えている
// 唯一の場所なので、ここに聞くのが確実。loginctl が無い・答えない環境では
// 「ログインしていない」とみなす(require_login の操作が押せないだけで、安全側)。
func loginSession() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	uid := strconv.Itoa(os.Getuid())
	out, err := exec.CommandContext(ctx, "loginctl", "show-user", uid, "-p", "Sessions", "--value").Output()
	if err != nil {
		return false // logind がこの利用者を知らない(誰もログインしていない)
	}
	for _, id := range strings.Fields(string(out)) {
		props, err := exec.CommandContext(ctx, "loginctl", "show-session", id,
			"-p", "Type", "-p", "Class", "-p", "State", "-p", "Remote").Output()
		if err == nil && localLoginSession(string(props)) {
			return true
		}
	}
	return false
}
