package actions

import (
	"io/fs"
	"os"
	"strings"
	"testing"
)

func TestIsRunnableByOS(t *testing.T) {
	reg, exe, dir := fs.FileMode(0o644), fs.FileMode(0o755), fs.ModeDir|0o755
	cases := []struct {
		goos, name string
		mode       fs.FileMode
		want       bool
	}{
		{"windows", "A.PS1", reg, true},
		{"windows", "a.sh", reg, false},
		{"windows", "a.lnk", dir, false},
		{"darwin", "Game.app", dir, true},
		{"darwin", "a.command", reg, true},
		{"darwin", "tool", exe, false},
		{"linux", "tool", exe, true},
		{"linux", "a.desktop", reg, true},
		{"linux", "memo.txt", reg, false},
		{"linux", "kido.toml", exe, false},
		{"linux", ".hidden.sh", reg, false},
	}
	for _, c := range cases {
		if got := IsRunnable(c.goos, c.name, c.mode); got != c.want {
			t.Errorf("%s %s: %v", c.goos, c.name, got)
		}
	}
}

func TestWaitOnScript(t *testing.T) {
	root := t.TempDir()
	d := mkAction(t, root, "90_build", "build.bat")
	writeFile(t, d+"/kido.toml", "wait = true\n")
	a := scanOne(t, root, "windows")
	if !a.Wait || a.Broken || len(a.Warnings) != 0 {
		t.Fatalf("スクリプトには wait を付けられる: %+v", a)
	}
}

// ショートカットやアプリはシェルに渡すので終わりが分からない。黙って待たずに返すより、
// 押せない操作にして check で直し方を見せるほうが、利用者の意図と食い違わない。
func TestWaitOnShortcutIsBroken(t *testing.T) {
	cases := []struct{ goos, file string }{
		{"windows", "game.lnk"},
		{"darwin", "Game.app"},
		{"linux", "game.desktop"},
	}
	for _, c := range cases {
		root := t.TempDir()
		d := mkAction(t, root, "50_game")
		if c.file == "Game.app" {
			os.Mkdir(d+"/Game.app", 0o755)
		} else {
			writeFile(t, d+"/"+c.file, "x")
		}
		writeFile(t, d+"/kido.toml", "wait = true\n")
		a := scanOne(t, root, c.goos)
		if !a.Broken || len(a.Problems) != 1 || !strings.Contains(a.Problems[0], "wait") ||
			!strings.Contains(a.Problems[0], "スクリプトを直接置く") {
			t.Errorf("%s %s: %+v", c.goos, c.file, a)
		}
	}
}

// Windows の wait の .bat は、% ^ の入ったパスだと cmd に正しく渡せないので押せない。
// & ( ) や空白は引用符で囲めば渡せるので、そのまま使える。
func TestWaitBatchUnsafePath(t *testing.T) {
	cases := []struct {
		id, file, goos string
		broken         bool
	}{
		{"60_100%", "start.bat", "windows", true},
		{"60_a^b", "start.cmd", "windows", true},
		{"60_A&B (1)", "start.bat", "windows", false},
		{"60_ok", "start.ps1", "windows", false},
		{"60_100%", "start.sh", "linux", false},
	}
	for _, c := range cases {
		root := t.TempDir()
		d := mkAction(t, root, c.id, c.file)
		writeFile(t, d+"/kido.toml", "wait = true\n")
		a := scanOne(t, root, c.goos)
		if a.Broken != c.broken {
			t.Errorf("%s/%s: %+v", c.id, c.file, a)
		}
	}
	// wait でなければ ShellExecute に渡すので、% が入っていても押せる
	root := t.TempDir()
	mkAction(t, root, "60_100%", "start.bat")
	if a := scanOne(t, root, "windows"); a.Broken {
		t.Fatalf("%+v", a)
	}
}
