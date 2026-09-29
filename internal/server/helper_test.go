package server

import (
	"context"
	"testing"
	"time"
)

// fastHub は待ち時間を短くしたテスト用のハブ。
func fastHub() *helperHub {
	h := newHelperHub(time.Now)
	h.dispatchGrace = 200 * time.Millisecond
	h.resultGrace = 500 * time.Millisecond
	h.pollHold = 100 * time.Millisecond
	return h
}

func TestHubDispatchToPoller(t *testing.T) {
	h := fastHub()
	done := make(chan error, 1)
	go func() {
		_, err := h.dispatch(context.Background(), "10_sleep", "/dir", "sleep.ps1", false)
		done <- err
	}()
	// 手足役が取りに来る
	j := h.poll(context.Background())
	if j == nil || j.ID != "10_sleep#1" || j.Run != "sleep.ps1" {
		t.Fatalf("仕事が渡らない: %+v", j)
	}
	if !h.active() {
		t.Fatal("取りに来たのに active でない")
	}
	h.complete(j.ID, helperResult{}) // 成功を返す
	if err := <-done; err != nil {
		t.Fatalf("成功のはず: %v", err)
	}
}

func TestHubDispatchNoHelper(t *testing.T) {
	h := fastHub()
	// 誰も取りに来ない → dispatchGrace で errNoHelper
	if _, err := h.dispatch(context.Background(), "x", "/d", "r", false); err != errNoHelper {
		t.Fatalf("手足役なしのはず: %v", err)
	}
}

func TestHubResultError(t *testing.T) {
	h := fastHub()
	done := make(chan error, 1)
	go func() {
		_, err := h.dispatch(context.Background(), "x", "/d", "r", false)
		done <- err
	}()
	j := h.poll(context.Background())
	h.complete(j.ID, helperResult{Err: "起動に失敗"})
	if err := <-done; err == nil || err.Error() != "起動に失敗" {
		t.Fatalf("エラーが返るはず: %v", err)
	}
}

func TestHubPollTimeoutEmpty(t *testing.T) {
	h := fastHub()
	start := time.Now()
	if j := h.poll(context.Background()); j != nil {
		t.Fatal("仕事が無いのに返った")
	}
	if time.Since(start) < h.pollHold {
		t.Fatal("空応答が早すぎる")
	}
}

func TestSessionGrace(t *testing.T) {
	now := time.Unix(1000, 0)
	h := newHelperHub(func() time.Time { return now })
	if h.active() {
		t.Fatal("一度も来ていないのに active")
	}
	h.mu.Lock()
	h.lastPoll = now
	h.mu.Unlock()
	now = now.Add(89 * time.Second)
	if !h.active() {
		t.Fatal("89 秒はまだ active")
	}
	now = now.Add(2 * time.Second)
	if h.active() {
		t.Fatal("90 秒を過ぎたら active でない")
	}
}

// wait の仕事は resultGrace を過ぎても待ち続け、相手が去ったら(ctx)やめて控えを消す。
func TestHubWaitHasNoTimeoutButStopsOnCancel(t *testing.T) {
	h := fastHub()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := h.dispatch(ctx, "x", "/d", "r", true); done <- err }()
	j := h.poll(context.Background())
	select {
	case err := <-done:
		t.Fatalf("wait なのに時間切れになった: %v", err)
	case <-time.After(2 * h.resultGrace):
	}
	cancel()
	if err := <-done; err != context.Canceled {
		t.Fatalf("取り消しで戻るはず: %v", err)
	}
	h.mu.Lock()
	left := len(h.waiting)
	h.mu.Unlock()
	if left != 0 {
		t.Fatal("待ちの控えが残っている")
	}
	h.complete(j.ID, helperResult{Code: 1}) // 後から来た結果は捨てるだけ
}
