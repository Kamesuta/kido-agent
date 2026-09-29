package launch

import "strings"

// CmdUnsafeChars は、引用符で囲んでも cmd に読み違えられる文字。% は引用符の中でも
// 環境変数として展開され(call でもう一度展開される)、^ は call が引用符の中の分まで
// 2 つに増やしてしまう。こういうパスの .bat .cmd は wait では動かさず、
// check で壊れた操作として知らせる(actions)。
const CmdUnsafeChars = "%^"

// CmdPathProblem は、パスを cmd /c call に正しく渡せないときの理由を返す。渡せれば "" 。
func CmdPathProblem(path string) string {
	if strings.ContainsAny(path, CmdUnsafeChars) {
		return "パスに % か ^ が入っていると、終わるのを待つ .bat .cmd を正しく動かせません"
	}
	return ""
}

// batchCmdLine は .bat .cmd を終わるまで待って動かすための cmd の命令行を組む。
//
// Go の exec に引数で渡すと、空白の無いパスは引用符で囲まれないので、& ( ) を
// 含むフォルダ名を cmd が区切りとして読み違える。そこで命令行を自分で組み、
// パスを必ず二重引用符で囲む。/s を付けると cmd は /c の後ろの最初と最後の
// 引用符だけを外すので、中身は call "<パス>" の形のまま残る(引用符で始まる
// 命令を cmd /c が崩す決まりにも触れない)。call を通すのは、スクリプトの
// exit /b の値をそのまま終了コードにするため。/v:off は ! を遅延展開させないため。
// Windows のファイル名に " は使えないので、パスの中で引用符が閉じることはない。
func batchCmdLine(cmdExe, path string) string {
	return `"` + cmdExe + `" /d /v:off /s /c "call "` + path + `""`
}
