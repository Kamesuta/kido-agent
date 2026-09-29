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
		var job struct{ ID, Dir, Run string }
		if json.Unmarshal(data, &job) != nil || job.ID == "" {
			continue // 空応答。つなぎ直す
		}
		errMsg := ""
		if e := launch.Run(job.Dir, job.Run); e != nil {
			errMsg = e.Error()
		}
		log.Printf("手足役: 実行しました %s (%s) err=%q", job.ID, job.Run, errMsg)
		res, _ := json.Marshal(map[string]any{"id": job.ID, "ok": errMsg == "", "error": errMsg})
		control.Do("POST", "/control/helper/result", res, 5*time.Second)
	}
}
