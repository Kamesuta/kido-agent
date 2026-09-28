package actions

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const TomlName = "kido.toml"

// Action は操作フォルダ 1 つぶん。本体へ送る項目のほか、check で見せる理由も持つ。
type Action struct {
	ID       string
	Name     string
	Icon     string
	Confirm  *string // nil は「確認なし」、"" は「本文なしで確認する」
	Broken   bool
	Dir      string
	Run      string   // 実行するファイル名(Dir の中)
	Invalid  string   // ID として使えない理由。あれば本体へは送らない
	Problems []string // 押せない理由(Broken のとき)
	Warnings []string // 動くが気を付けてほしいこと
}

// 先頭の「番号_」は並び順のためのものなので、表示名からは外す。
var orderPrefix = regexp.MustCompile(`^[0-9]+_`)

func defaultName(id string) string {
	if n := orderPrefix.ReplaceAllString(id, ""); n != "" {
		return n
	}
	return id
}

// Scan は操作フォルダを読み直す。本体から要求が来るたびに呼ぶので、
// 利用者がフォルダを足したり直したりしたら、常駐アプリを再起動しなくても反映される。
func Scan(dir, goos string) ([]Action, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Action
	for _, e := range entries { // ReadDir はフォルダ名の昇順で返す
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		out = append(out, loadAction(filepath.Join(dir, e.Name()), goos))
	}
	return out, nil
}

// Find は一覧に載せるものと同じ条件(ID として使える)で操作を探す。
// 一覧から切り詰めて落ちた操作も、ID が分かっていれば動かせてよい。
func Find(dir, goos, id string) (Action, bool) {
	list, _ := Scan(dir, goos)
	for _, x := range list {
		if x.ID == id && x.Invalid == "" {
			return x, true
		}
	}
	return Action{}, false
}

func loadAction(dir, goos string) Action {
	a := Action{ID: filepath.Base(dir), Dir: dir}
	a.Name = defaultName(a.ID)
	a.Invalid = IDProblem(a.ID)
	cfg := readToml(&a)
	if n, cut := truncateRunes(a.Name, MaxNameRunes); cut {
		a.Name = n
		a.warn("名前が長いので 32 文字で切ります")
	}
	runnables := listRunnables(dir, goos)
	switch {
	case cfg.Run != nil:
		pickRun(&a, *cfg.Run, goos)
	case len(runnables) == 1:
		a.Run = runnables[0]
	case len(runnables) == 0:
		a.fail("実行できるファイルがありません", RunnableHint(goos))
	default:
		a.fail("実行できるファイルが 2 つ以上あります("+strings.Join(runnables, "、")+")",
			"どれを動かすか kido.toml に書いてください。例: run = \""+runnables[0]+"\"")
	}
	return a
}

// pickRun は kido.toml の run を確かめる。フォルダの外を指されると、
// 操作フォルダを見ただけでは何が動くか分からなくなるので、ファイル名だけを許す。
func pickRun(a *Action, run, goos string) {
	if run == "" || run == "." || run == ".." || strings.ContainsAny(run, `/\:`) {
		a.fail("kido.toml の run はフォルダの中のファイル名だけを書けます: "+run,
			"例: run = \"start.bat\"")
		return
	}
	info, err := os.Stat(filepath.Join(a.Dir, run))
	if err != nil {
		a.fail("kido.toml の run のファイルが見つかりません: "+run, "ファイル名の綴りを確かめてください")
		return
	}
	if !IsRunnable(goos, run, info.Mode()) {
		a.fail("kido.toml の run のファイルは実行できる種類ではありません: "+run, RunnableHint(goos))
		return
	}
	a.Run = run
}

func listRunnables(dir, goos string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		info, err := os.Stat(filepath.Join(dir, e.Name())) // ショートカット先ではなく実体を見る
		if err == nil && IsRunnable(goos, e.Name(), info.Mode()) {
			names = append(names, e.Name())
		}
	}
	return names
}

func (a *Action) fail(problem, hint string) {
	a.Broken = true
	a.Problems = append(a.Problems, problem+"\n    → "+hint)
}

func (a *Action) warn(msg string) { a.Warnings = append(a.Warnings, msg) }

func RunnableHint(goos string) string {
	switch goos {
	case "windows":
		return "ショートカット(.lnk)か .bat .cmd .ps1 .exe を 1 つだけ置いてください"
	case "darwin":
		return ".app か .sh .command を 1 つだけ置いてください"
	}
	return ".sh .desktop か、実行権限の付いたファイルを 1 つだけ置いてください"
}
