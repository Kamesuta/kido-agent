package defaults

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Kamesuta/kido-agent/internal/actions"
	"github.com/Kamesuta/kido-agent/internal/paths"
)

func TestDefaultsWrittenOnlyWhenAbsent(t *testing.T) {
	for _, goos := range []string{"windows", "darwin", "linux"} {
		dir := filepath.Join(t.TempDir(), "Kido")
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
		// 利用者が消したものを戻さない
		os.RemoveAll(filepath.Join(dir, "00_sleep"))
		if wrote, _ := Write(dir, goos); wrote {
			t.Fatal("~/Kido があるのに書いた")
		}
		if _, err := os.Stat(filepath.Join(dir, "00_sleep")); !os.IsNotExist(err) {
			t.Fatal("消したものが戻った")
		}
	}
}

func TestDefaultsUseHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir, err := paths.ActionsDir()
	if err != nil || dir != filepath.Join(home, "Kido") {
		t.Fatalf("%s %v", dir, err)
	}
	if _, err := Write(dir, runtime.GOOS); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(home); len(entries) != 1 || entries[0].Name() != "Kido" {
		t.Fatalf("作業用の一時フォルダが残っている: %v", entries)
	}
}
