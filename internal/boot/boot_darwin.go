package boot

import (
	"fmt"
	"io"
)

// Mac は FileVault が起動時にパスワードを求めるため、ログイン前には動かせない。
// (ディスクの中身が暗号のままで、鍵も操作フォルダも読めない)
func enable(out io.Writer) int {
	fmt.Fprintln(out, "Mac は、起動時のディスクの暗号化(FileVault)がパスワードを求めるため、")
	fmt.Fprintln(out, "ログイン前には動かせません。ログインすれば、これまでどおり使えます。")
	return 0
}

func disable(out io.Writer) int { return enable(out) }

func status(out io.Writer) int {
	fmt.Fprintln(out, "ログイン前: 非対応(Mac)")
	return 0
}

func usage(out io.Writer) int {
	fmt.Fprintln(out, "使い方: kido-agent boot [on|off]")
	return 2
}
