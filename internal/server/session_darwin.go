package server

// loginSession は、手足役がいなくても誰かがログインしているとみなせるか。
// Mac の常駐アプリは LaunchAgent(gui/<uid> の領域)で動き、これはその利用者が
// 画面にログインしている間だけ起こされ、ログアウトで止められる。手足役は生まれない
// (プロセスは 1 つ)ので、待ち受け役が動いていること自体をログイン中の印にする。
func loginSession() bool { return true }
