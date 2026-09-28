package auth

import (
	"sync"
	"time"
)

// PairWindow は kido-agent pair で開く「組める時間」の長さ。
const PairWindow = 10 * time.Minute

// 状態の名前は本体との取り決め(PROTOCOL.md)どおり。
const (
	StateUnpaired = "unpaired"
	StatePairing  = "pairing"
	StatePaired   = "paired"
)

// Pairing は鍵と組める時間を持つ。時刻はテストで差し替えられるよう関数で受ける。
type Pairing struct {
	mu      sync.Mutex
	key     []byte
	until   time.Time
	keyPath string
	now     func() time.Time
}

func NewPairing(keyPath string, key []byte, now func() time.Time) *Pairing {
	return &Pairing{key: key, keyPath: keyPath, now: now}
}

// stateLocked は組める時間の中なら鍵の有無にかかわらず pairing を返す。
// 組める時間を開いた時点で古い鍵は消しているので、両方が同時に立つことは無い。
func (p *Pairing) stateLocked() string {
	if p.now().Before(p.until) {
		return StatePairing
	}
	if p.key != nil {
		return StatePaired
	}
	return StateUnpaired
}

func (p *Pairing) State() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stateLocked()
}

// Key は paired のときだけ鍵を返す。
func (p *Pairing) Key() []byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stateLocked() != StatePaired {
		return nil
	}
	return p.key
}

// Open は組める時間を開く。組み直しの途中で古い鍵が生き残ると、
// 手放した本体(や盗まれた鍵)からまだ操作できてしまうので、先に消す。
func (p *Pairing) Open() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := RemoveKey(p.keyPath); err != nil {
		return err
	}
	p.key = nil
	p.until = p.now().Add(PairWindow)
	return nil
}

// Accept は組める時間の中でだけ鍵を受け取る。受け取ったら時間を閉じ、
// 同じ時間の中で別の誰かに鍵を上書きされないようにする。
func (p *Pairing) Accept(key []byte) (ok bool, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stateLocked() != StatePairing {
		return false, nil
	}
	if err := SaveKey(p.keyPath, key); err != nil {
		return false, err
	}
	p.key = key
	p.until = time.Time{}
	return true, nil
}

// Remaining は組める時間の残り(秒)。閉じていれば 0。
func (p *Pairing) Remaining() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	d := p.until.Sub(p.now())
	if d <= 0 {
		return 0
	}
	return int((d + time.Second - 1) / time.Second)
}
