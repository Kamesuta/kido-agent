package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// controlHeader が無い要求は断る。ブラウザの中のページから 127.0.0.1 へ
// 勝手に POST されても、独自ヘッダ付きの要求は事前確認で止まるので届かない。
const controlHeader = "X-Kido-Control"

type controlStatus struct {
	State     string `json:"state"`
	Remaining int    `json:"remaining"`
	Paired    bool   `json:"paired"`
	Version   string `json:"version"`
}

// controlHandler は同じ PC の CLI(pair・check・uninstall)から使う窓口。
func (a *agent) controlHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /control/status", func(w http.ResponseWriter, r *http.Request) {
		writeBody(w, http.StatusOK, encodeJSON(a.status()), "")
	})
	mux.HandleFunc("POST /control/pair", func(w http.ResponseWriter, r *http.Request) {
		if err := a.openPairing(); err != nil {
			a.logf("古い鍵を消せません: %v", err)
			writeError(w, http.StatusInternalServerError, "internal")
			return
		}
		a.logf("組める時間を開きました(10 分)")
		writeBody(w, http.StatusOK, encodeJSON(a.status()), "")
	})
	mux.HandleFunc("POST /control/stop", func(w http.ResponseWriter, r *http.Request) {
		writeBody(w, http.StatusOK, []byte(`{"ok":true}`), "")
		select {
		case a.stopRequest <- struct{}{}:
		default:
		}
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(controlHeader) != "1" {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (a *agent) status() controlStatus {
	st := a.state()
	return controlStatus{State: st, Remaining: a.remaining(), Paired: st == statePaired, Version: version}
}

// errNotRunning は常駐アプリに届かなかったことを表す(動いていない)。
var errNotRunning = errors.New("常駐アプリが動いていません")

// callControl は CLI から常駐アプリへ要求を送る。
func callControl(method, path string) (controlStatus, error) {
	var st controlStatus
	req, _ := http.NewRequest(method, "http://"+controlAddr+path, nil)
	req.Header.Set(controlHeader, "1")
	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return st, errNotRunning
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return st, fmt.Errorf("常駐アプリが要求を断りました(%s)", res.Status)
	}
	return st, json.NewDecoder(res.Body).Decode(&st)
}
