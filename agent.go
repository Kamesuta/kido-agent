package main

import (
	"log"
	"sync"
	"time"
)

const pairWindow = 10 * time.Minute

// 状態の名前は本体との取り決め(PROTOCOL.md)どおり。
const (
	stateUnpaired = "unpaired"
	statePairing  = "pairing"
	statePaired   = "paired"
)

// agent は常駐アプリの中心。鍵・組める時間・nonce・操作フォルダを持つ。
// 時刻と実行はテストで差し替えられるよう、関数として持つ。
type agent struct {
	mu          sync.Mutex
	key         []byte
	pairUntil   time.Time
	keyPath     string
	actionsDir  string
	goos        string
	now         func() time.Time
	nonces      *nonceStore
	launch      func(a action) error
	runDelay    time.Duration
	after       func(d time.Duration, f func())
	logf        func(format string, args ...any)
	stopRequest chan struct{}
}

func newAgent(keyPath, actionsDir, goos string, key []byte) *agent {
	a := &agent{
		key:         key,
		keyPath:     keyPath,
		actionsDir:  actionsDir,
		goos:        goos,
		now:         time.Now,
		launch:      launchAction,
		runDelay:    300 * time.Millisecond,
		after:       func(d time.Duration, f func()) { time.AfterFunc(d, f) },
		logf:        log.Printf,
		stopRequest: make(chan struct{}, 1),
	}
	// nonce の失効もテストで差し替えた時計に従わせる。
	a.nonces = newNonceStore(func() time.Time { return a.now() })
	return a
}

// stateLocked は組める時間の中なら鍵の有無にかかわらず pairing を返す。
// 組める時間を開いた時点で古い鍵は消しているので、両方が同時に立つことは無い。
func (a *agent) stateLocked() string {
	if a.now().Before(a.pairUntil) {
		return statePairing
	}
	if a.key != nil {
		return statePaired
	}
	return stateUnpaired
}

func (a *agent) state() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.stateLocked()
}

// currentKey は paired のときだけ鍵を返す。
func (a *agent) currentKey() []byte {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stateLocked() != statePaired {
		return nil
	}
	return a.key
}

// openPairing は組める時間を開く。組み直しの途中で古い鍵が生き残ると、
// 手放した本体(や盗まれた鍵)からまだ操作できてしまうので、先に消す。
func (a *agent) openPairing() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := removeKey(a.keyPath); err != nil {
		return err
	}
	a.key = nil
	a.pairUntil = a.now().Add(pairWindow)
	return nil
}

// acceptKey は組める時間の中でだけ鍵を受け取る。受け取ったら時間を閉じ、
// 同じ時間の中で別の誰かに鍵を上書きされないようにする。
func (a *agent) acceptKey(key []byte) (ok bool, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stateLocked() != statePairing {
		return false, nil
	}
	if err := saveKey(a.keyPath, key); err != nil {
		return false, err
	}
	a.key = key
	a.pairUntil = time.Time{}
	return true, nil
}

// remaining は組める時間の残り(秒)。閉じていれば 0。
func (a *agent) remaining() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	d := a.pairUntil.Sub(a.now())
	if d <= 0 {
		return 0
	}
	return int((d + time.Second - 1) / time.Second)
}
