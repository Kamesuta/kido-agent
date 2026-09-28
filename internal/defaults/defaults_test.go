package defaults

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"kido-agent/internal/actions"
	"kido-agent/internal/paths"
)

func TestDefaultsWrittenOnlyWhenAbsent(t *testing.T) {
	for _, goos := range []string{"windows", "darwin", "linux"} {
		dir := filepath.Join(t.TempDir(), "KidoButtons")
		wrote, err := Write(dir, goos)
		if err != nil || !wrote {
			t.Fatalf("%s: %v", goos, err)
		}
		list, _ := actions.Scan(dir, goos)
		if len(list) != 4 {
			t.Fatalf("%s: 同梱は 4 つ: %d", goos, len(list))
		}
		for _, a := range list {
			if a.Broken || a.Icon == "" || len(a.Warnings) > 0 {
				t.Errorf("%s/%s: %+v", goos, a.ID, a)
			}
		}
		if list[3].Name != "シャットダウン" || list[3].Confirm == nil {
			t.Errorf("%+v", list[3])
		}
		// 使いかた.txt はフォルダ直下のファイルなので操作にならない(上の 4 つで確かめ済み)。
		// Windows だけメモ帳向けに BOM と CRLF にする
		txt, err := os.ReadFile(filepath.Join(dir, "使いかた.txt"))
		if err != nil {
			t.Fatalf("%s: %v", goos, err)
		}
		bom := len(txt) >= 3 && string(txt[:3]) == "\ufeff"
		crlf := strings.Contains(string(txt), "\r\n")
		if bom != (goos == "windows") || crlf != (goos == "windows") {
			t.Errorf("%s: BOM=%v CRLF=%v", goos, bom, crlf)
		}
		// 利用者が消したものを戻さない
		os.RemoveAll(filepath.Join(dir, "10_sleep"))
		if wrote, _ := Write(dir, goos); wrote {
			t.Fatal("~/KidoButtons があるのに書いた")
		}
		if _, err := os.Stat(filepath.Join(dir, "10_sleep")); !os.IsNotExist(err) {
			t.Fatal("消したものが戻った")
		}
	}
}

func TestDefaultsUseHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir, err := paths.ActionsDir()
	if err != nil || dir != filepath.Join(home, "KidoButtons") {
		t.Fatalf("%s %v", dir, err)
	}
	if _, err := Write(dir, runtime.GOOS); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(home); len(entries) != 1 || entries[0].Name() != "KidoButtons" {
		t.Fatalf("作業用の一時フォルダが残っている: %v", entries)
	}
}
