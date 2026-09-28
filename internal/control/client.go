// Package control は、同じ PC の CLI(pair・check・uninstall)と常駐アプリの間の取り決め。
package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"kido-agent/internal/paths"
)

// Header が無い要求を常駐アプリは断る。ブラウザの中のページから 127.0.0.1 へ
// 勝手に POST されても、独自ヘッダ付きの要求は事前確認で止まるので届かない。
const Header = "X-Kido-Control"

// Status は常駐アプリの今の状態。
type Status struct {
	State     string `json:"state"`
	Remaining int    `json:"remaining"`
	Paired    bool   `json:"paired"`
	Version   string `json:"version"`
}

// ErrNotRunning は常駐アプリに届かなかったことを表す(動いていない)。
var ErrNotRunning = errors.New("常駐アプリが動いていません")

// Call は CLI から常駐アプリへ要求を送る。
func Call(method, path string) (Status, error) {
	var st Status
	req, _ := http.NewRequest(method, "http://"+paths.ControlAddr+path, nil)
	req.Header.Set(Header, "1")
	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return st, ErrNotRunning
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return st, fmt.Errorf("常駐アプリが要求を断りました(%s)", res.Status)
	}
	return st, json.NewDecoder(res.Body).Decode(&st)
}
