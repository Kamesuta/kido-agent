package server

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"kido-agent/internal/actions"
	"kido-agent/internal/auth"
)

func runReq(nonce, id string) string {
	return fmt.Sprintf(`{"nonce":%q,"id":%q,"sig":%q}`, nonce, id, auth.Sign(testKey, "run", nonce, id))
}

func TestRunExecutesAfterReply(t *testing.T) {
	ta := pairedAgent(t)
	// kido.toml なし(既定)。ログインしていなくても待ち受け役が自分で動かす
	mkAction(t, ta.actionsDir, "10_sleep", "sleep.ps1", "readme.txt")
	var order []string
	launch := ta.launch
	ta.launch = func(a actions.Action) error { order = append(order, "launch"); return launch(a) }
	after := ta.after
	ta.after = func(d time.Duration, f func()) { order = append(order, "scheduled"); after(d, f) }
	_, n := ta.hello(t)
	res := call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, "10_sleep"))
	if res.status != 200 || res.body != `{"ok":true}` || res.sig != auth.Sign(testKey, "run-ok", n, res.body) {
		t.Fatalf("%+v", res)
	}
	if strings.Join(order, ",") != "scheduled,launch" || ta.delays[0] < 300*time.Millisecond {
		t.Fatalf("返事の後に 300ms 以上待って動かす: %v %v", order, ta.delays)
	}
	if len(ta.launched) != 1 || ta.launched[0] != "10_sleep/sleep.ps1" {
		t.Fatal(ta.launched)
	}
}

func TestRunErrors(t *testing.T) {
	ta := pairedAgent(t)
	mkAction(t, ta.actionsDir, "50_Game", "a.bat", "b.bat")
	cases := []struct {
		id     string
		status int
		code   string
	}{
		{"nope", 404, "unknown_action"},
		{"50_Game", 409, "broken_action"},
		{"../50_Game", 404, "unknown_action"},
	}
	for _, c := range cases {
		_, n := ta.hello(t)
		r := call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, c.id))
		if r.status != c.status || !strings.Contains(r.body, c.code) {
			t.Errorf("%s: %+v", c.id, r)
		}
	}
	_, n := ta.hello(t)
	bad := fmt.Sprintf(`{"nonce":%q,"id":"50_Game","sig":%q}`, n, auth.Sign(testKey, "run", n, "other"))
	if r := call(t, ta.apiHandler(), "POST", "/v1/run", bad); r.status != 401 {
		t.Errorf("署名違い: %+v", r)
	}
	if len(ta.launched) != 0 {
		t.Fatal("失敗したのに動かした")
	}
}

func TestListLimits(t *testing.T) {
	ta := pairedAgent(t)
	ta.setLoggedIn(true) // login の印が入らない状態で切り詰めだけを見る
	long := strings.Repeat("あ", 200)
	for i := 0; i < 30; i++ {
		d := mkAction(t, ta.actionsDir, fmt.Sprintf("%02d_%s", i, strings.Repeat("x", 50)), "a.bat")
		writeFile(t, d+"/kido.toml", "name = \""+strings.Repeat("名", 40)+"\"\nconfirm = \""+long+"\"\nicon = \""+strings.Repeat("a", 40)+"\"\n")
	}
	_, n := ta.hello(t)
	res := call(t, ta.apiHandler(), "POST", "/v1/list", listReq(n, auth.Sign(testKey, "list", n)))
	if len(res.body) > actions.MaxBodyBytes {
		t.Fatalf("本文が %d バイト", len(res.body))
	}
	if res.sig != auth.Sign(testKey, "list-ok", n, res.body) {
		t.Fatal("切り詰めた後の本文に署名する")
	}
	if !strings.Contains(res.body, `"name":"`+strings.Repeat("名", 32)+`"`) {
		t.Fatal("名前は 32 文字に切る")
	}
	list, _ := actions.Scan(ta.actionsDir, ta.goos)
	body, sent := actions.ListBody(list, true)
	if sent == 0 || sent >= actions.MaxActions || string(body) != res.body {
		t.Fatalf("sent=%d", sent)
	}
}

func TestRunNeedsLoginWhenNotLoggedIn(t *testing.T) {
	ta := pairedAgent(t)
	// ログインしていない。require_login の操作は動かせない
	d := mkAction(t, ta.actionsDir, "20_lock", "lock.ps1")
	writeFile(t, d+"/kido.toml", "require_login = true\n")
	_, n := ta.hello(t)
	r := call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, "20_lock"))
	if r.status != 409 || !strings.Contains(r.body, "needs_login") {
		t.Fatalf("%+v", r)
	}
	if len(ta.launched) != 0 {
		t.Fatal("動かしてはいけない")
	}
}

func TestRunViaHelperWhenLoggedIn(t *testing.T) {
	ta := pairedAgent(t)
	ta.setLoggedIn(true)
	mkAction(t, ta.actionsDir, "20_lock", "lock.ps1")
	// 手足役が取りに来る役をたてる
	got := make(chan *helperJob, 1)
	go func() {
		j := ta.hub.poll()
		got <- j
		if j != nil {
			ta.hub.complete(j.ID, helperResult{})
		}
	}()
	_, n := ta.hello(t)
	r := call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, "20_lock"))
	if r.status != 200 {
		t.Fatalf("%+v", r)
	}
	select {
	case j := <-got:
		if j == nil || j.ID != "20_lock#1" || j.Wait {
			t.Fatalf("手足役に流れていない: %+v", j)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("手足役に流れてこない")
	}
	// ログイン中はローカル実行(a.launch)は使わない
	if len(ta.launched) != 0 {
		t.Fatal("ログイン中は手足役経由のはず")
	}
}

func TestHelloAndListSessionFlag(t *testing.T) {
	ta := pairedAgent(t)
	mkAction(t, ta.actionsDir, "10_sleep", "sleep.ps1")
	mkAction(t, ta.actionsDir, "20_lock", "lock.ps1")
	writeFile(t, ta.actionsDir+"/20_lock/kido.toml", "require_login = true\n")

	// ログインしていない: hello の session は false、list の 20_lock に login
	res := call(t, ta.apiHandler(), "GET", "/v1/hello", "")
	if !strings.Contains(res.body, `"session":false`) {
		t.Fatalf("session false のはず: %s", res.body)
	}
	_, n := ta.hello(t)
	lr := call(t, ta.apiHandler(), "POST", "/v1/list", listReq(n, auth.Sign(testKey, "list", n)))
	if !strings.Contains(lr.body, `"id":"20_lock","name":"lock","login":true`) {
		t.Fatalf("require_login の操作に login が要る: %s", lr.body)
	}
	if strings.Contains(lr.body, `"id":"10_sleep","name":"sleep","login":true`) {
		t.Fatalf("設定なしの操作に login は付けない: %s", lr.body)
	}

	// ログイン中: session true、login は付かない
	ta.setLoggedIn(true)
	res = call(t, ta.apiHandler(), "GET", "/v1/hello", "")
	if !strings.Contains(res.body, `"session":true`) {
		t.Fatalf("session true のはず: %s", res.body)
	}
	_, n = ta.hello(t)
	lr = call(t, ta.apiHandler(), "POST", "/v1/list", listReq(n, auth.Sign(testKey, "list", n)))
	if strings.Contains(lr.body, `"login":true`) {
		t.Fatalf("ログイン中は login を付けない: %s", lr.body)
	}
}
