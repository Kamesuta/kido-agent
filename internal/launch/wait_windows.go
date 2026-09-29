package launch

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// RunWait は終わるまで待って終了コードを返す(kido.toml の wait)。
// 待たないときの ShellExecute は子の手がかりを返さないので、ここでは自分で
// プロセスを起こす。.bat .cmd .exe は窓を隠さない(ダブルクリックと同じく、
// ログイン中なら黒い窓が出る)。.ps1 は待たないときと同じく窓なしの PowerShell。
func RunWait(dir, file string) (int, error) {
	path := filepath.Join(dir, file)
	var cmd *exec.Cmd
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ps1":
		cmd = exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", path)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	case ".bat", ".cmd":
		// cmd /c に渡すと、スクリプトの exit /b の値がそのまま終了コードになる。
		// 先頭を call にするのは、引用符で始めると cmd /c が引用符を外す決まりに
		// 触れて、空白や & を含むフォルダ名で壊れるため。
		cmd = exec.Command(cmdExe(), "/d", "/c", "call", path)
	case ".exe":
		cmd = exec.Command(path)
	default:
		return 0, errors.New("終わるのを待てない種類です: " + file)
	}
	cmd.Dir = dir
	err := cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), nil
	}
	return 0, err
}

func cmdExe() string {
	if c := strings.TrimSpace(os.Getenv("ComSpec")); c != "" {
		return c
	}
	return "cmd.exe"
}
