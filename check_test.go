package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestCheckActionsOutput(t *testing.T) {
	root := t.TempDir()
	mkAction(t, root, "00_sleep", "sleep.ps1")
	mkAction(t, root, "10_two", "a.bat", "b.bat")
	d := mkAction(t, root, "20_warn", "a.bat")
	writeFile(t, d+"/kido.toml", "icon = \"BAD\"\n")
	var out bytes.Buffer
	if bad := checkActions(&out, root, "windows"); !bad {
		t.Fatal("壊れた操作があれば true")
	}
	s := out.String()
	for _, want := range []string{"✓ 00_sleep", "✗ 10_two", `run = "a.bat"`, "! 20_warn", "lucide"} {
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
	if bad := checkPairing(&out, controlStatus{State: statePaired, Paired: true}, nil, true); bad || !strings.Contains(out.String(), "✓") {
		t.Fatal(out.String())
	}
	out.Reset()
	if bad := checkPairing(&out, controlStatus{}, errNotRunning, true); !bad || !strings.Contains(out.String(), "動いていません") || !strings.Contains(out.String(), "鍵は保存") {
		t.Fatal(out.String())
	}
	out.Reset()
	if bad := checkPairing(&out, controlStatus{State: stateUnpaired}, nil, false); !bad || !strings.Contains(out.String(), "kido-agent pair") {
		t.Fatal(out.String())
	}
}

func TestWaitPairing(t *testing.T) {
	steps := []controlStatus{
		{State: statePairing, Remaining: 599},
		{State: statePaired, Paired: true},
	}
	var out bytes.Buffer
	get := func() (controlStatus, error) { s := steps[0]; steps = steps[1:]; return s, nil }
	code := waitPairing(&out, controlStatus{State: statePairing, Remaining: 600}, get, func(time.Duration) {})
	if code != 0 || !strings.Contains(out.String(), "残り 10:00") || !strings.Contains(out.String(), "✓ 本体とつながりました") {
		t.Fatalf("%d\n%s", code, out.String())
	}
}

func TestWaitPairingTimeout(t *testing.T) {
	var out bytes.Buffer
	get := func() (controlStatus, error) { return controlStatus{State: stateUnpaired}, nil }
	code := waitPairing(&out, controlStatus{State: statePairing, Remaining: 1}, get, func(time.Duration) {})
	s := out.String()
	if code != 1 || !strings.Contains(s, "登録したか") || !strings.Contains(s, "「許可」") || !strings.Contains(s, "電源") || !strings.Contains(s, "kido-agent pair") {
		t.Fatal(s)
	}
}

func TestControlRequiresHeader(t *testing.T) {
	ta := newTestAgent(t)
	h := ta.controlHandler()
	if r := call(t, h, "POST", "/control/pair", ""); r.status != 403 || ta.state() != stateUnpaired {
		t.Fatal("独自ヘッダの無い要求で組める時間が開いた")
	}
}
