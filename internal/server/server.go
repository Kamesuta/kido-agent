// Package server は常駐アプリの本体。起動丸の本体から届く要求と、CLI からの要求を受ける。
package server

import (
	"context"
	"log"
	"time"

	"kido-agent/internal/actions"
	"kido-agent/internal/auth"
	"kido-agent/internal/launch"
)

// agent は鍵・組める時間・nonce・操作フォルダを束ねる。
// 時刻と実行はテストで差し替えられるよう、関数として持つ。
type agent struct {
	pairing     *auth.Pairing
	nonces      *auth.NonceStore
	hub         *helperHub
	actionsDir  string
	goos        string
	version     string
	token       string // 手元の窓口の合言葉(空なら誰も通さない)
	now         func() time.Time
	launch      func(a actions.Action) error
	launchWait  func(a actions.Action) (int, error) // 終わるまで待ち、終了コードを返す
	runDelay    time.Duration
	after       func(d time.Duration, f func())
	logf        func(format string, args ...any)
	stopRequest chan struct{}
}

func newAgent(keyPath, actionsDir, goos, version string, key []byte) *agent {
	a := &agent{
		actionsDir:  actionsDir,
		goos:        goos,
		version:     version,
		now:         time.Now,
		launch:      func(x actions.Action) error { return launch.Run(x.Dir, x.Run) },
		launchWait:  func(x actions.Action) (int, error) { return launch.RunWait(x.Dir, x.Run) },
		runDelay:    300 * time.Millisecond,
		after:       func(d time.Duration, f func()) { time.AfterFunc(d, f) },
		logf:        log.Printf,
		stopRequest: make(chan struct{}, 1),
	}
	// 組める時間と nonce の失効、手足役がいるかの判断も、テストで差し替えた時計に従わせる。
	clock := func() time.Time { return a.now() }
	a.pairing = auth.NewPairing(keyPath, key, clock)
	a.nonces = auth.NewNonceStore(clock)
	a.hub = newHelperHub(clock)
	return a
}

// sessionActive は誰かがログインしている(手足役がいる)とみなせるか。
func (a *agent) sessionActive() bool { return a.hub.active() }

// execute は操作を動かす。手足役がいればそちらへ流し(ログイン中の見た目・窓で動く)、
// いなければ待ち受け役が自分で画面なしに動かす(require_login でない操作だけここに来る)。
func (a *agent) execute(t actions.Action, viaHelper bool) {
	if viaHelper {
		if _, err := a.hub.dispatch(context.Background(), t.ID, t.Dir, t.Run, false); err != nil {
			a.logf("手足役での実行に失敗しました: %s: %v", t.ID, err)
		}
		return
	}
	if err := a.launch(t); err != nil {
		a.logf("実行に失敗しました: %s: %v", t.ID, err)
	}
}

// executeWait は execute の wait 版。終わるまで待って終了コードを返す。
// ctx は手足役に流したときだけ効く(自分で動かした子は、相手が去っても最後まで待って回収する)。
func (a *agent) executeWait(ctx context.Context, t actions.Action, viaHelper bool) (int, error) {
	if viaHelper {
		return a.hub.dispatch(ctx, t.ID, t.Dir, t.Run, true)
	}
	return a.launchWait(t)
}
