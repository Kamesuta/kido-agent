package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"kido-agent/internal/auth"
	"kido-agent/internal/control"
	"kido-agent/internal/defaults"
	"kido-agent/internal/logfile"
	"kido-agent/internal/paths"
)

// Serve は常駐する。止める要求かシグナルが来るまで戻らない。
func Serve(version string) int {
	cfgDir, err := paths.ConfigDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "設定の置き場が分かりません:", err)
		return 1
	}
	if err := paths.PrepareConfigDir(cfgDir); err != nil {
		fmt.Fprintln(os.Stderr, "設定の置き場を作れません:", err)
		return 1
	}
	if lf, err := logfile.Open(cfgDir); err == nil {
		// 画面の無い GUI 版では標準エラーへ書けないので、ファイルを先に置く
		// (MultiWriter は失敗した所で止まる)。
		log.SetOutput(io.MultiWriter(lf, os.Stderr))
	}
	log.Printf("起動丸エージェント %s を起動します", version)
	dir, err := paths.ActionsDir()
	if err != nil {
		log.Printf("ホームフォルダが分かりません: %v", err)
		return 1
	}
	if wrote, err := defaults.Write(dir, runtime.GOOS); err != nil {
		log.Printf("操作フォルダを用意できません: %v", err)
	} else if wrote {
		log.Printf("操作フォルダを作りました: %s", dir)
	}
	kp, _ := paths.KeyFile()
	key, err := auth.LoadKey(kp)
	if err != nil {
		log.Printf("%v。組み直してください(kido-agent pair)", err)
	}
	ag := newAgent(kp, dir, runtime.GOOS, version, key)
	// Windows で昇格して起動されていたら、普通のユーザーの権限で起動し直す
	// (起動時タスクは管理者で動くため)。起動し直したら親はここで終わる。
	if dropPrivilege() {
		return 0
	}
	return runRole(ag)
}

// runRole は役を決める。手元の窓口を取れたら待ち受け役(全機能)。取れず、既存の
// 待ち受け役が応えるなら手足役(ログイン中の画面で実行する側)。待ち受け役が
// 消えたら、また窓口を取りに行って待ち受け役になろうとする。
func runRole(a *agent) int {
	for {
		ctl, err := net.Listen("tcp", paths.ControlAddr)
		if err == nil {
			return runListener(a, ctl)
		}
		if _, e := control.Call("GET", "/control/status"); e == nil {
			log.Printf("手足役として動きます(別のプロセスが待ち受けています)")
			runHelper()
			// 待ち受け役が消えた。少し待ってから、自分が待ち受け役になろうとする。
			time.Sleep(time.Second)
			continue
		}
		log.Printf("もう動いているようです(%v)", err)
		return 1
	}
}

// runListener は待ち受け役として、本体の窓口と手元の窓口を開いて待つ。
func runListener(a *agent, ctl net.Listener) int {
	// 窓口を取れた側が待ち受け役。合言葉を書き、CLI と手足役が読めるようにする。
	tok, err := control.WriteToken()
	if err != nil {
		log.Printf("合言葉を書けません: %v", err)
		return 1
	}
	a.token = tok
	api, err := net.Listen("tcp", paths.APIAddr)
	if err != nil {
		log.Printf("%s で待ち受けられません: %v", paths.APIAddr, err)
		return 1
	}
	go newServer(a.controlHandler()).Serve(ctl)
	go newServer(a.apiHandler()).Serve(api)
	log.Printf("待ち受けています(%s)", paths.APIAddr)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {
	case <-a.stopRequest:
		log.Printf("止める要求を受けました")
		time.Sleep(200 * time.Millisecond) // 止める要求への返事を送り切るため
	case s := <-sig:
		log.Printf("止めます(%v)", s)
	}
	return 0
}

// newServer は本体の HTTP/1.0 に合わせ、1 要求ごとに接続を閉じる。
// 時間切れは、途中で黙った相手に接続を握られ続けないようにするため。
func newServer(h http.Handler) *http.Server {
	s := &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		MaxHeaderBytes:    8 << 10,
		ErrorLog:          log.New(io.Discard, "", 0),
	}
	s.SetKeepAlivesEnabled(false)
	return s
}
