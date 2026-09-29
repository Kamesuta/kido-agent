package uninstall

import (
	"os/exec"
	"testing"
	"time"
)

// 自分の利用者のプロセスを名前で探して止める。ping を身代わりにする。
func TestStopOwnProcesses(t *testing.T) {
	cmd := exec.Command("ping", "-n", "30", "127.0.0.1")
	if err := cmd.Start(); err != nil {
		t.Skip("ping を起こせません:", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	n, err := stopOwnProcesses("PING.EXE")
	if err != nil || n < 1 {
		t.Fatalf("止められない: %d %v", n, err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		t.Fatal("止まっていない")
	}
}
