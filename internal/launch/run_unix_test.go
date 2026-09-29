//go:build darwin || linux

package launch

import (
	"os"
	"path/filepath"
	"testing"
)

// #! の解釈器で動く。sh で流すと本文の exit 0 で終わるが、解釈器の false なら 1 になる。
func TestRunWaitHonorsShebang(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.sh"), []byte("#!/usr/bin/false\nexit 0\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.sh"), []byte("exit 3\n"), 0o644)
	if code, err := RunWait(dir, "a.sh"); err != nil || code != 1 {
		t.Fatalf("#! の解釈器で動いていない: %d %v", code, err)
	}
	if code, err := RunWait(dir, "b.sh"); err != nil || code != 3 {
		t.Fatalf("#! が無ければ sh で動く: %d %v", code, err)
	}
}
