package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"runtime"

	"kido-agent/internal/actions"
	"kido-agent/internal/auth"
	"kido-agent/internal/control"
	"kido-agent/internal/paths"
)

func cmdCheck(out io.Writer) int {
	dir, err := paths.ActionsDir()
	if err != nil {
		fmt.Fprintln(out, "✗ ホームフォルダが分かりません:", err)
		return 1
	}
	st, stErr := control.Call("GET", "/control/status")
	hasKey := false
	if kp, err := paths.KeyFile(); err == nil {
		key, _ := auth.LoadKey(kp)
		hasKey = key != nil
	}
	fmt.Fprintf(out, "起動丸エージェント %s\n\n", version)
	bad := checkPairing(out, st, stErr, hasKey)
	fmt.Fprintln(out)
	if checkActions(out, dir, runtime.GOOS) {
		bad = true
	}
	if bad {
		return 1
	}
	return 0
}

// checkPairing は組めているかを見せる。常駐アプリが動いていればその答えを、
// 動いていなければ鍵のファイルの有無を手がかりにする。
func checkPairing(out io.Writer, st control.Status, stErr error, hasKey bool) (bad bool) {
	if stErr != nil {
		reportControlError(out, stErr)
		if hasKey {
			fmt.Fprintln(out, "  (鍵は保存されているので、起動すれば本体から使えます)")
		}
		return true
	}
	switch st.State {
	case auth.StatePaired:
		fmt.Fprintln(out, "✓ 起動丸の本体と組めています")
	case auth.StatePairing:
		fmt.Fprintf(out, "… 組める時間が開いています(残り %d:%02d)\n", st.Remaining/60, st.Remaining%60)
	default:
		fmt.Fprintln(out, "✗ まだ起動丸の本体と組んでいません")
		fmt.Fprintln(out, "    → kido-agent pair を実行して、スマホの起動ページでこの PC を登録してください")
		return true
	}
	return false
}

// checkActions は操作フォルダを 1 つずつ確かめる。✗ が 1 つでもあれば true を返す。
func checkActions(out io.Writer, dir, goos string) (bad bool) {
	fmt.Fprintln(out, "操作フォルダ:", dir)
	list, err := actions.Scan(dir, goos)
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintln(out, "✗ 操作フォルダがありません → kido-agent open で作れます")
		return true
	} else if err != nil {
		fmt.Fprintln(out, "✗ 操作フォルダを読めません:", err)
		return true
	}
	valid := 0
	for _, a := range list {
		switch {
		case a.Invalid != "":
			fmt.Fprintf(out, "✗ %s: 本体に送りません: %s\n    → フォルダ名を変えてください\n", a.ID, a.Invalid)
			bad = true
			continue
		case a.Broken:
			fmt.Fprintf(out, "✗ %s(%s): 押せません\n", a.ID, a.Name)
			bad = true
		case len(a.Warnings) > 0:
			fmt.Fprintf(out, "! %s(%s)→ %s\n", a.ID, a.Name, a.Run)
		default:
			fmt.Fprintf(out, "✓ %s(%s)→ %s\n", a.ID, a.Name, a.Run)
		}
		valid++
		for _, p := range a.Problems {
			fmt.Fprintln(out, "    "+p)
		}
		for _, w := range a.Warnings {
			fmt.Fprintln(out, "    ! "+w)
		}
	}
	if valid == 0 {
		fmt.Fprintln(out, "! 操作がありません。~/KidoButtons にフォルダを作り、ショートカットやスクリプトを置いてください")
	}
	if _, sent := actions.ListBody(list); sent < valid {
		fmt.Fprintf(out, "! 本体に送れるのは先頭の %d 個までです(%d 個まで・合わせて %d バイトまで)。残りは表示されません\n",
			sent, actions.MaxActions, actions.MaxBodyBytes)
	}
	return bad
}
