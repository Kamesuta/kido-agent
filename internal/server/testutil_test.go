package server

import (
	"os"
	"path/filepath"
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
