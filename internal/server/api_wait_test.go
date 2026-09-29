package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"kido-agent/internal/actions"
	"kido-agent/internal/auth"
)

// waitAction は kido.toml に wait = true を書いた操作を作る。
func waitAction(t *testing.T, ta *testAgent, id, file string) {
	d := mkAction(t, ta.actionsDir, id, file)
	writeFile(t, d+"/kido.toml", "wait = true\n")
}

func TestRunWaitReportsExitCode(t *testing.T) {
	cases := []struct {
		code int
		err  error
		body string
	}{
		{0, nil, `{"ok":true}`},
		{3, nil, `{"ok":false,"err":"exit","code":3}`},
		{0, errors.New("見つからない"), `{"ok":false,"err":"start"}`},
	}
	for _, c := range cases {
		ta := pairedAgent(t)
		waitAction(t, ta, "90_build", "build.bat")
		ta.launchWait = func(a actions.Action) (int, error) {
			ta.launched = append(ta.launched, a.ID+"/"+a.Run)
			return c.code, c.err
		}
		_, n := ta.hello(t)
		r := call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, "90_build"))
		if r.status != 200 || r.body != c.body || r.sig != auth.Sign(testKey, "run-ok", n, r.body) {
			t.Errorf("code=%d: %+v", c.code, r)
		}
		// 待つ操作は、返事の後に遅らせて動かす道(after)を通らない
		if len(ta.launched) != 1 || len(ta.delays) != 0 {
			t.Errorf("code=%d: launched=%v delays=%v", c.code, ta.launched, ta.delays)
		}
	}
}

func TestRunWithoutWaitDoesNotWait(t *testing.T) {
	ta := pairedAgent(t)
	mkAction(t, ta.actionsDir, "10_sleep", "sleep.ps1")
	ta.launchWait = func(actions.Action) (int, error) { t.Fatal("wait なしで待った"); return 0, nil }
	var scheduled bool
	ta.after = func(time.Duration, func()) { scheduled = true } // 動かすのは返事の後
	_, n := ta.hello(t)
	r := call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, "10_sleep"))
	if r.body != `{"ok":true}` || !scheduled || len(ta.launched) != 0 {
		t.Fatalf("返事が先: %+v scheduled=%v launched=%v", r, scheduled, ta.launched)
	}
}

func TestRunWaitShortcutIsBroken(t *testing.T) {
	ta := pairedAgent(t)
	waitAction(t, ta, "50_game", "game.lnk")
	_, n := ta.hello(t)
	r := call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, "50_game"))
	if r.status != 409 || !strings.Contains(r.body, "broken_action") {
		t.Fatalf("%+v", r)
	}
}

func TestRunWaitViaHelper(t *testing.T) {
	ta := pairedAgent(t)
	ta.setLoggedIn(true)
	waitAction(t, ta, "90_build", "build.bat")
	go func() {
		if j := ta.hub.poll(context.Background()); j != nil && j.Wait {
			ta.hub.complete(j.ID, helperResult{Code: 5})
		}
	}()
	_, n := ta.hello(t)
	r := call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, "90_build"))
	if r.body != `{"ok":false,"err":"exit","code":5}` || len(ta.launched) != 0 {
		t.Fatalf("手足役の終了コードを返す: %+v", r)
	}
}

func TestRunWaitOutlivesServerTimeouts(t *testing.T) {
	ta := pairedAgent(t)
	waitAction(t, ta, "90_build", "build.bat")
	ta.launchWait = func(actions.Action) (int, error) { time.Sleep(400 * time.Millisecond); return 2, nil }
	base := serveReal(t, ta)
	_, n := ta.hello(t)
	body, err := postRun(base, n, "90_build", 5*time.Second)
	if err != nil || body != `{"ok":false,"err":"exit","code":2}` {
		t.Fatalf("%q %v", body, err)
	}
}

func TestRunWaitCallerHangsUp(t *testing.T) {
	ta := pairedAgent(t)
	waitAction(t, ta, "90_build", "build.bat")
	release, finished := make(chan struct{}), make(chan struct{})
	ta.launchWait = func(actions.Action) (int, error) { <-release; return 0, nil }
	logs := make(chan string, 8)
	ta.logf = func(f string, args ...any) { logs <- fmt.Sprintf(f, args...) }
	base := serveReal(t, ta)
	_, n := ta.hello(t)
	if _, err := postRun(base, n, "90_build", 200*time.Millisecond); err == nil {
		t.Fatal("相手の時間切れで切れるはず")
	}
	waitLog(t, logs, "相手が切った")
	// 実行は止めずに最後まで待ち、終わったら後始末して抜ける(漏れない)
	go func() { close(release); waitLog(t, logs, "終わりました"); close(finished) }()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("実行の終わりを回収していない")
	}
}
