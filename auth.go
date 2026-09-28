package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// sign は取り決めどおり、部品を "\n" でつないだものの HMAC-SHA256 を小文字 16 進で返す。
func sign(key []byte, parts ...string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(mac.Sum(nil))
}

// verify は時間差で中身を推し量られないよう、比較を一定時間で行う。
func verify(key []byte, got string, parts ...string) bool {
	if len(key) == 0 {
		return false
	}
	return hmac.Equal([]byte(sign(key, parts...)), []byte(got))
}
