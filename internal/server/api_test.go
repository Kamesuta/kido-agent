package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kido-agent/internal/auth"
)

type apiResult struct {
	status int
	body   string
	sig    string
}

func call(t *testing.T, h http.Handler, method, path, body string) apiResult {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	data, _ := io.ReadAll(rec.Body)
	if cl := rec.Header().Get("Content-Length"); cl == "" {
		t.Fatal("HTTP/1.0 の相手には Content-Length が要る")
	}
	return apiResult{rec.Code, string(data), rec.Header().Get("X-Kido-Sig")}
}

func callToken(t *testing.T, h http.Handler, method, path, body, token string) apiResult {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("X-Kido-Control", token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	data, _ := io.ReadAll(rec.Body)
	return apiResult{rec.Code, string(data), rec.Header().Get("X-Kido-Sig")}
}

func (ta *testAgent) hello(t *testing.T) (state, nonce string) {
	t.Helper()
	res := call(t, ta.apiHandler(), "GET", "/v1/hello", "")
	var v struct {
		V     int    `json:"v"`
		State string `json:"state"`
		Nonce string `json:"nonce"`
	}
	if err := json.Unmarshal([]byte(res.body), &v); err != nil || v.V != 1 {
		t.Fatalf("hello: %s", res.body)
	}
	return v.State, v.Nonce
}

func pairedAgent(t *testing.T) *testAgent {
	ta := newTestAgent(t)
	ta.pairing.Open()
	res := call(t, ta.apiHandler(), "POST", "/v1/pair", `{"key":"`+hexKey()+`"}`)
	if res.status != 200 || res.body != `{"ok":true}` {
		t.Fatalf("pair: %d %s", res.status, res.body)
	}
	return ta
}

func hexKey() string { return "3031323334353637383961626364656630313233343536373839616263646566" }

func listReq(nonce, sig string) string {
	return `{"nonce":"` + nonce + `","sig":"` + sig + `"}`
}

func TestPairEndpoint(t *testing.T) {
	ta := newTestAgent(t)
	h := ta.apiHandler()
	if st, _ := ta.hello(t); st != auth.StateUnpaired {
		t.Fatal(st)
	}
	if r := call(t, h, "POST", "/v1/pair", `{"key":"`+hexKey()+`"}`); r.status != 409 || !strings.Contains(r.body, "not_pairing") {
		t.Fatalf("組める時間の外: %+v", r)
	}
	ta.pairing.Open()
	if st, _ := ta.hello(t); st != auth.StatePairing {
		t.Fatal(st)
	}
	for _, bad := range []string{`{"key":"abc"}`, `not json`, `{"key":"` + strings.Repeat("zz", 32) + `"}`} {
		if r := call(t, h, "POST", "/v1/pair", bad); r.status != 400 {
			t.Fatalf("%s: %+v", bad, r)
		}
	}
	if r := call(t, h, "POST", "/v1/pair", `{"key":"`+hexKey()+`"}`); r.status != 200 {
		t.Fatalf("%+v", r)
	}
	if string(ta.pairing.Key()) != string(testKey) {
		t.Fatal("鍵が違う")
	}
}

func TestListSigned(t *testing.T) {
	ta := pairedAgent(t)
	mkAction(t, ta.actionsDir, "10_sleep", "sleep.ps1")
	mkAction(t, ta.actionsDir, "50_Game", "a.bat", "b.bat")
	_, n := ta.hello(t)
	res := call(t, ta.apiHandler(), "POST", "/v1/list", listReq(n, auth.Sign(testKey, "list", n)))
	if res.status != 200 {
		t.Fatalf("%+v", res)
	}
	want := `{"actions":[{"id":"10_sleep","name":"sleep"},{"id":"50_Game","name":"Game","broken":true}]}`
	if res.body != want {
		t.Fatalf("本文:\n%s", res.body)
	}
	if res.sig != auth.Sign(testKey, "list-ok", n, res.body) {
		t.Fatal("X-Kido-Sig が本文と合わない")
	}
	// 同じ nonce は 2 度使えない
	if r := call(t, ta.apiHandler(), "POST", "/v1/list", listReq(n, auth.Sign(testKey, "list", n))); r.status != 401 {
		t.Fatalf("使い回しが通った: %+v", r)
	}
}

func TestListRejects(t *testing.T) {
	ta := newTestAgent(t)
	_, n := ta.hello(t)
	if r := call(t, ta.apiHandler(), "POST", "/v1/list", listReq(n, auth.Sign(testKey, "list", n))); r.status != 409 || !strings.Contains(r.body, "not_paired") {
		t.Fatalf("組んでいない: %+v", r)
	}
	ta = pairedAgent(t)
	_, n = ta.hello(t)
	if r := call(t, ta.apiHandler(), "POST", "/v1/list", listReq(n, auth.Sign(testKey, "run", n))); r.status != 401 || r.sig != "" {
		t.Fatalf("署名違い: %+v", r)
	}
	// 署名が違っても nonce は捨てる
	if r := call(t, ta.apiHandler(), "POST", "/v1/list", listReq(n, auth.Sign(testKey, "list", n))); r.status != 401 {
		t.Fatalf("失敗した nonce が生き残った: %+v", r)
	}
	if r := call(t, ta.apiHandler(), "POST", "/v1/list", listReq("0000", auth.Sign(testKey, "list", "0000"))); r.status != 401 {
		t.Fatalf("出していない nonce: %+v", r)
	}
}

func TestControlRequiresToken(t *testing.T) {
	ta := newTestAgent(t)
	h := ta.controlHandler()
	// 合言葉なし
	if r := call(t, h, "POST", "/control/pair", ""); r.status != 403 || ta.pairing.State() != auth.StateUnpaired {
		t.Fatal("合言葉の無い要求で組める時間が開いた")
	}
	// 合言葉が違う
	if r := callToken(t, h, "POST", "/control/pair", "", "wrong"); r.status != 403 {
		t.Fatalf("違う合言葉が通った: %+v", r)
	}
	// 正しい合言葉
	if r := callToken(t, h, "POST", "/control/pair", "", "test-token"); r.status != 200 {
		t.Fatalf("正しい合言葉が通らない: %+v", r)
	}
}
