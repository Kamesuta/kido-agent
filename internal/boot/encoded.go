package boot

import (
	"encoding/base64"
	"strings"
	"unicode/utf16"
)

// maxCommandLine は Windows の命令行の長さの上限(CreateProcess、文字数)。
const maxCommandLine = 32767

// encodedCommand は埋め込んだ .ps1 を、ファイルを経由せず powershell -EncodedCommand に
// 渡せる形(UTF-16LE の base64)にする。%TEMP% に固定の名前で置くと、UAC の確認を
// 待つ間に同じ利用者の別のプロセスが中身を差し替え、管理者の権限で動かせてしまうため。
// 引数は -EncodedCommand では渡せないので、スクリプトを & { … } で包み、名前付きで渡す。
// args は「-名前, 値」の組を並べたもの。
//
// 丸ごとの行のコメントと空行は落とす。命令行の長さの上限(maxCommandLine)に
// 余裕を持たせるため(中身は UTF-16 で 2 倍、base64 でさらに 4/3 倍になる)。
func encodedCommand(script string, args ...string) string {
	var b strings.Builder
	b.WriteString("& {\n")
	for _, line := range strings.Split(strings.ReplaceAll(script, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("}")
	for i, a := range args {
		if i%2 == 0 {
			b.WriteString(" " + a)
		} else {
			b.WriteString(" " + psQuote(a))
		}
	}
	u := utf16.Encode([]rune(b.String()))
	raw := make([]byte, 2*len(u))
	for i, c := range u {
		raw[2*i], raw[2*i+1] = byte(c), byte(c>>8)
	}
	return base64.StdEncoding.EncodeToString(raw)
}

// psQuote は値を PowerShell の単一引用符の文字列にする。PowerShell は ' のほかに
// ‘ ’ ‚ ‛ も単一引用符として読むので、どれも 2 つ重ねて閉じないようにする。
func psQuote(s string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range s {
		b.WriteRune(r)
		switch r {
		case '\'', '\u2018', '\u2019', '\u201A', '\u201B':
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}
