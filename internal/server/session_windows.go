package server

import "golang.org/x/sys/windows"

// loginSession は、待ち受け役そのものがログイン中の画面(セッション 0 以外)で
// 動いているか。起動時タスクを入れていない PC では、Run キーで起きた 1 つだけが
// 待ち受け役になり、手足役は生まれない。手足役だけで決めると、ログイン中でも
// require_login の操作がずっと押せなかった。
// 起動時タスクの待ち受け役はセッション 0 で動くので false になり、今までどおり
// 手足役が取りに来ているかで決まる。
func loginSession() bool {
	var id uint32
	if err := windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &id); err != nil {
		return false
	}
	return id != 0
}
