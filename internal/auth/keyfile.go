package auth

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type keyFile struct {
	Key string `json:"key"`
}

// LoadKey は保存済みの鍵を読む。まだ組んでいなければ nil を返す。
func LoadKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var kf keyFile
	if err := json.Unmarshal(data, &kf); err != nil {
		return nil, fmt.Errorf("鍵のファイルが壊れています: %w", err)
	}
	key, err := hex.DecodeString(kf.Key)
	if err != nil || len(key) != 32 {
		return nil, errors.New("鍵のファイルの中身が 32 バイトの鍵ではありません")
	}
	return key, nil
}

// SaveKey は一時ファイルに書いてから名前を付け替える。
// 書いている途中で電源が落ちても、半端な鍵のファイルが残らないようにするため。
func SaveKey(path string, key []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, _ := json.Marshal(keyFile{Key: hex.EncodeToString(key)})
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// RemoveKey は組み直しのときに古い鍵を消す。無くても失敗にしない。
func RemoveKey(path string) error {
	err := os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
