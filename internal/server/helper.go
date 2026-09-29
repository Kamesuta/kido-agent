package server

import (
	"context"
	"errors"
	"fmt"
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
	seq      int                   // 仕事の番号の元
	now      func() time.Time

	sessionGrace  time.Duration // これを過ぎて取りに来なければ「いない」
	dispatchGrace time.Duration // 手足役が仕事を拾うまで待つ上限
	resultGrace   time.Duration // 実行の結果を待つ上限
	pollHold      time.Duration // 1 回のロングポーリングを保つ長さ
}

// helperJob は手足役に動かしてもらう 1 件。ID は「操作の ID#番号」で 1 件ごとに違う。
// wait の操作を待っている間に同じ操作がまた押されても、結果を取り違えないため。
type helperJob struct {
	ID     string `json:"id"`
	Dir    string `json:"dir"`
	Run    string `json:"run"`
	Wait   bool   `json:"wait"` // 終わるまで待って終了コードを返してほしいか
	result chan helperResult
}

// helperResult は手足役から返った結果。Err が空なら動かせた(wait なら Code が終了コード)。
type helperResult struct {
	Code int
	Err  string
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
// 待っている側がもういない(相手が接続を切った)なら、結果は捨てる。
func (h *helperHub) complete(id string, res helperResult) {
	h.mu.Lock()
	j := h.waiting[id]
	delete(h.waiting, id)
	h.mu.Unlock()
	if j != nil {
		select {
		case j.result <- res:
		default:
		}
	}
}

func (h *helperHub) forget(id string) {
	h.mu.Lock()
	delete(h.waiting, id)
	h.mu.Unlock()
}

// dispatch は 1 件を手足役に流し、結果を待つ。wait でなければ「動かせたか」だけを
// resultGrace まで待つ。wait なら終わるまで待ち、時間切れは持たない(長さの見積もりは
// スクリプトを書いた人に任せる)。どちらも ctx が終われば(相手が切った)待つのをやめる。
func (h *helperHub) dispatch(ctx context.Context, action, dir, run string, wait bool) (int, error) {
	h.mu.Lock()
	h.seq++
	j := &helperJob{ID: fmt.Sprintf("%s#%d", action, h.seq), Dir: dir, Run: run, Wait: wait,
		result: make(chan helperResult, 1)}
	h.mu.Unlock()
	select {
	case h.pending <- j:
	case <-time.After(h.dispatchGrace):
		return 0, errNoHelper
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	var limit <-chan time.Time // nil のままなら時間切れにならない
	if !wait {
		limit = time.After(h.resultGrace)
	}
	select {
	case r := <-j.result:
		if r.Err != "" {
			return 0, errors.New(r.Err)
		}
		return r.Code, nil
	case <-limit:
		h.forget(j.ID)
		return 0, errors.New("手足役からの結果が返りませんでした")
	case <-ctx.Done():
		h.forget(j.ID)
		return 0, ctx.Err()
	}
}
