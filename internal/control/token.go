package control

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"

	"kido-agent/internal/paths"
)

// WriteToken は 32 バイトの乱数の合言葉を作って置き場に書き、その 16 進を返す。
// 一時ファイルから付け替えるのは、読み手が半端な中身を拾わないようにするため。
// 起動のたびに作り直すので、前に落ちた待ち受け役の古い合言葉は残らない。
func WriteToken() (string, error) {
	path, err := paths.TokenFile()
	if err != nil {
		return "", err
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	// 0600 は Unix の権限。Windows では利用者のプロファイル配下にあり、
	// 既定で本人しか読めないので、これで同じ PC の別ユーザーからは隠れる。
	if err := os.WriteFile(tmp, []byte(token), 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return "", err
	}
	return token, nil
}

// tokenPathForTest はテストから置き場を知るためのもの。
func tokenPathForTest() (string, error) { return paths.TokenFile() }

// ReadToken は待ち受け役が置いた合言葉を読む。待ち受け役がいなければ空を返す。
func ReadToken() string {
	path, err := paths.TokenFile()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
