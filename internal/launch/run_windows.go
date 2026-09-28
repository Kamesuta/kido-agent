package launch

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// Run は .ps1 だけ窓を出さずに PowerShell で流し、それ以外は
// エクスプローラーでダブルクリックしたのと同じ ShellExecute に任せる。
// .ps1 をダブルクリックするとメモ帳で開いてしまうので、ここだけ別扱いにする。
func Run(dir, file string) error {
	path := filepath.Join(dir, file)
	if strings.EqualFold(filepath.Ext(path), ".ps1") {
		cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", path)
		cmd.Dir = dir
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
		if err := cmd.Start(); err != nil {
			return err
		}
		go cmd.Wait()
		return nil
	}
	return shellOpen(path, dir)
}

// shellOpen は COM を初期化したスレッドで ShellExecute を呼ぶ。
// ショートカット(.lnk)の解決などはシェル拡張(COM)を通るため、初期化が要る。
func shellOpen(path, dir string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE); err == nil {
		defer windows.CoUninitialize()
	}
	verb, _ := windows.UTF16PtrFromString("open")
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	cwd, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, file, nil, cwd, windows.SW_SHOWNORMAL)
}
