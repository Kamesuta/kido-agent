package actions

import (
	"fmt"
	"testing"
)

func TestListCountLimit(t *testing.T) {
	var list []Action
	for i := 0; i < 30; i++ {
		list = append(list, Action{ID: fmt.Sprintf("%02d", i), Name: "n"})
	}
	if _, sent := ListBody(list, true); sent != MaxActions {
		t.Fatalf("24 個まで: %d", sent)
	}
	if body, _ := ListBody(nil, true); string(body) != `{"actions":[]}` {
		t.Fatal(string(body))
	}
}

func TestListBodyLoginMarking(t *testing.T) {
	list := []Action{
		{ID: "10_sleep", Name: "s", BeforeLogin: true},
		{ID: "20_lock", Name: "l"},
	}
	// ログインしていない(手足役がいない): before_login でないものに login を付ける
	body, _ := ListBody(list, false)
	if got := string(body); got != `{"actions":[{"id":"10_sleep","name":"s"},{"id":"20_lock","name":"l","login":true}]}` {
		t.Fatalf("login 未ログイン時:\n%s", got)
	}
	// ログイン中(手足役がいる): login は付けない
	body, _ = ListBody(list, true)
	if got := string(body); got != `{"actions":[{"id":"10_sleep","name":"s"},{"id":"20_lock","name":"l"}]}` {
		t.Fatalf("login ログイン中:\n%s", got)
	}
}
