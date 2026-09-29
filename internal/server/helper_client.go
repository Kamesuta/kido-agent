package server

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kido-agent/internal/control"
	"kido-agent/internal/launch"
)

// runHelper は手足役として、待ち受け役にロングポーリングでつなぎっぱなしにする。
// 仕事が来たら、ログイン中のこの画面で今までどおりに動かして、結果を返す。
// 待ち受け役に届かない状態が続いたら(たぶん消えた)戻る。呼び出し側が
// 自分を待ち受け役にしようとする。
func runHelper() {
	log.Printf("手足役: ログイン中の画面で操作を動かします")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	fails := 0
	for {
		select {
		case s := <-stop:
			log.Printf("手足役を止めます(%v)", s)
			return
		default:
		}
		// pollHold(待ち受け役側 30 秒)より長めに待つ。空応答はすぐ返る。
		data, err := control.Do("GET", "/control/helper/next", nil, 40*time.Second)
		if err != nil {
			fails++
			if fails > 15 { // 30 秒ほど届かなければ、待ち受け役は消えたとみなす
				log.Printf("待ち受け役に届きません。役を取り直します")
				return
			}
			time.Sleep(2 * time.Second)
			continue
		}
		fails = 0
		var job struct {
			ID, Dir, Run string
			Wait         bool
		}
		if json.Unmarshal(data, &job) != nil || job.ID == "" {
			continue // 空応答。つなぎ直す
		}
		// 別の流れで動かし、すぐ次を取りに行く。wait の仕事を待っている間も取りに
		// 行き続けないと、待ち受け役に「ログインしていない」と見なされてしまう。
		go runJob(job.ID, job.Dir, job.Run, job.Wait)
	}
}

// runJob は 1 件を動かして、結果を待ち受け役へ返す。
func runJob(id, dir, run string, wait bool) {
	code, err := 0, error(nil)
	if wait {
		code, err = launch.RunWait(dir, run)
	} else {
		err = launch.Run(dir, run)
	}
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	log.Printf("手足役: 実行しました %s (%s) code=%d err=%q", id, run, code, errMsg)
	res, _ := json.Marshal(map[string]any{"id": id, "ok": errMsg == "", "error": errMsg, "code": code})
	control.Do("POST", "/control/helper/result", res, 5*time.Second)
}
