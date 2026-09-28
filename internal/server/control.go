package server

import (
	"encoding/json"
	"net/http"

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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(control.Header) != "1" {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (a *agent) writeStatus(w http.ResponseWriter) {
	st := a.pairing.State()
	body, _ := json.Marshal(control.Status{
		State:     st,
		Remaining: a.pairing.Remaining(),
		Paired:    st == auth.StatePaired,
		Version:   a.version,
	})
	writeBody(w, http.StatusOK, body, "")
}
