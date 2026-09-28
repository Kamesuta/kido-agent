//go:build !windows

package launch

import (
	"os/exec"
	"syscall"
)

// startDetached は子を別のセッションで起こし、待たない。
// 常駐アプリが再起動されても、動かした操作が道連れで止まらないようにするため。
func startDetached(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() // 終わった子がゾンビとして残らないよう回収だけする
	return nil
}
