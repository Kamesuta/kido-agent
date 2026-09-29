package server

import (
	"log"
	"os"
	"strings"
	"syscall"
	"unicode/utf16"
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
// elevatedChildEnv は「これは権限を下げて起動し直した子」の目印。二重に下げて
// 起動し直す無限ループを防ぐ。UAC を切った環境では、権限を下げたトークンでも
// IsElevated が真のままになることがあるので、昇格の判定だけには頼らない。
const elevatedChildEnv = "KIDO_AGENT_ELEVATED_CHILD"

func dropPrivilege() bool {
	if os.Getenv(elevatedChildEnv) == "1" {
		log.Printf("権限: 普通のユーザー(起動し直し済み)")
		return false
	}
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
	// SAFER は管理者の権限を外すが、整合性レベルは High のまま残る(SAFER は UAC より
	// 前の仕組みで、整合性を触らない)。High のままだと結局「管理者相当」で動くので、
	// 明示的に Medium(普通のユーザー)へ下げる。
	if err := setMediumIntegrity(token); err != nil {
		token.Close()
		return 0, err
	}
	return token, nil
}

// setMediumIntegrity はトークンの整合性レベルを Medium(S-1-16-8192)にする。
func setMediumIntegrity(token windows.Token) error {
	sid, err := windows.StringToSid("S-1-16-8192")
	if err != nil {
		return err
	}
	label := windows.Tokenmandatorylabel{
		Label: windows.SIDAndAttributes{Sid: sid, Attributes: windows.SE_GROUP_INTEGRITY},
	}
	return windows.SetTokenInformation(token, windows.TokenIntegrityLevel,
		(*byte)(unsafe.Pointer(&label)), label.Size())
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
	// 今の環境に目印を足して渡す。目印があれば、子は二度と起動し直さない。
	block := envBlockWithGuard()
	si := &windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{}))}
	var pi windows.ProcessInformation
	err = windows.CreateProcessAsUser(token, nil, argv, nil, nil, false,
		windows.CREATE_UNICODE_ENVIRONMENT|windows.CREATE_NO_WINDOW|syscall.CREATE_NEW_PROCESS_GROUP,
		block, nil, si, &pi)
	if err != nil {
		return err
	}
	windows.CloseHandle(pi.Thread)
	windows.CloseHandle(pi.Process)
	return nil
}

// envBlockWithGuard は今の環境変数に目印を足し、UTF-16 の環境ブロック
// (二重ヌル終端)にする。目印があれば、起動し直した子は権限降格を繰り返さない。
func envBlockWithGuard() *uint16 {
	env := append(os.Environ(), elevatedChildEnv+"=1")
	var buf []uint16
	for _, e := range env {
		if strings.IndexByte(e, 0) >= 0 {
			continue
		}
		buf = append(buf, utf16.Encode([]rune(e))...)
		buf = append(buf, 0)
	}
	buf = append(buf, 0)
	return &buf[0]
}
