package control

import (
	"os"
	"testing"
)

func TestTokenRoundTrip(t *testing.T) {
	t.Setenv("KIDO_AGENT_CONFIG_DIR", t.TempDir())
	if ReadToken() != "" {
		t.Fatal("まだ無いのに合言葉が返った")
	}
	tok, err := WriteToken()
	if err != nil || len(tok) != 64 {
		t.Fatalf("合言葉を書けない: %v %q", err, tok)
	}
	if ReadToken() != tok {
		t.Fatal("読み戻せない")
	}
	// 起動のたびに作り直す(前の合言葉は残らない)
	tok2, _ := WriteToken()
	if tok2 == tok || ReadToken() != tok2 {
		t.Fatal("作り直していない")
	}
	fi, _ := os.Stat(mustPath(t))
	if fi.Mode().Perm()&0o077 != 0 {
		t.Skipf("この環境の権限: %v(Windows では ACL で守る)", fi.Mode())
	}
}

func mustPath(t *testing.T) string {
	t.Helper()
	p, err := tokenPathForTest()
	if err != nil {
		t.Fatal(err)
	}
	return p
}
