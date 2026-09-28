package main

import (
	"bytes"
	"encoding/json"
)

type wireAction struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Icon    string  `json:"icon,omitempty"`
	Confirm *string `json:"confirm,omitempty"`
	Broken  bool    `json:"broken,omitempty"`
}

// listBody は本体へ送る一覧の本文を作る。数の上限を超えた分と、本文が
// 4096 バイトに収まらない分は後ろから落とす。sent は実際に載せた数。
func listBody(actions []action) (body []byte, sent int) {
	var wire []wireAction
	for _, a := range actions {
		if a.Invalid != "" {
			continue
		}
		wire = append(wire, wireAction{a.ID, a.Name, a.Icon, a.Confirm, a.Broken})
	}
	if len(wire) > maxActions {
		wire = wire[:maxActions]
	}
	for {
		body = encodeJSON(map[string]any{"actions": nonNil(wire)})
		if len(body) <= maxBodyBytes || len(wire) == 0 {
			return body, len(wire)
		}
		wire = wire[:len(wire)-1]
	}
}

func nonNil(w []wireAction) []wireAction {
	if w == nil {
		return []wireAction{} // 空でも null ではなく [] を送る
	}
	return w
}

// encodeJSON は署名する本文を作る。HTML 向けのエスケープ(< を < にする等)は
// 本体には要らず、バイト数を無駄に増やすだけなので切る。
func encodeJSON(v any) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		panic(err) // 自分で組んだ値しか渡さないので、ここに来たら作りの誤り
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
}
