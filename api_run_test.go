package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func runReq(nonce, id string) string {
	return fmt.Sprintf(`{"nonce":%q,"id":%q,"sig":%q}`, nonce, id, sign(testKey, "run", nonce, id))
}

func TestRunExecutesAfterReply(t *testing.T) {
	ta := pairedAgent(t)
	mkAction(t, ta.actionsDir, "00_sleep", "sleep.ps1", "readme.txt")
	var order []string
	launch := ta.launch
	ta.launch = func(a action) error { order = append(order, "launch"); return launch(a) }
	after := ta.after
	ta.after = func(d time.Duration, f func()) { order = append(order, "scheduled"); after(d, f) }
	_, n := ta.hello(t)
	res := call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, "00_sleep"))
	if res.status != 200 || res.body != `{"ok":true}` || res.sig != sign(testKey, "run-ok", n, res.body) {
		t.Fatalf("%+v", res)
	}
	if strings.Join(order, ",") != "scheduled,launch" || ta.delays[0] < 300*time.Millisecond {
		t.Fatalf("返事の後に 300ms 以上待って動かす: %v %v", order, ta.delays)
	}
	if len(ta.launched) != 1 || ta.launched[0] != "00_sleep/sleep.ps1" {
		t.Fatal(ta.launched)
	}
}

func TestRunErrors(t *testing.T) {
	ta := pairedAgent(t)
	mkAction(t, ta.actionsDir, "40_Game", "a.bat", "b.bat")
	cases := []struct {
		id     string
		status int
		code   string
	}{
		{"nope", 404, "unknown_action"},
		{"40_Game", 409, "broken_action"},
		{"../40_Game", 404, "unknown_action"},
	}
	for _, c := range cases {
		_, n := ta.hello(t)
		r := call(t, ta.apiHandler(), "POST", "/v1/run", runReq(n, c.id))
		if r.status != c.status || !strings.Contains(r.body, c.code) {
			t.Errorf("%s: %+v", c.id, r)
		}
	}
	_, n := ta.hello(t)
	bad := fmt.Sprintf(`{"nonce":%q,"id":"40_Game","sig":%q}`, n, sign(testKey, "run", n, "other"))
	if r := call(t, ta.apiHandler(), "POST", "/v1/run", bad); r.status != 401 {
		t.Errorf("署名違い: %+v", r)
	}
	if len(ta.launched) != 0 {
		t.Fatal("失敗したのに動かした")
	}
}

func TestListLimits(t *testing.T) {
	ta := pairedAgent(t)
	long := strings.Repeat("あ", 200)
	for i := 0; i < 30; i++ {
		d := mkAction(t, ta.actionsDir, fmt.Sprintf("%02d_%s", i, strings.Repeat("x", 50)), "a.bat")
		writeFile(t, d+"/kido.toml", "name = \""+strings.Repeat("名", 40)+"\"\nconfirm = \""+long+"\"\nicon = \""+strings.Repeat("a", 40)+"\"\n")
	}
	_, n := ta.hello(t)
	res := call(t, ta.apiHandler(), "POST", "/v1/list", listReq(n, sign(testKey, "list", n)))
	if len(res.body) > maxBodyBytes {
		t.Fatalf("本文が %d バイト", len(res.body))
	}
	if res.sig != sign(testKey, "list-ok", n, res.body) {
		t.Fatal("切り詰めた後の本文に署名する")
	}
	if !strings.Contains(res.body, `"name":"`+strings.Repeat("名", 32)+`"`) {
		t.Fatal("名前は 32 文字に切る")
	}
	actions, _ := ta.scan()
	body, sent := listBody(actions)
	if sent == 0 || sent >= maxActions || string(body) != res.body {
		t.Fatalf("sent=%d", sent)
	}
}

func TestListCountLimit(t *testing.T) {
	var actions []action
	for i := 0; i < 30; i++ {
		actions = append(actions, action{ID: fmt.Sprintf("%02d", i), Name: "n"})
	}
	if _, sent := listBody(actions); sent != maxActions {
		t.Fatalf("24 個まで: %d", sent)
	}
	if body, _ := listBody(nil); string(body) != `{"actions":[]}` {
		t.Fatal(string(body))
	}
}
