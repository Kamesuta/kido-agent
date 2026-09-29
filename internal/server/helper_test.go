package server

import (
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
	go func() { done <- h.dispatch("10_sleep", "/dir", "sleep.ps1") }()
	// 手足役が取りに来る
	j := h.poll()
	if j == nil || j.ID != "10_sleep" || j.Run != "sleep.ps1" {
		t.Fatalf("仕事が渡らない: %+v", j)
	}
	if !h.active() {
		t.Fatal("取りに来たのに active でない")
	}
	h.complete("10_sleep", "") // 成功を返す
	if err := <-done; err != nil {
		t.Fatalf("成功のはず: %v", err)
	}
}

func TestHubDispatchNoHelper(t *testing.T) {
	h := fastHub()
	// 誰も取りに来ない → dispatchGrace で errNoHelper
	if err := h.dispatch("x", "/d", "r"); err != errNoHelper {
		t.Fatalf("手足役なしのはず: %v", err)
	}
}

func TestHubResultError(t *testing.T) {
	h := fastHub()
	done := make(chan error, 1)
	go func() { done <- h.dispatch("x", "/d", "r") }()
	h.poll()
	h.complete("x", "起動に失敗")
	if err := <-done; err == nil || err.Error() != "起動に失敗" {
		t.Fatalf("エラーが返るはず: %v", err)
	}
}

func TestHubPollTimeoutEmpty(t *testing.T) {
	h := fastHub()
	start := time.Now()
	if j := h.poll(); j != nil {
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
