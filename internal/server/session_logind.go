package server

import "strings"

// localLoginSession は loginctl show-session の出力(Key=Value の行)から、
// 画面の前にいるログインとみなせるかを決める。Linux でしか呼ばないが、
// どの OS の上でもテストできるよう OS を問わないファイルに置く。
//
//   - Class が user 系でないもの(ログイン画面の greeter、裏方の background や
//     manager)は人のログインではない
//   - Remote=yes(ssh など)は画面を持たないので数えない
//   - State が closing(ログアウト中)は数えない
//   - x11・wayland・mir は画面のあるログイン。切り替えて裏に回っても(online)数える
//   - tty は、いま前に出ている(active)ときだけ数える(画面の前の端末ログイン)
func localLoginSession(props string) bool {
	p := map[string]string{}
	for _, line := range strings.Split(props, "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), "="); ok {
			p[k] = v
		}
	}
	class := p["Class"]
	if (class != "user" && !strings.HasPrefix(class, "user-")) || class == "user-incomplete" {
		return false
	}
	if p["Remote"] == "yes" {
		return false
	}
	switch p["State"] {
	case "active":
		return true
	case "online":
		switch p["Type"] {
		case "x11", "wayland", "mir":
			return true
		}
	}
	return false
}
