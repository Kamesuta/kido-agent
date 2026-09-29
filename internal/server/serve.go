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
	return serveAgent(ag)
}

func serveAgent(a *agent) int {
	// 操作用の窓口を先に取る。取れなければ、もう 1 つ動いているので黙って引き下がる。
	ctl, err := net.Listen("tcp", paths.ControlAddr)
	if err != nil {
		log.Printf("もう動いているようです(%v)", err)
		return 1
	}
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
