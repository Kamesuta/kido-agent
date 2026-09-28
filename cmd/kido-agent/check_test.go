package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kido-agent/internal/auth"
	"kido-agent/internal/control"
)

func TestCheckActionsOutput(t *testing.T) {
	root := t.TempDir()
	mkAction(t, root, "10_sleep", "sleep.ps1")
	mkAction(t, root, "50_two", "a.bat", "b.bat")
	d := mkAction(t, root, "60_warn", "a.bat")
	writeFile(t, d+"/kido.toml", "icon = \"BAD\"\n")
	var out bytes.Buffer
	if bad := checkActions(&out, root, "windows"); !bad {
		t.Fatal("壊れた操作があれば true")
	}
	s := out.String()
	for _, want := range []string{"✓ 10_sleep", "✗ 50_two", `run = "a.bat"`, "! 60_warn", "lucide"} {
		if !strings.Contains(s, want) {
			t.Errorf("%q が無い:\n%s", want, s)
		}
	}
}

func TestCheckActionsTruncationWarning(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 26; i++ {
		mkAction(t, root, strings.Repeat("a", i+1), "a.bat")
	}
	var out bytes.Buffer
	checkActions(&out, root, "windows")
	if !strings.Contains(out.String(), "先頭の 24 個まで") {
		t.Fatal(out.String())
	}
}

func TestCheckPairing(t *testing.T) {
	var out bytes.Buffer
	if bad := checkPairing(&out, control.Status{State: auth.StatePaired, Paired: true}, nil, true); bad || !strings.Contains(out.String(), "✓") {
		t.Fatal(out.String())
	}
	out.Reset()
	if bad := checkPairing(&out, control.Status{}, control.ErrNotRunning, true); !bad || !strings.Contains(out.String(), "動いていません") || !strings.Contains(out.String(), "鍵は保存") {
		t.Fatal(out.String())
	}
	out.Reset()
	if bad := checkPairing(&out, control.Status{State: auth.StateUnpaired}, nil, false); !bad || !strings.Contains(out.String(), "kido-agent pair") {
		t.Fatal(out.String())
	}
}

// mkAction は操作フォルダを作り、中にファイルを置く(中身は名前をそのまま)。
func mkAction(t *testing.T, root, id string, files ...string) string {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		writeFile(t, filepath.Join(dir, f), f)
	}
	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
