package actions

import (
	"io/fs"
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
