package actions

import (
	"io/fs"
	"path/filepath"
	"strings"

	"kido-agent/internal/launch"
)

// IsRunnable は操作フォルダの中のものが「実行できるファイル」かを OS の決まりで判定する。
// goos を引数に取るのは、どの OS の上でも全 OS の決まりをテストできるようにするため。
func IsRunnable(goos, name string, mode fs.FileMode) bool {
	// ._foo.sh(Mac が他のディスクに残す付随ファイル)などを数えないよう、隠しファイルは見ない。
	if strings.HasPrefix(name, ".") || strings.EqualFold(name, TomlName) {
		return false
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch goos {
	case "windows":
		return mode.IsRegular() && inList(ext, ".ps1", ".bat", ".cmd", ".exe", ".lnk")
	case "darwin":
		// .app はフォルダの形をした 1 つのアプリなので、フォルダでも数える。
		if mode.IsDir() {
			return ext == ".app"
		}
		return mode.IsRegular() && inList(ext, ".sh", ".command")
	default:
		if !mode.IsRegular() {
			return false
		}
		return inList(ext, ".sh", ".desktop") || mode.Perm()&0o111 != 0
	}
}

// Waitable は、終わるまで待って終了コードを受け取れる種類か。ショートカットやアプリは
// シェル(ShellExecute・open・gio)に渡すと手元に子が残らず、終わりも成否も分からない。
func Waitable(goos, name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch goos {
	case "windows":
		return inList(ext, ".ps1", ".bat", ".cmd", ".exe")
	case "darwin":
		return inList(ext, ".sh", ".command")
	}
	return ext != ".desktop"
}

// WaitHint は wait の付いた操作を直す方法。
func WaitHint(goos string) string {
	switch goos {
	case "windows":
		return "終わるのを待てるのは .bat .cmd .ps1 .exe だけです。スクリプトを直接置くか、wait の行を消してください"
	case "darwin":
		return "終わるのを待てるのは .sh .command だけです。スクリプトを直接置くか、wait の行を消してください"
	}
	return "終わるのを待てるのは .sh や実行ファイルだけです。スクリプトを直接置くか、wait の行を消してください"
}

// checkCmdPath は、Windows で wait の .bat .cmd を cmd に正しく渡せるかを見る。
// % ^ の入ったパスは引用符で囲んでも読み違えられるので、押したときに黙って
// 違うものを動かすより、先に壊れた操作として知らせる(launch.CmdUnsafeChars)。
func checkCmdPath(a *Action, goos string) {
	ext := strings.ToLower(filepath.Ext(a.Run))
	if !a.Wait || a.Broken || goos != "windows" || (ext != ".bat" && ext != ".cmd") {
		return
	}
	if p := launch.CmdPathProblem(filepath.Join(a.Dir, a.Run)); p != "" {
		a.fail(p, "フォルダ名やファイル名から % と ^ を外してください")
	}
}

func inList(s string, list ...string) bool {
	for _, v := range list {
		if s == v {
			return true
		}
	}
	return false
}

// needsScreen は、動かすのに画面(ログイン中のデスクトップ)がいりがちな種類か。
func needsScreen(file string) bool {
	switch strings.ToLower(filepath.Ext(file)) {
	case ".lnk", ".app", ".command", ".desktop":
		return true
	}
	return false
}

func RunnableHint(goos string) string {
	switch goos {
	case "windows":
		return "ショートカット(.lnk)か .bat .cmd .ps1 .exe を 1 つだけ置いてください"
	case "darwin":
		return ".app か .sh .command を 1 つだけ置いてください"
	}
	return ".sh .desktop か、実行権限の付いたファイルを 1 つだけ置いてください"
}
