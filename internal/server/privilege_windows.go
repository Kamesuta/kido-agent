package server

import (
	"log"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// SAFER の値(advapi32)。普通のユーザーの水準のトークンを作るために使う。
const (
	saferScopeIDUser      = 2
	saferLevelIDNormal    = 0x20000
	saferLevelOpen        = 1
	saferTokenNullIfEqual = 0 // 既に同じ水準なら NULL を返す指定は使わない(0)
)

var (
	advapi32                       = windows.NewLazySystemDLL("advapi32.dll")
	procSaferCreateLevel           = advapi32.NewProc("SaferCreateLevel")
	procSaferComputeTokenFromLevel = advapi32.NewProc("SaferComputeTokenFromLevel")
	procSaferCloseLevel            = advapi32.NewProc("SaferCloseLevel")
)

// dropPrivilege は、昇格(管理者)で起動されていたら、普通のユーザーの権限で
// 自分を同じ引数で起動し直し、親(昇格側)を終わらせる。起動時タスクは管理者で
// 動くが、操作は普通のユーザーの権限で足りる(SetSuspendState・shutdown.exe は
// Users が持つ SeShutdownPrivilege で効く)。子は昇格していないので、これは繰り返さない。
// 起動し直したら true(親は終わってよい)。
func dropPrivilege() bool {
	token := windows.GetCurrentProcessToken()
	if !token.IsElevated() {
		log.Printf("権限: 普通のユーザー")
		return false
	}
	restricted, err := saferNormalUserToken()
	if err != nil {
		// 落とせなくても動きはするので、管理者のまま続ける
		log.Printf("権限を下げられませんでした(管理者のまま続けます): %v", err)
		return false
	}
	defer restricted.Close()
	if err := relaunchAs(restricted); err != nil {
		log.Printf("権限を下げて起動し直せませんでした(管理者のまま続けます): %v", err)
		return false
	}
	log.Printf("権限: 管理者だったので、普通のユーザーで起動し直しました")
	return true
}

// saferNormalUserToken は「普通のユーザー」の水準のトークンを作る
// (runas /trustlevel:0x20000 と同じ)。
func saferNormalUserToken() (windows.Token, error) {
	var level uintptr
	r, _, e := procSaferCreateLevel.Call(saferScopeIDUser, saferLevelIDNormal,
		saferLevelOpen, uintptr(unsafe.Pointer(&level)), 0)
	if r == 0 {
		return 0, e
	}
	defer procSaferCloseLevel.Call(level)
	var token windows.Token
	r, _, e = procSaferComputeTokenFromLevel.Call(level, 0,
		uintptr(unsafe.Pointer(&token)), saferTokenNullIfEqual, 0)
	if r == 0 {
		return 0, e
	}
	return token, nil
}

// relaunchAs は与えたトークンで、自分と同じ実行ファイル・引数の子を起こす。
func relaunchAs(token windows.Token) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// 引数はそのまま渡す。os.Args[0] は実行ファイルの名前に置き換えておく。
	cmdline := windows.ComposeCommandLine(append([]string{exe}, os.Args[1:]...))
	argv, err := windows.UTF16PtrFromString(cmdline)
	if err != nil {
		return err
	}
	// 普通のユーザーの環境を作る(管理者の環境変数を持ち込まない)。
	var envBlock *uint16
	if err := windows.CreateEnvironmentBlock(&envBlock, token, false); err == nil {
		defer windows.DestroyEnvironmentBlock(envBlock)
	}
	si := &windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{}))}
	var pi windows.ProcessInformation
	err = windows.CreateProcessAsUser(token, nil, argv, nil, nil, false,
		windows.CREATE_UNICODE_ENVIRONMENT|windows.CREATE_NO_WINDOW|syscall.CREATE_NEW_PROCESS_GROUP,
		envBlock, nil, si, &pi)
	if err != nil {
		return err
	}
	windows.CloseHandle(pi.Thread)
	windows.CloseHandle(pi.Process)
	return nil
}
