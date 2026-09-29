package server

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// execJob は ID だけを受け取り、自分の操作フォルダで引き直したものを動かす。
func TestExecJobResolvesOwnFolder(t *testing.T) {
	root := t.TempDir()
	mkAction(t, root, "10_sleep", "sleep.ps1")
	w := mkAction(t, root, "90_build", "build.bat")
	writeFile(t, filepath.Join(w, "kido.toml"), "wait = true\n")
	l := mkAction(t, root, "20_lock", "lock.ps1")
	writeFile(t, filepath.Join(l, "kido.toml"), "require_login = true\n")
	mkAction(t, root, "50_game", "a.bat", "b.bat") // 2 つあって押せない

	var ran []string
	run := func(dir, file string) error { ran = append(ran, filepath.Join(dir, file)); return nil }
	runWait := func(dir, file string) (int, error) { ran = append(ran, filepath.Join(dir, file)); return 7, nil }

	if _, err := execJob(root, "windows", helperJob{ID: "10_sleep#1", Action: "10_sleep"}, run, runWait); err != nil {
		t.Fatal(err)
	}
	if code, err := execJob(root, "windows", helperJob{ID: "90_build#2", Action: "90_build", Wait: true}, run, runWait); err != nil || code != 7 {
		t.Fatalf("wait は終了コードを返す: %d %v", code, err)
	}
	if _, err := execJob(root, "windows", helperJob{ID: "20_lock#3", Action: "20_lock", RequireLogin: true}, run, runWait); err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(root, "10_sleep", "sleep.ps1"),
		filepath.Join(root, "90_build", "build.bat"),
		filepath.Join(root, "20_lock", "lock.ps1"),
	}
	if strings.Join(ran, "|") != strings.Join(want, "|") {
		t.Fatalf("動かしたもの: %v", ran)
	}

	ran = nil
	bad := []struct {
		job  helperJob
		want string
	}{
		{helperJob{ID: "x#1", Action: "nope"}, "見つかりません"},
		{helperJob{ID: "x#2", Action: "../10_sleep"}, "見つかりません"},
		{helperJob{ID: "x#3", Action: "50_game"}, "押せない"},
		{helperJob{ID: "x#4", Action: "10_sleep", Wait: true}, "食い違って"},
		{helperJob{ID: "x#5", Action: "90_build"}, "食い違って"},
		{helperJob{ID: "x#6", Action: "20_lock"}, "食い違って"},
		{helperJob{ID: "x#7", Action: "10_sleep", RequireLogin: true}, "食い違って"},
	}
	for _, c := range bad {
		if _, err := execJob(root, "windows", c.job, run, runWait); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%+v: %v", c.job, err)
		}
	}
	if len(ran) != 0 {
		t.Fatalf("失敗のはずなのに動かした: %v", ran)
	}
}

// 待ち受け役から手足役へ流れる本文に、フォルダや実行するファイルが載らない。
func TestHelperJobCarriesOnlyID(t *testing.T) {
	ta := pairedAgent(t)
	ta.hub.pollHold = 2 * time.Second
	base := serveControl(t, ta)
	got := make(chan string, 1)
	go func() { b, _ := pollOnce(context.Background(), base); got <- b }()
	time.Sleep(100 * time.Millisecond)
	go ta.hub.dispatch(context.Background(), "20_lock", true, false)
	b := <-got
	if b != `{"id":"20_lock#1","action":"20_lock","wait":false,"require_login":true}` {
		t.Fatalf("本文: %s", b)
	}
}
