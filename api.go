package main

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

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
	nonce, err := a.nonces.issue()
	if err != nil {
		a.logf("nonce を作れません: %v", err)
		writeError(w, http.StatusInternalServerError, "internal")
		return
	}
	body := encodeJSON(struct {
		V     int    `json:"v"`
		State string `json:"state"`
		Nonce string `json:"nonce"`
	}{1, a.state(), nonce})
	writeBody(w, http.StatusOK, body, "")
}

func (a *agent) handlePair(w http.ResponseWriter, r *http.Request) {
	if a.state() != statePairing {
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
	ok, err := a.acceptKey(key)
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
	fresh := a.nonces.take(nonce)
	key := a.currentKey()
	if key == nil {
		writeError(w, http.StatusConflict, "not_paired")
		return nil
	}
	if !fresh || !verify(key, sig, parts...) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return nil
	}
	return key
}

func readJSON(r *http.Request, v any) error {
	data, err := io.ReadAll(io.LimitReader(r.Body, maxReqBodyLen))
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
	writeBody(w, status, encodeJSON(map[string]any{"ok": false, "error": code}), "")
}
