package server

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"kido-agent/internal/auth"
)

// 本体から届く要求の本文の上限。取り決めの要求はどれも数百バイトに収まる。
const maxRequestBytes = 8192

// apiHandler は本体から届く要求の窓口。PROTOCOL.md の取り決めをそのまま実装する。
func (a *agent) apiHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/hello", a.handleHello)
	mux.HandleFunc("POST /v1/pair", a.handlePair)
	mux.HandleFunc("POST /v1/list", a.handleList)
	mux.HandleFunc("POST /v1/run", a.handleRun)
	return mux
}

func (a *agent) handleHello(w http.ResponseWriter, r *http.Request) {
	nonce, err := a.nonces.Issue()
	if err != nil {
		a.logf("nonce を作れません: %v", err)
		writeError(w, http.StatusInternalServerError, "internal")
		return
	}
	body, _ := json.Marshal(struct {
		V       int    `json:"v"`
		State   string `json:"state"`
		Nonce   string `json:"nonce"`
		Session bool   `json:"session"` // 誰かがログインしているとみなせるか(sessionActive)
	}{1, a.pairing.State(), nonce, a.sessionActive()})
	writeBody(w, http.StatusOK, body, "")
}

func (a *agent) handlePair(w http.ResponseWriter, r *http.Request) {
	if a.pairing.State() != auth.StatePairing {
		writeError(w, http.StatusConflict, "not_pairing")
		return
	}
	var req struct {
		Key string `json:"key"`
	}
	if readJSON(r, &req) != nil || len(req.Key) != 64 {
		writeError(w, http.StatusBadRequest, "bad_request")
		return
	}
	key, err := hex.DecodeString(req.Key)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request")
		return
	}
	ok, err := a.pairing.Accept(key)
	switch {
	case err != nil:
		a.logf("鍵を保存できません: %v", err)
		writeError(w, http.StatusInternalServerError, "internal")
	case !ok:
		writeError(w, http.StatusConflict, "not_pairing")
	default:
		a.logf("本体と組みました(%s)", r.RemoteAddr)
		writeBody(w, http.StatusOK, []byte(`{"ok":true}`), "")
	}
}

// authorize は nonce を捨ててから鍵と署名を確かめる。通らなければ応答まで済ませて nil を返す。
func (a *agent) authorize(w http.ResponseWriter, nonce, sig string, parts ...string) []byte {
	fresh := a.nonces.Take(nonce)
	key := a.pairing.Key()
	if key == nil {
		writeError(w, http.StatusConflict, "not_paired")
		return nil
	}
	if !fresh || !auth.Verify(key, sig, parts...) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return nil
	}
	return key
}

func readJSON(r *http.Request, v any) error {
	data, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBytes))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// writeBody は Content-Length を付けて本文を送る。本体は HTTP/1.0 で話すので、
// 分割転送(chunked)にならないよう長さを先に決めておく。
func writeBody(w http.ResponseWriter, status int, body []byte, sig string) {
	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("Content-Length", strconv.Itoa(len(body)))
	h.Set("Connection", "close")
	if sig != "" {
		h.Set("X-Kido-Sig", sig)
	}
	w.WriteHeader(status)
	w.Write(body)
}

func writeError(w http.ResponseWriter, status int, code string) {
	body, _ := json.Marshal(struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}{false, code})
	writeBody(w, status, body, "")
}
