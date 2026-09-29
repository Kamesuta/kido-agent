package server

// loginSession は Windows では常に false。起動時タスクで待ち受け役がログイン前から
// 動くので、ログイン中かは手足役(Run キーで起きる 2 つ目のプロセス)が
// 取りに来ているかだけで決める(helperHub.active)。
func loginSession() bool { return false }
