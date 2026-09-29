//go:build !windows

package launch

import (
	"errors"
	"os/exec"
	"syscall"
)

// runWaited は子を別のセッションで起こし、終わるまで待って終了コードを返す。
// 待っている間に常駐アプリが止められても、子は道連れにしない(待たない場合と同じ)。
func runWaited(dir, name string, args ...string) (int, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return exitCode(cmd.Run())
}

// exitCode は Run の結果を終了コードに直す。起動できなかったときだけ err を返す。
func exitCode(err error) (int, error) {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		// シグナルで止まったときは ExitCode が -1 になる。シェルの流儀(128+番号)に合わせ、
		// 「0 以外」として本体へ返せるようにする。
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return 128 + int(ws.Signal()), nil
		}
		return ee.ExitCode(), nil
	}
	if err != nil {
		return 0, err
	}
	return 0, nil
}
