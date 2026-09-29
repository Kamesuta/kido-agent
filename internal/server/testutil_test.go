package server

import (
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"kido-agent/internal/actions"
)

// testAgent は時計と実行を差し替えた agent。実際にスリープさせないため。
type testAgent struct {
	*agent
	clock    time.Time
	mu       sync.Mutex
	launched []string
	delays   []time.Duration
}

func newTestAgent(t *testing.T) *testAgent {
	t.Helper()
	base := t.TempDir()
	dir := filepath.Join(base, "KidoButtons")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	ta := &testAgent{clock: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	ta.agent = newAgent(filepath.Join(base, "cfg", "key.json"), dir, "windows", "test", nil)
	ta.token = "test-token"
	ta.now = func() time.Time { return ta.clock }
	ta.logf = t.Logf
	// 動かしている OS に左右されないよう、既定は Windows と同じ「手足役だけで決める」。
	ta.osSession = func() bool { return false }
	ta.launch = func(a actions.Action) error {
		ta.mu.Lock()
		defer ta.mu.Unlock()
		ta.launched = append(ta.launched, a.ID+"/"+a.Run)
		return nil
	}
	ta.after = func(d time.Duration, f func()) {
		ta.delays = append(ta.delays, d)
		f()
	}
	return ta
}

func (ta *testAgent) advance(d time.Duration) { ta.clock = ta.clock.Add(d) }

// setLoggedIn は手足役がいる/いないを装う。いるときは今の時刻を最後のポーリングに
// することで active() を真にし、いないときは 0 に戻す。
func (ta *testAgent) setLoggedIn(v bool) {
	ta.hub.mu.Lock()
	if v {
		ta.hub.lastPoll = ta.clock
	} else {
		ta.hub.lastPoll = time.Time{}
	}
	ta.hub.mu.Unlock()
}

// mkAction は操作フォルダを作り、中にファイルを置く(中身は名前をそのまま)。
func mkAction(t *testing.T, root, id string, files ...string) string {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		writeFile(t, filepath.Join(dir, f), f)
	}
	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

var testKey = []byte("0123456789abcdef0123456789abcdef")

// serveReal は本物の接続で試すための待ち受け。時間切れを短くして、
// wait がそれを越えても返事できることを確かめる。
func serveReal(t *testing.T, ta *testAgent) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := newServer(ta.apiHandler())
	s.ReadTimeout, s.WriteTimeout = 100*time.Millisecond, 100*time.Millisecond
	go s.Serve(ln)
	t.Cleanup(func() { s.Close() })
	return "http://" + ln.Addr().String()
}

func postRun(base, nonce, id string, timeout time.Duration) (string, error) {
	c := &http.Client{Timeout: timeout}
	res, err := c.Post(base+"/v1/run", "application/json", strings.NewReader(runReq(nonce, id)))
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	return string(b), err
}

func waitLog(t *testing.T, logs chan string, want string) {
	for {
		select {
		case l := <-logs:
			if strings.Contains(l, want) {
				return
			}
		case <-time.After(2 * time.Second):
			t.Errorf("ログに %q が出ない", want)
			return
		}
	}
}
