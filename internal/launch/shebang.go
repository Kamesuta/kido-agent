package launch

import (
	"bufio"
	"os"
	"strings"
)

// scriptCommand は、スクリプトを動かす解釈器と引数(最後がスクリプトのパス)を返す。
// 先頭が #! なら、そこに書かれた解釈器(と引数 1 つまで)で動かす。bash の書き方の
// スクリプトを /bin/sh(Ubuntu では dash)で流すと、[[ や配列で止まってしまうため。
// 引数を 1 つまでにするのは、Linux のカーネルが #! の行をそう読む(解釈器の後ろは
// 空白を含めて 1 つの引数)のに合わせるため。#! が無い・読めないときは今までどおり
// /bin/sh に渡す(実行権限が無くても動くように)。
func scriptCommand(path string) (string, []string) {
	if interp, arg := readShebang(path); interp != "" {
		if arg != "" {
			return interp, []string{arg, path}
		}
		return interp, []string{path}
	}
	return "/bin/sh", []string{path}
}

// readShebang は 1 行目の #! を読む。無ければ interp は空。
func readShebang(path string) (interp, arg string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()
	// カーネルも #! の行は数百バイトまでしか読まない。長い行に付き合って全部読まない。
	line, _ := bufio.NewReaderSize(f, 512).ReadSlice('\n')
	s := string(line)
	if !strings.HasPrefix(s, "#!") {
		return "", ""
	}
	// Windows で書いたスクリプトの \r が解釈器の名前に付くと見つからないので外す。
	s = strings.TrimSpace(strings.TrimRight(s[2:], "\r\n"))
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i], strings.TrimSpace(s[i+1:])
	}
	return s, ""
}
