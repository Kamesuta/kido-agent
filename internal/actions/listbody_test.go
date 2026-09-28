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
	if _, sent := ListBody(list); sent != MaxActions {
		t.Fatalf("24 個まで: %d", sent)
	}
	if body, _ := ListBody(nil); string(body) != `{"actions":[]}` {
		t.Fatal(string(body))
	}
}
