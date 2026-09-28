package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"kido-agent/internal/auth"
	"kido-agent/internal/control"
)

func TestWaitPairing(t *testing.T) {
	steps := []control.Status{
		{State: auth.StatePairing, Remaining: 599},
		{State: auth.StatePaired, Paired: true},
	}
	var out bytes.Buffer
	get := func() (control.Status, error) { s := steps[0]; steps = steps[1:]; return s, nil }
	code := waitPairing(&out, control.Status{State: auth.StatePairing, Remaining: 600}, get, func(time.Duration) {})
	if code != 0 || !strings.Contains(out.String(), "残り 10:00") || !strings.Contains(out.String(), "✓ 本体とつながりました") {
		t.Fatalf("%d\n%s", code, out.String())
	}
}

func TestWaitPairingTimeout(t *testing.T) {
	var out bytes.Buffer
	get := func() (control.Status, error) { return control.Status{State: auth.StateUnpaired}, nil }
	code := waitPairing(&out, control.Status{State: auth.StatePairing, Remaining: 1}, get, func(time.Duration) {})
	s := out.String()
	if code != 1 || !strings.Contains(s, "登録したか") || !strings.Contains(s, "「許可」") || !strings.Contains(s, "電源") || !strings.Contains(s, "kido-agent pair") {
		t.Fatal(s)
	}
}
