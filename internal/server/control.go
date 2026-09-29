package server

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"time"

	"kido-agent/internal/auth"
	"kido-agent/internal/control"
)

// controlHandler は同じ PC の CLI(pair・check・uninstall)から使う窓口。
func (a *agent) controlHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /control/status", func(w http.ResponseWriter, r *http.Request) {
		a.writeStatus(w)
	})
	mux.HandleFunc("POST /control/pair", func(w http.ResponseWriter, r *http.Request) {
		if err := a.pairing.Open(); err != nil {
			a.logf("古い鍵を消せません: %v", err)
			writeError(w, http.StatusInternalServerError, "internal")
			return
		}
		a.logf("組める時間を開きました(10 分)")
		a.writeStatus(w)
	})
	mux.HandleFunc("POST /control/stop", func(w http.ResponseWriter, r *http.Request) {
		writeBody(w, http.StatusOK, []byte(`{"ok":true}`), "")
		select {
		case a.stopRequest <- struct{}{}:
		default:
		}
	})
	// 手足役がつなぎっぱなしにする窓口。next で仕事を待ち、result で結果を返す。
	mux.HandleFunc("GET /control/helper/next", func(w http.ResponseWriter, r *http.Request) {
		// 窓口の WriteTimeout(10 秒)は要求を読んだ時点から数えるので、30 秒保つこの
		// つなぎでは、10 秒より後に来た仕事を書き出せずに失ってしまう。保つ長さに合わせる。
		http.NewResponseController(w).SetWriteDeadline(time.Now().Add(a.hub.pollHold + 10*time.Second))
		if j := a.hub.poll(r.Context()); j != nil {
			body, _ := json.Marshal(j)
			writeBody(w, http.StatusOK, body, "")
			return
		}
		writeBody(w, http.StatusOK, []byte(`{}`), "") // 空応答。手足役はすぐつなぎ直す
	})
	mux.HandleFunc("POST /control/helper/result", func(w http.ResponseWriter, r *http.Request) {
		var res struct {
			ID    string `json:"id"`
			OK    bool   `json:"ok"`
			Error string `json:"error"`
			Code  int    `json:"code"` // wait の仕事の終了コード
		}
		if readJSON(r, &res) != nil {
			writeError(w, http.StatusBadRequest, "bad_request")
			return
		}
		msg := res.Error
		if res.OK {
			msg = ""
		} else if msg == "" {
			msg = "実行に失敗しました"
		}
		a.hub.complete(res.ID, helperResult{Code: res.Code, Err: msg})
		writeBody(w, http.StatusOK, []byte(`{"ok":true}`), "")
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 合言葉が空、または合わなければ断る。同じ PC の別ユーザーからのなりすましと、
		// ブラウザの中のページからの要求(合言葉を知りようがない)を止める。
		// 比べるのにかかる時間から合言葉を 1 文字ずつ当てられないよう、時間の一定な比較にする。
		if !a.tokenOK(r.Header.Get(control.Header)) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		mux.ServeHTTP(w, r)
	})
}

// tokenOK は手元の窓口の合言葉が合うか。空の合言葉(まだ書けていない)は誰も通さない。
func (a *agent) tokenOK(got string) bool {
	return a.token != "" && subtle.ConstantTimeCompare([]byte(got), []byte(a.token)) == 1
}

func (a *agent) writeStatus(w http.ResponseWriter) {
	st := a.pairing.State()
	body, _ := json.Marshal(control.Status{
		State:     st,
		Remaining: a.pairing.Remaining(),
		Paired:    st == auth.StatePaired,
		Version:   a.version,
		Session:   a.sessionActive(),
	})
	writeBody(w, http.StatusOK, body, "")
}
