package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"kido-agent/internal/actions"
)

// loggedOutHelper は、手足役が「いる」とみなされる猶予の中だが、もう取りに来ない状態にする
// (ログアウトの直後)。
func loggedOutHelper(ta *testAgent) {
	ta.setLoggedIn(true)
	ta.hub.dispatchGrace = 50 * time.Millisecond
}

func TestExecuteFallsBackWhenHelperGone(t *testing.T) {
	ta := pairedAgent(t)
	loggedOutHelper(ta)
	mkAction(t, ta.actionsDir, "60_server", "start.bat")
	_, n := ta.hello(t)
	if r := call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, "60_server")); r.status != 200 {
		t.Fatalf("%+v", r)
	}
	if len(ta.launched) != 1 || ta.launched[0] != "60_server/start.bat" {
		t.Fatalf("手足役がいなければ待ち受け役が動かす: %v", ta.launched)
	}
}

func TestExecuteNoFallbackForRequireLogin(t *testing.T) {
	ta := pairedAgent(t)
	loggedOutHelper(ta)
	d := mkAction(t, ta.actionsDir, "20_lock", "lock.ps1")
	writeFile(t, d+"/kido.toml", "require_login = true\n")
	_, n := ta.hello(t)
	if r := call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, "20_lock")); r.status != 200 {
		t.Fatalf("%+v", r)
	}
	if len(ta.launched) != 0 {
		t.Fatalf("require_login は画面のないところで動かさない: %v", ta.launched)
	}
	// OS の決まりでログイン中(Mac・Linux)なら、待ち受け役が動かしてよい
	ta.osSession = func() bool { return true }
	_, n = ta.hello(t)
	call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, "20_lock"))
	if len(ta.launched) != 1 {
		t.Fatalf("OS がログイン中なら自分で動かす: %v", ta.launched)
	}
}

func TestExecuteWaitFallsBackWhenHelperGone(t *testing.T) {
	ta := pairedAgent(t)
	loggedOutHelper(ta)
	waitAction(t, ta, "90_build", "build.bat")
	ta.launchWait = func(a actions.Action) (int, error) {
		ta.launched = append(ta.launched, a.ID)
		return 4, nil
	}
	_, n := ta.hello(t)
	r := call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, "90_build"))
	if r.body != `{"ok":false,"err":"exit","code":4}` || len(ta.launched) != 1 {
		t.Fatalf("手足役がいなければ待ち受け役が待って動かす: %+v %v", r, ta.launched)
	}

	// require_login の wait は今までどおり「起動できなかった」
	ta.launched = nil
	d := mkAction(t, ta.actionsDir, "91_login", "x.bat")
	writeFile(t, d+"/kido.toml", "wait = true\nrequire_login = true\n")
	_, n = ta.hello(t)
	r = call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, "91_login"))
	if !strings.Contains(r.body, `"err":"start"`) || len(ta.launched) != 0 {
		t.Fatalf("%+v %v", r, ta.launched)
	}
}

// 手足役が拾ってから失敗したときは、動かし直さない(二重に動かさない)。
func TestExecuteNoFallbackAfterHelperTookIt(t *testing.T) {
	ta := pairedAgent(t)
	ta.setLoggedIn(true)
	mkAction(t, ta.actionsDir, "60_server", "start.bat")
	go func() {
		if j := ta.hub.poll(context.Background()); j != nil {
			ta.hub.complete(j.ID, helperResult{Err: "起動に失敗"})
		}
	}()
	_, n := ta.hello(t)
	call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, "60_server"))
	if len(ta.launched) != 0 {
		t.Fatalf("手足役が拾った仕事を動かし直した: %v", ta.launched)
	}
}
