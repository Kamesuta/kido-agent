package uninstall

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	runKey       = `Software\Microsoft\Windows\CurrentVersion\Run`
	uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\KidoAgent`
	shortcutName = "起動丸の操作フォルダ.lnk"
)

// RemoveAutostart はログオン時の自動起動・「アプリ」一覧・スタートメニューから外す。
func RemoveAutostart(out io.Writer) {
	if k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE); err == nil {
		k.DeleteValue("KidoAgent")
		k.Close()
	}
	registry.DeleteKey(registry.CURRENT_USER, uninstallKey)
	if appdata, err := os.UserConfigDir(); err == nil {
		os.Remove(filepath.Join(appdata, `Microsoft\Windows\Start Menu\Programs`, shortcutName))
	}
	removeBootTask(out)
}

// removeBootTask は起動時タスク KidoAgent を管理者(UAC)で消す。
// 断られても取り除きは続け、残っている旨と消し方を伝える。
func removeBootTask(out io.Writer) {
	cmd := exec.Command("schtasks", "/query", "/tn", "KidoAgent")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	if cmd.Run() != nil {
		return // タスクが無ければ何もしない
	}
	ps := "$p = Start-Process schtasks -Verb RunAs -Wait -PassThru -ArgumentList '/delete','/tn','KidoAgent','/f'; exit $p.ExitCode"
	del := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", ps)
	del.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	if err := del.Run(); err != nil {
		fmt.Fprintln(out, "! 起動時タスク KidoAgent が残っています。管理者の PowerShell で消せます:")
		fmt.Fprintln(out, "    schtasks /delete /tn KidoAgent /f")
	}
}

// RemoveProgram はインストール先のフォルダを消す。動いている自分自身は
// Windows では消せないので、少し待ってから消す cmd を切り離して残す。
func RemoveProgram(out io.Writer) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	dir := filepath.Dir(exe)
	// インストールの手順が作ったフォルダだけを消す(開発中のビルドを巻き込まない)。
	if !strings.EqualFold(filepath.Base(dir), "kido-agent") {
		fmt.Fprintln(out, "! このプログラムはインストール先の外にあるので残します:", dir)
		return
	}
	if _, err := os.Stat(filepath.Join(dir, "kido-agentd.exe")); err != nil {
		fmt.Fprintln(out, "! インストール先ではないようなので残します:", dir)
		return
	}
	removeUserPath(dir)
	cmd := exec.Command("cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
		CmdLine:       `cmd.exe /c ping -n 3 127.0.0.1 >nul & rmdir /s /q "` + dir + `"`,
	}
	cmd.Dir = os.TempDir() // 消すフォルダの中に居座らないため
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(out, "! プログラムを消せませんでした。手で消してください:", dir)
	}
}
