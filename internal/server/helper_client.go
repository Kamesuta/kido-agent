package server

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kido-agent/internal/actions"
	"kido-agent/internal/control"
	"kido-agent/internal/launch"
)

// runHelper は手足役として、待ち受け役にロングポーリングでつなぎっぱなしにする。
// 仕事が来たら、ログイン中のこの画面で今までどおりに動かして、結果を返す。
// 待ち受け役に届かない状態が続いたら(たぶん消えた)戻る。呼び出し側が
// 自分を待ち受け役にしようとする。dir は自分の操作フォルダで、動かすものはここから引く。
func runHelper(dir, goos string) {
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
		var job helperJob
		if json.Unmarshal(data, &job) != nil || job.ID == "" {
			continue // 空応答。つなぎ直す
		}
		// 別の流れで動かし、すぐ次を取りに行く。wait の仕事を待っている間も取りに
		// 行き続けないと、待ち受け役に「ログインしていない」と見なされてしまう。
		go runJob(dir, goos, job)
	}
}

// runJob は 1 件を動かして、結果を待ち受け役へ返す。
func runJob(dir, goos string, job helperJob) {
	code, err := execJob(dir, goos, job, launch.Run, launch.RunWait)
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	log.Printf("手足役: 実行しました %s code=%d err=%q", job.ID, code, errMsg)
	res, _ := json.Marshal(map[string]any{"id": job.ID, "ok": errMsg == "", "error": errMsg, "code": code})
	control.Do("POST", "/control/helper/result", res, 5*time.Second)
}

// execJob は仕事の操作 ID を自分の操作フォルダで引き直して動かす。待ち受け役から
// 届いた中身は ID と設定の突き合わせにしか使わない(何を動かすかは自分で決める)。
// 実行の関数はテストで差し替えられるよう引数で受け取る。
func execJob(dir, goos string, job helperJob,
	run func(dir, file string) error, runWait func(dir, file string) (int, error)) (int, error) {
	t, found := actions.Find(dir, goos, job.Action)
	switch {
	case !found:
		return 0, errors.New("操作が見つかりません: " + job.Action)
	case t.Broken:
		return 0, errors.New("押せない操作です: " + job.Action)
	case t.Wait != job.Wait || t.RequireLogin != job.RequireLogin:
		// 待ち受け役と手足役で見ているフォルダが違う(別の利用者の窓口など)か、
		// 押した後に kido.toml が書き換わった。どちらの意図か分からないので動かさない。
		return 0, errors.New("待ち受け役と操作の設定が食い違っています: " + job.Action)
	}
	if t.Wait {
		return runWait(t.Dir, t.Run)
	}
	return 0, run(t.Dir, t.Run)
}
