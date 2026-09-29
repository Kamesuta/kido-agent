package server

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// serveControl は手元の窓口を本物の接続で開く。時間切れを短くして、
// ロングポーリングがそれより長く保たれても仕事を渡せることを確かめる。
func serveControl(t *testing.T, ta *testAgent) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := newServer(ta.controlHandler())
	s.ReadTimeout, s.WriteTimeout = 100*time.Millisecond, 100*time.Millisecond
	go s.Serve(ln)
	t.Cleanup(func() { s.Close() })
	return "http://" + ln.Addr().String()
}

func pollOnce(ctx context.Context, base string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", base+"/control/helper/next", nil)
	req.Header.Set("X-Kido-Control", "test-token")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	return string(b), err
}

// 仕事が WriteTimeout より後に来ても、つなぎっぱなしの手足役に届く。
func TestHelperPollOutlivesWriteTimeout(t *testing.T) {
	ta := pairedAgent(t)
	ta.hub.pollHold = time.Second
	base := serveControl(t, ta)
	got := make(chan string, 1)
	go func() { b, _ := pollOnce(context.Background(), base); got <- b }()
	time.Sleep(300 * time.Millisecond) // 窓口の時間切れ(100ms)を過ぎてから流す
	go ta.hub.dispatch(context.Background(), "10_sleep", "/d", "s.ps1", false)
	select {
	case b := <-got:
		var j helperJob
		if json.Unmarshal([]byte(b), &j) != nil || !strings.HasPrefix(j.ID, "10_sleep#") {
			t.Fatalf("仕事が届かない: %q", b)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("返事が来ない")
	}
}

// 手足役が去った(接続が切れた)ロングポーリングは、仕事を横取りしない。
func TestHelperPollGoneDoesNotTakeJob(t *testing.T) {
	ta := pairedAgent(t)
	ta.hub.pollHold = 2 * time.Second
	ta.hub.dispatchGrace = 300 * time.Millisecond
	base := serveControl(t, ta)
	ctx, cancel := context.WithCancel(context.Background())
	go pollOnce(ctx, base)
	time.Sleep(100 * time.Millisecond)
	cancel() // 手足役が消えた
	time.Sleep(100 * time.Millisecond)
	wctx, stop := context.WithTimeout(context.Background(), 2*time.Second)
	defer stop()
	if _, err := ta.hub.dispatch(wctx, "x", "/d", "r", true); err != errNoHelper {
		t.Fatalf("去った手足役に渡してはいけない: %v", err)
	}
}
