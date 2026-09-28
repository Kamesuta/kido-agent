package main

import (
	"net/http"
)

func (a *agent) handleList(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Nonce string `json:"nonce"`
		Sig   string `json:"sig"`
	}
	if readJSON(r, &req) != nil {
		writeError(w, http.StatusBadRequest, "bad_request")
		return
	}
	key := a.authorize(w, req.Nonce, req.Sig, "list", req.Nonce)
	if key == nil {
		return
	}
	actions, err := a.scan()
	if err != nil {
		a.logf("操作フォルダを読めません: %v", err)
	}
	body, _ := listBody(actions)
	writeBody(w, http.StatusOK, body, sign(key, "list-ok", req.Nonce, string(body)))
}

func (a *agent) handleRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Nonce string `json:"nonce"`
		ID    string `json:"id"`
		Sig   string `json:"sig"`
	}
	if readJSON(r, &req) != nil {
		writeError(w, http.StatusBadRequest, "bad_request")
		return
	}
	key := a.authorize(w, req.Nonce, req.Sig, "run", req.Nonce, req.ID)
	if key == nil {
		return
	}
	target, found := a.find(req.ID)
	switch {
	case !found:
		writeError(w, http.StatusNotFound, "unknown_action")
		return
	case target.Broken:
		writeError(w, http.StatusConflict, "broken_action")
		return
	}
	body := []byte(`{"ok":true}`)
	writeBody(w, http.StatusOK, body, sign(key, "run-ok", req.Nonce, string(body)))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	// スリープやシャットダウンを先に始めると返事が本体に届かないので、
	// 送り切ってから少し待って動かす。
	a.logf("実行します: %s (%s)", target.ID, target.Run)
	a.after(a.runDelay, func() {
		if err := a.launch(target); err != nil {
			a.logf("実行に失敗しました: %s: %v", target.ID, err)
		}
	})
}

func (a *agent) scan() ([]action, error) {
	return scanActions(a.actionsDir, a.goos)
}

// find は一覧に載せたものと同じ条件(ID として使える)で探す。
// 一覧から切り詰めて落ちた操作も、ID が分かっていれば動かせてよい。
func (a *agent) find(id string) (action, bool) {
	actions, _ := a.scan()
	for _, x := range actions {
		if x.ID == id && x.Invalid == "" {
			return x, true
		}
	}
	return action{}, false
}
