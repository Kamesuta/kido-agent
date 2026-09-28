// Package server は常駐アプリの本体。起動丸の本体から届く要求と、CLI からの要求を受ける。
package server

import (
	"log"
	"time"

	"github.com/Kamesuta/kido-agent/internal/actions"
	"github.com/Kamesuta/kido-agent/internal/auth"
	"github.com/Kamesuta/kido-agent/internal/launch"
)

// agent は鍵・組める時間・nonce・操作フォルダを束ねる。
// 時刻と実行はテストで差し替えられるよう、関数として持つ。
type agent struct {
	pairing     *auth.Pairing
	nonces      *auth.NonceStore
	actionsDir  string
	goos        string
	version     string
	now         func() time.Time
	launch      func(a actions.Action) error
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
		runDelay:    300 * time.Millisecond,
		after:       func(d time.Duration, f func()) { time.AfterFunc(d, f) },
		logf:        log.Printf,
		stopRequest: make(chan struct{}, 1),
	}
	// 組める時間と nonce の失効も、テストで差し替えた時計に従わせる。
	clock := func() time.Time { return a.now() }
	a.pairing = auth.NewPairing(keyPath, key, clock)
	a.nonces = auth.NewNonceStore(clock)
	return a
}
