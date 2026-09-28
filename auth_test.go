package main

import (
	"strings"
	"testing"
	"time"
)

func TestSignKnownVector(t *testing.T) {
	// RFC 4231 の試験 2(key="Jefe")。"\n" でつなぐ前の 1 部品なら HMAC そのもの。
	got := sign([]byte("Jefe"), "what do ya want for nothing?")
	want := "5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843"
	if got != want {
		t.Fatalf("sign = %s", got)
	}
	if sign(testKey, "list", "n") != sign(testKey, "list\nn") {
		t.Fatal("部品は \\n でつなぐ")
	}
}

func TestVerify(t *testing.T) {
	s := sign(testKey, "run", "n", "id")
	if !verify(testKey, s, "run", "n", "id") {
		t.Fatal("正しい署名が通らない")
	}
	if verify(testKey, strings.ToUpper(s), "run", "n", "id") {
		t.Fatal("大文字の 16 進は取り決めと違う")
	}
	if verify(nil, s, "run", "n", "id") || verify(testKey, s, "run", "n", "other") {
		t.Fatal("鍵なし・中身違いが通った")
	}
}

func TestNonceSingleUse(t *testing.T) {
	now := time.Unix(0, 0)
	s := newNonceStore(func() time.Time { return now })
	n, _ := s.issue()
	if len(n) != 32 {
		t.Fatalf("nonce は 32 文字: %q", n)
	}
	if !s.take(n) || s.take(n) {
		t.Fatal("1 回目だけ通るはず")
	}
	if s.take("unknown") {
		t.Fatal("出していない nonce が通った")
	}
}

func TestNonceExpires(t *testing.T) {
	now := time.Unix(0, 0)
	s := newNonceStore(func() time.Time { return now })
	a, _ := s.issue()
	b, _ := s.issue()
	now = now.Add(29 * time.Second)
	if !s.take(a) {
		t.Fatal("29 秒ではまだ生きている")
	}
	now = now.Add(time.Second)
	if s.take(b) {
		t.Fatal("30 秒で失効する")
	}
}

func TestNonceCap(t *testing.T) {
	now := time.Unix(0, 0)
	s := newNonceStore(func() time.Time { return now })
	var all []string
	for i := 0; i < nonceMax+2; i++ {
		n, _ := s.issue()
		all = append(all, n)
	}
	if s.take(all[0]) || s.take(all[1]) {
		t.Fatal("上限を超えたら古いものから捨てる")
	}
	for _, n := range all[2:] {
		if !s.take(n) {
			t.Fatal("新しい 32 個は残る")
		}
	}
}
