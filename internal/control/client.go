// Package control は、同じ PC の CLI(pair・check・uninstall)や手足役と、
// 待ち受け役の常駐アプリとの間の取り決め。
package control

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"kido-agent/internal/paths"
)

// Header の値は合言葉(control.token)。値が合わない要求は待ち受け役が断る。
// ブラウザの中のページから 127.0.0.1 へ勝手に POST されても、独自ヘッダ付きの
// 要求は事前確認で止まって届かないうえ、合言葉も知りようがない。
const Header = "X-Kido-Control"

// Status は常駐アプリの今の状態。
type Status struct {
	State     string `json:"state"`
	Remaining int    `json:"remaining"`
	Paired    bool   `json:"paired"`
	Version   string `json:"version"`
	Session   bool   `json:"session"` // 誰かがログインしているとみなせるか
}

// ErrNotRunning は待ち受け役に届かなかったことを表す(動いていない)。
var ErrNotRunning = errors.New("常駐アプリが動いていません")

// Do は合言葉を添えて待ち受け役へ要求を送り、応答の本文を返す。
// timeout はロングポーリング(手足役)のために呼び出し側が決める。
func Do(method, path string, body []byte, timeout time.Duration) ([]byte, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, "http://"+paths.ControlAddr+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set(Header, ReadToken())
	client := &http.Client{Timeout: timeout}
	res, err := client.Do(req)
	if err != nil {
		return nil, ErrNotRunning
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode != http.StatusOK {
		return data, fmt.Errorf("待ち受け役が要求を断りました(%s)", res.Status)
	}
	return data, nil
}

// Call は CLI から待ち受け役の状態を尋ねる・操作するときに使う。
func Call(method, path string) (Status, error) {
	var st Status
	data, err := Do(method, path, nil, 3*time.Second)
	if err != nil {
		return st, err
	}
	return st, json.Unmarshal(data, &st)
}
