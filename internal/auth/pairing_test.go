package auth

import (
	"os"
	"testing"
	"time"
)

// testPairing は時計を進められる Pairing。10 分を実際に待たないため。
type testPairing struct {
	*Pairing
	clock time.Time
}

func newTestPairing(t *testing.T) *testPairing {
	tp := &testPairing{clock: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	tp.Pairing = NewPairing(t.TempDir()+"/cfg/key.json", nil, func() time.Time { return tp.clock })
	return tp
}

func (tp *testPairing) advance(d time.Duration) { tp.clock = tp.clock.Add(d) }

func TestPairWindow(t *testing.T) {
	ta := newTestPairing(t)
	if ta.State() != StateUnpaired {
		t.Fatal("最初は unpaired")
	}
	if ok, _ := ta.Accept(testKey); ok {
		t.Fatal("組める時間の外で鍵を受け取った")
	}
	ta.Open()
	if ta.State() != StatePairing || ta.Remaining() != 600 {
		t.Fatalf("state=%s remaining=%d", ta.State(), ta.Remaining())
	}
	if ok, err := ta.Accept(testKey); !ok || err != nil {
		t.Fatalf("組めない: %v", err)
	}
	if ta.State() != StatePaired || ta.Remaining() != 0 {
		t.Fatal("組んだら時間を閉じて paired")
	}
	if ok, _ := ta.Accept([]byte("other-key-other-key-other-key-00")); ok {
		t.Fatal("組んだ後に鍵を上書きできてしまう")
	}
	if key, _ := LoadKey(ta.keyPath); string(key) != string(testKey) {
		t.Fatal("鍵がファイルに残っていない")
	}
}

func TestPairWindowExpires(t *testing.T) {
	ta := newTestPairing(t)
	ta.Open()
	ta.advance(10*time.Minute - time.Second)
	if ta.State() != StatePairing {
		t.Fatal("10 分までは開いている")
	}
	ta.advance(time.Second)
	if ta.State() != StateUnpaired {
		t.Fatal("10 分で閉じる")
	}
}

func TestRepairDiscardsOldKey(t *testing.T) {
	ta := newTestPairing(t)
	ta.Open()
	ta.Accept(testKey)
	ta.Open()
	if ta.Key() != nil {
		t.Fatal("組み直しを始めたら古い鍵は使えない")
	}
	if _, err := os.Stat(ta.keyPath); !os.IsNotExist(err) {
		t.Fatal("古い鍵のファイルが残っている")
	}
	ta.advance(PairWindow)
	if ta.State() != StateUnpaired {
		t.Fatal("組まずに時間切れなら unpaired に戻る")
	}
}

func TestKeyFileRoundTrip(t *testing.T) {
	p := t.TempDir() + "/sub/key.json"
	if k, err := LoadKey(p); k != nil || err != nil {
		t.Fatal("無ければ nil")
	}
	if err := SaveKey(p, testKey); err != nil {
		t.Fatal(err)
	}
	k, err := LoadKey(p)
	if err != nil || string(k) != string(testKey) {
		t.Fatalf("読み戻せない: %v", err)
	}
	os.WriteFile(p, []byte(`{"key":"zz"}`), 0o600)
	if _, err := LoadKey(p); err == nil {
		t.Fatal("壊れた鍵を受け入れた")
	}
}
