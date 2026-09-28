package actions

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// 本体との取り決め(PROTOCOL.md の「上限」)。本体は小さな機器で、
// 1 回に受け取れる量が限られているので、こちらで先に切り詰める。
const (
	MaxActions   = 24
	MaxIDBytes   = 64
	MaxNameRunes = 32
	MaxIconRunes = 40
	MaxConfirm   = 200
	MaxBodyBytes = 4096
)

// IDProblem は、フォルダ名が操作の ID として使えない理由を返す。使えれば "" を返す。
func IDProblem(id string) string {
	switch {
	case !utf8.ValidString(id):
		return "フォルダ名の文字コードが読めません"
	case len(id) > MaxIDBytes:
		return fmt.Sprintf("フォルダ名が長すぎます(%d バイトまで。日本語は 1 文字 3 バイト)", MaxIDBytes)
	case strings.HasPrefix(id, "."):
		return "フォルダ名が . で始まっています"
	case strings.ContainsAny(id, `/\`):
		return "フォルダ名に / か \\ が入っています"
	case strings.IndexFunc(id, unicode.IsControl) >= 0:
		return "フォルダ名に制御文字が入っています"
	}
	return ""
}

// validIcon はアイコン名が lucide の名前の形([a-z0-9-])で、長すぎないかを見る。
func validIcon(s string) bool {
	if s == "" || utf8.RuneCountInString(s) > MaxIconRunes {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

// truncateRunes は文字数(バイト数ではない)で切る。日本語の途中で割らないため。
func truncateRunes(s string, n int) (string, bool) {
	if utf8.RuneCountInString(s) <= n {
		return s, false
	}
	return string([]rune(s)[:n]), true
}
