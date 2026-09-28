package main

import (
	"os"
	"testing"
	"time"
)

func TestPairWindow(t *testing.T) {
	ta := newTestAgent(t)
	if ta.state() != stateUnpaired {
		t.Fatal("最初は unpaired")
	}
	if ok, _ := ta.acceptKey(testKey); ok {
		t.Fatal("組める時間の外で鍵を受け取った")
	}
	ta.openPairing()
	if ta.state() != statePairing || ta.remaining() != 600 {
		t.Fatalf("state=%s remaining=%d", ta.state(), ta.remaining())
	}
	if ok, err := ta.acceptKey(testKey); !ok || err != nil {
		t.Fatalf("組めない: %v", err)
	}
	if ta.state() != statePaired || ta.remaining() != 0 {
		t.Fatal("組んだら時間を閉じて paired")
	}
	if ok, _ := ta.acceptKey([]byte("other-key-other-key-other-key-00")); ok {
		t.Fatal("組んだ後に鍵を上書きできてしまう")
	}
	if key, _ := loadKey(ta.keyPath); string(key) != string(testKey) {
		t.Fatal("鍵がファイルに残っていない")
	}
}

func TestPairWindowExpires(t *testing.T) {
	ta := newTestAgent(t)
	ta.openPairing()
	ta.advance(10*time.Minute - time.Second)
	if ta.state() != statePairing {
		t.Fatal("10 分までは開いている")
	}
	ta.advance(time.Second)
	if ta.state() != stateUnpaired {
		t.Fatal("10 分で閉じる")
	}
}

func TestRepairDiscardsOldKey(t *testing.T) {
	ta := newTestAgent(t)
	ta.openPairing()
	ta.acceptKey(testKey)
	ta.openPairing()
	if ta.currentKey() != nil {
		t.Fatal("組み直しを始めたら古い鍵は使えない")
	}
	if _, err := os.Stat(ta.keyPath); !os.IsNotExist(err) {
		t.Fatal("古い鍵のファイルが残っている")
	}
	ta.advance(pairWindow)
	if ta.state() != stateUnpaired {
		t.Fatal("組まずに時間切れなら unpaired に戻る")
	}
}

func TestKeyFileRoundTrip(t *testing.T) {
	p := t.TempDir() + "/sub/key.json"
	if k, err := loadKey(p); k != nil || err != nil {
		t.Fatal("無ければ nil")
	}
	if err := saveKey(p, testKey); err != nil {
		t.Fatal(err)
	}
	k, err := loadKey(p)
	if err != nil || string(k) != string(testKey) {
		t.Fatalf("読み戻せない: %v", err)
	}
	writeFile(t, p, `{"key":"zz"}`)
	if _, err := loadKey(p); err == nil {
		t.Fatal("壊れた鍵を受け入れた")
	}
}
