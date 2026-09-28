package uninstall

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var procSendMessageTimeout = windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageTimeoutW")

// removeUserPath はインストールの手順が利用者の Path に足したフォルダを外す。
// 値の種類(REG_EXPAND_SZ か REG_SZ か)を保つのは、%USERPROFILE% などを含む
// ほかの項目を展開されない形に変えて壊さないため。
func removeUserPath(dir string) {
	k, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	v, typ, err := k.GetStringValue("Path")
	if err != nil {
		return
	}
	var keep []string
	for _, p := range strings.Split(v, ";") {
		if !strings.EqualFold(strings.TrimRight(p, `\`), dir) {
			keep = append(keep, p)
		}
	}
	nv := strings.Join(keep, ";")
	if nv == v {
		return
	}
	if typ == registry.EXPAND_SZ {
		err = k.SetExpandStringValue("Path", nv)
	} else {
		err = k.SetStringValue("Path", nv)
	}
	if err == nil {
		broadcastEnvChange()
	}
}

// broadcastEnvChange は開いているエクスプローラーに環境変数が変わったことを知らせる。
// 知らせないと、次にサインインするまで新しく開く窓に古い Path が渡る。
func broadcastEnvChange() {
	const hwndBroadcast, wmSettingChange, smtoAbortIfHung = 0xffff, 0x001A, 0x0002
	env, _ := windows.UTF16PtrFromString("Environment")
	var result uintptr
	procSendMessageTimeout.Call(hwndBroadcast, wmSettingChange, 0, uintptr(unsafe.Pointer(env)),
		smtoAbortIfHung, 1000, uintptr(unsafe.Pointer(&result)))
}
