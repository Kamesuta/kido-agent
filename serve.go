package main

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
)

func cmdServe() int {
	cfgDir, err := configDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "設定の置き場が分かりません:", err)
		return 1
	}
	if lf, err := openLog(cfgDir); err == nil {
		// 画面の無い GUI 版では標準エラーへ書けないので、ファイルを先に置く
		// (MultiWriter は失敗した所で止まる)。
		log.SetOutput(io.MultiWriter(lf, os.Stderr))
	}
	log.Printf("起動丸エージェント %s を起動します", version)
	dir, err := actionsDir()
	if err != nil {
		log.Printf("ホームフォルダが分かりません: %v", err)
		return 1
	}
	if wrote, err := writeDefaults(dir, runtime.GOOS); err != nil {
		log.Printf("操作フォルダを用意できません: %v", err)
	} else if wrote {
		log.Printf("操作フォルダを作りました: %s", dir)
	}
	kp, _ := keyPath()
	key, err := loadKey(kp)
	if err != nil {
		log.Printf("%v。組み直してください(kido-agent pair)", err)
	}
	return serveAgent(newAgent(kp, dir, runtime.GOOS, key))
}

func serveAgent(a *agent) int {
	// 操作用の窓口を先に取る。取れなければ、もう 1 つ動いているので黙って引き下がる。
	ctl, err := net.Listen("tcp", controlAddr)
	if err != nil {
		log.Printf("もう動いているようです(%v)", err)
		return 1
	}
	api, err := net.Listen("tcp", apiAddr)
	if err != nil {
		log.Printf("%s で待ち受けられません: %v", apiAddr, err)
		return 1
	}
	go newServer(a.controlHandler()).Serve(ctl)
	go newServer(a.apiHandler()).Serve(api)
	log.Printf("待ち受けています(%s)", apiAddr)

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
