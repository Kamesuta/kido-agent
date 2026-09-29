package server

import (
	"fmt"
	"net/http"
	"time"

	"kido-agent/internal/actions"
	"kido-agent/internal/auth"
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
	list, err := actions.Scan(a.actionsDir, a.goos)
	if err != nil {
		a.logf("操作フォルダを読めません: %v", err)
	}
	body, _ := actions.ListBody(list, a.sessionActive())
	writeBody(w, http.StatusOK, body, auth.Sign(key, "list-ok", req.Nonce, string(body)))
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
	target, found := actions.Find(a.actionsDir, a.goos, req.ID)
	session := a.sessionActive()
	switch {
	case !found:
		writeError(w, http.StatusNotFound, "unknown_action")
		return
	case target.Broken:
		writeError(w, http.StatusConflict, "broken_action")
		return
	case !session && target.RequireLogin:
		// 誰もログインしておらず、ログインしてから使う約束(require_login)の操作。
		writeError(w, http.StatusConflict, "needs_login")
		return
	}
	if target.Wait {
		a.runWaited(w, r, key, req.Nonce, target, session)
		return
	}
	body := []byte(`{"ok":true}`)
	writeBody(w, http.StatusOK, body, auth.Sign(key, "run-ok", req.Nonce, string(body)))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	// スリープやシャットダウンを先に始めると返事が本体に届かないので、
	// 送り切ってから少し待って動かす。ログイン中は手足役へ、そうでなければ自分で。
	a.logf("実行します: %s (%s, helper=%v)", target.ID, target.Run, session)
	a.after(a.runDelay, func() { a.execute(target, session) })
}

// runWaited は wait の操作を動かし、終わってから終了コードで返事をする。
// 常駐アプリ側は時間切れを持たないので、この接続の書き込みの時間切れも外す。
// WriteTimeout は要求を読んだ時点から数えるので、外さないと長く待った後の返事が書けない。
func (a *agent) runWaited(w http.ResponseWriter, r *http.Request, key []byte, nonce string, t actions.Action, session bool) {
	http.NewResponseController(w).SetWriteDeadline(time.Time{})
	a.logf("実行して終わりを待ちます: %s (%s, helper=%v)", t.ID, t.Run, session)
	// 1 つ空けておき、相手が先に去っても実行の側が送れずに残らないようにする。
	done := make(chan []byte, 1)
	go func() {
		code, err := a.executeWait(r.Context(), t, session)
		a.logf("終わりました: %s code=%d err=%v", t.ID, code, err)
		done <- waitBody(code, err)
	}()
	select {
	case body := <-done:
		writeBody(w, http.StatusOK, body, auth.Sign(key, "run-ok", nonce, string(body)))
	case <-r.Context().Done():
		// 相手が接続を切った。返事は捨てるが、動いているものは止めない。
		a.logf("相手が切ったので結果は返しません(%s は動かしたまま)", t.ID)
	}
}

// waitBody は wait の結果の本文。0 で終われば待たないときと同じ成功の本文にし、
// 本体が wait の有無で読み分けなくて済むようにする。
func waitBody(code int, err error) []byte {
	switch {
	case err != nil:
		return []byte(`{"ok":false,"err":"start"}`)
	case code != 0:
		return fmt.Appendf(nil, `{"ok":false,"err":"exit","code":%d}`, code)
	}
	return []byte(`{"ok":true}`)
}
