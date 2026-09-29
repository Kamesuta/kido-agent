// Package boot は「ログイン前(電源を入れただけ)でも動く」ようにする段取り。
// OS ごとに仕組みが違う(Windows は起動時タスク、Linux は linger、Mac は非対応)。
package boot

import "io"

// Enable はログイン前でも動くようにする。Disable は元に戻す。Status は今の状態を出す。
// 実体は OS ごとのファイル(boot_windows.go など)にある。

// action は on/off/status の入口。cmd から呼ぶ。
func Run(out io.Writer, arg string) int {
	switch arg {
	case "on":
		return enable(out)
	case "off":
		return disable(out)
	case "", "status":
		return status(out)
	}
	return usage(out)
}
