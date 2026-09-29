package boot

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// hidden は補助のコマンド(schtasks・powershell)を黒い窓を出さずに動かす設定。
func hidden() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
}
