package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func scanOne(t *testing.T, root, goos string) action {
	t.Helper()
	list, err := scanActions(root, goos)
	if err != nil || len(list) != 1 {
		t.Fatalf("%v %+v", err, list)
	}
	return list[0]
}

func TestScanOneRunnable(t *testing.T) {
	root := t.TempDir()
	mkAction(t, root, "10_Minecraft起動", "start.lnk", "memo.txt", "desktop.ini")
	a := scanOne(t, root, "windows")
	if a.Broken || a.Run != "start.lnk" || a.Name != "Minecraft起動" || a.Icon != "" || a.Confirm != nil {
		t.Fatalf("%+v", a)
	}
}

func TestScanBrokenCases(t *testing.T) {
	root := t.TempDir()
	mkAction(t, root, "a_none", "memo.txt")
	mkAction(t, root, "b_two", "a.bat", "b.cmd")
	d := mkAction(t, root, "c_badtoml", "a.bat")
	writeFile(t, d+"/kido.toml", "name = ゲーム\n")
	d = mkAction(t, root, "d_badrun", "a.bat")
	writeFile(t, d+"/kido.toml", "run = \"../x.bat\"\n")
	d = mkAction(t, root, "e_missing", "a.bat")
	writeFile(t, d+"/kido.toml", "run = \"b.bat\"\n")
	d = mkAction(t, root, "f_notrunnable", "a.bat", "memo.txt")
	writeFile(t, d+"/kido.toml", "run = \"memo.txt\"\n")
	list, _ := scanActions(root, "windows")
	if len(list) != 6 {
		t.Fatal(len(list))
	}
	for _, a := range list {
		if !a.Broken || len(a.Problems) == 0 {
			t.Errorf("%s は壊れているはず: %+v", a.ID, a)
		}
	}
	if !strings.Contains(list[1].Problems[0], `run = "a.bat"`) {
		t.Errorf("2 つあるときは run の書き方を案内する: %s", list[1].Problems[0])
	}
}

func TestScanTomlRunAndBOM(t *testing.T) {
	root := t.TempDir()
	d := mkAction(t, root, "20_restart", "a.bat", "start.bat")
	toml := "\xEF\xBB\xBFname = \"再起動\"\nicon = \"rotate-cw\"\nconfirm = \"\"\nrun = \"start.bat\"\ncolor = \"red\"\n"
	writeFile(t, d+"/kido.toml", toml)
	a := scanOne(t, root, "windows")
	if a.Broken || a.Run != "start.bat" || a.Name != "再起動" || a.Icon != "rotate-cw" {
		t.Fatalf("%+v", a)
	}
	if a.Confirm == nil || *a.Confirm != "" {
		t.Fatal("confirm = \"\" は本文なしの確認として残す")
	}
	if len(a.Warnings) != 1 || !strings.Contains(a.Warnings[0], "color") {
		t.Fatalf("知らないキーは警告だけ: %v", a.Warnings)
	}
}

func TestScanBadIconDropped(t *testing.T) {
	root := t.TempDir()
	d := mkAction(t, root, "x", "a.bat")
	writeFile(t, d+"/kido.toml", "icon = \"Moon Icon\"\n")
	a := scanOne(t, root, "windows")
	if a.Icon != "" || a.Broken || len(a.Warnings) != 1 {
		t.Fatalf("%+v", a)
	}
}

func TestScanIgnoresDotAndFiles(t *testing.T) {
	root := t.TempDir()
	mkAction(t, root, ".hidden", "a.bat")
	mkAction(t, root, "ok", "a.bat", "._a.bat")
	writeFile(t, filepath.Join(root, "loose.bat"), "")
	a := scanOne(t, root, "windows")
	if a.ID != "ok" || a.Broken {
		t.Fatalf("%+v", a)
	}
}

func TestIDRules(t *testing.T) {
	for id, ok := range map[string]bool{
		"00_sleep":              true,
		"マインクラフト":               true,
		strings.Repeat("x", 64): true,
		strings.Repeat("x", 65): false,
		strings.Repeat("あ", 22): false, // 66 バイト
		"a\x01b":                false,
		`a\b`:                   false,
		".x":                    false,
	} {
		if (idProblem(id) == "") != ok {
			t.Errorf("%q: %s", id, idProblem(id))
		}
	}
	root := t.TempDir()
	mkAction(t, root, strings.Repeat("x", 70), "a.bat")
	if a := scanOne(t, root, "windows"); a.Invalid == "" {
		t.Fatal("長すぎる ID は送らない")
	}
	if body, _ := listBody([]action{{ID: "x", Invalid: "bad"}}); string(body) != `{"actions":[]}` {
		t.Fatal(string(body))
	}
}

func TestDefaultName(t *testing.T) {
	for in, want := range map[string]string{"00_sleep": "sleep", "sleep": "sleep", "12_": "12_", "1_2_x": "2_x"} {
		if got := defaultName(in); got != want {
			t.Errorf("%s → %s", in, got)
		}
	}
}

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
		if got := isRunnable(c.goos, c.name, c.mode); got != c.want {
			t.Errorf("%s %s: %v", c.goos, c.name, got)
		}
	}
}

func TestScanMissingDir(t *testing.T) {
	if _, err := scanActions(filepath.Join(t.TempDir(), "none"), "linux"); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}
