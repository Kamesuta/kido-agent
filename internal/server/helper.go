package server

import (
	"errors"
	"sync"
	"time"
)

// helperHub は待ち受け役の中で、手足役(ログイン中の画面を持つ別プロセス)との
// やり取りを束ねる。手足役はロングポーリングでつなぎっぱなしにし、待ち受け役は
// 「画面のいる実行」をここに流して、手足役がログイン中の見た目・窓で動かす。
type helperHub struct {
	mu       sync.Mutex
	pending  chan *helperJob       // 待ち受け役 → 手足役に渡す仕事
	waiting  map[string]*helperJob // 渡して結果待ちの仕事(id で引く)
	lastPoll time.Time             // 手足役が最後に取りに来た時刻(いるかの目安)
	now      func() time.Time

	sessionGrace  time.Duration // これを過ぎて取りに来なければ「いない」
	dispatchGrace time.Duration // 手足役が仕事を拾うまで待つ上限
	resultGrace   time.Duration // 実行の結果を待つ上限
	pollHold      time.Duration // 1 回のロングポーリングを保つ長さ
}

// helperJob は手足役に動かしてもらう 1 件。
type helperJob struct {
	ID     string      `json:"id"`
	Dir    string      `json:"dir"`
	Run    string      `json:"run"`
	result chan string // 空文字は成功、それ以外はエラーの説明
}

func newHelperHub(now func() time.Time) *helperHub {
	return &helperHub{
		pending:       make(chan *helperJob),
		waiting:       map[string]*helperJob{},
		now:           now,
		sessionGrace:  90 * time.Second,
		dispatchGrace: 3 * time.Second,
		resultGrace:   60 * time.Second,
		pollHold:      30 * time.Second,
	}
}

var errNoHelper = errors.New("手足役がいません")

// active は手足役がいる(誰かがログインしている)とみなせるか。
// 手足役は 30 秒ごとに取りに来て、切れたら来なくなるので、少し猶予を見て判断する。
func (h *helperHub) active() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return !h.lastPoll.IsZero() && h.now().Sub(h.lastPoll) < h.sessionGrace
}

// poll は手足役の「次の仕事をください」に応える。仕事が来れば渡し、来なければ
// pollHold で空応答を返す(手足役はすぐつなぎ直す)。取りに来たこと自体を控える。
func (h *helperHub) poll() *helperJob {
	h.mu.Lock()
	h.lastPoll = h.now()
	h.mu.Unlock()
	select {
	case j := <-h.pending:
		h.mu.Lock()
		h.waiting[j.ID] = j
		h.lastPoll = h.now()
		h.mu.Unlock()
		return j
	case <-time.After(h.pollHold):
		return nil
	}
}

// complete は手足役から返ってきた結果を、待っている dispatch に渡す。
func (h *helperHub) complete(id, errMsg string) {
	h.mu.Lock()
	j := h.waiting[id]
	delete(h.waiting, id)
	h.mu.Unlock()
	if j != nil {
		select {
		case j.result <- errMsg:
		default:
		}
	}
}

// dispatch は 1 件を手足役に流し、実行の結果を待つ。手足役が拾わない・結果が
// 返らないときは、その旨のエラーを返す。
func (h *helperHub) dispatch(id, dir, run string) error {
	j := &helperJob{ID: id, Dir: dir, Run: run, result: make(chan string, 1)}
	select {
	case h.pending <- j:
	case <-time.After(h.dispatchGrace):
		return errNoHelper
	}
	select {
	case msg := <-j.result:
		if msg != "" {
			return errors.New(msg)
		}
		return nil
	case <-time.After(h.resultGrace):
		h.mu.Lock()
		delete(h.waiting, id)
		h.mu.Unlock()
		return errors.New("手足役からの結果が返りませんでした")
	}
}
