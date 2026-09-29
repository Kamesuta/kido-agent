package server

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kido-agent/internal/auth"
)

// ブラウザの印のある要求は、組める時間でも鍵を受け取らない。
func TestAPIRejectsBrowser(t *testing.T) {
	ta := newTestAgent(t)
	ta.pairing.Open()
	h := ta.apiHandler()
	body := `{"key":"` + hexKey() + `"}`
	cases := []struct {
		name   string
		host   string
		header map[string]string
		status int
		code   string
	}{
		{"Origin", "", map[string]string{"Origin": "https://evil.example", "Content-Type": "application/json"}, 403, "forbidden"},
		{"Origin null", "", map[string]string{"Origin": "null", "Content-Type": "application/json"}, 403, "forbidden"},
		{"Sec-Fetch-Site", "", map[string]string{"Sec-Fetch-Site": "cross-site", "Content-Type": "application/json"}, 403, "forbidden"},
		{"名前の Host", "evil.example:47821", map[string]string{"Content-Type": "application/json"}, 403, "forbidden"},
		{"localhost", "localhost:47821", map[string]string{"Content-Type": "application/json"}, 403, "forbidden"},
		{"no-cors の text/plain", "", map[string]string{"Content-Type": "text/plain;charset=UTF-8"}, 415, "unsupported_media_type"},
		{"フォーム", "", map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, 415, "unsupported_media_type"},
		{"Content-Type なし", "", nil, 415, "unsupported_media_type"},
	}
	for _, c := range cases {
		req := httptest.NewRequest("POST", "/v1/pair", strings.NewReader(body))
		req.Host = c.host
		for k, v := range c.header {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		b, _ := io.ReadAll(rec.Body)
		if rec.Code != c.status || !strings.Contains(string(b), `"error":"`+c.code+`"`) {
			t.Errorf("%s: %d %s", c.name, rec.Code, b)
		}
	}
	if ta.pairing.State() != auth.StatePairing {
		t.Fatal("断ったのに組んでしまった")
	}
	// DNS rebinding で hello を読ませない
	req := httptest.NewRequest("GET", "/v1/hello", nil)
	req.Host = "rebind.example:47821"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 403 || strings.Contains(rec.Body.String(), "nonce") {
		t.Fatalf("名前の Host に hello を答えた: %d %s", rec.Code, rec.Body.String())
	}
}

// 本体の形(Host なし・IP の Host、パラメータ付きの application/json)は通す。
func TestAPIAcceptsHub(t *testing.T) {
	for _, host := range []string{"", "192.168.1.20", "192.168.1.20:47821", "127.0.0.1:47821", "[::1]:47821", "[fe80::1]", "::1"} {
		for _, ct := range []string{"application/json", "application/json; charset=utf-8", "Application/JSON"} {
			ta := newTestAgent(t)
			ta.pairing.Open()
			req := httptest.NewRequest("POST", "/v1/pair", strings.NewReader(`{"key":"`+hexKey()+`"}`))
			req.Host = host
			req.Header.Set("Content-Type", ct)
			rec := httptest.NewRecorder()
			ta.apiHandler().ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("Host=%q Content-Type=%q: %d %s", host, ct, rec.Code, rec.Body.String())
			}
		}
	}
}

// 本体と同じく、HTTP/1.0 で Host を付けずに送った要求が本物の接続で通る。
func TestAPIAcceptsHTTP10WithoutHost(t *testing.T) {
	ta := newTestAgent(t)
	base := serveReal(t, ta)
	conn, err := net.Dial("tcp", strings.TrimPrefix(base, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	io.WriteString(conn, "GET /v1/hello HTTP/1.0\r\nConnection: close\r\n\r\n")
	b, _ := io.ReadAll(conn)
	if !strings.HasPrefix(string(b), "HTTP/1.0 200") || !strings.Contains(string(b), `"nonce"`) {
		t.Fatalf("%s", b)
	}
}
