package server

import (
	"mime"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// guardAPI は、本体ではなくブラウザの中のページから届いた要求を先に断る。
// 本体の窓口は LAN に開いているので、同じ PC や LAN の PC で開いた悪意あるページが
// fetch("http://127.0.0.1:47821/v1/pair", {mode: "no-cors"}) のように、組める時間に
// 鍵を送り込めてしまう。DNS rebinding を使えば hello の答えも読める。
// 本体は HTTP/1.0 で Host も Origin も送らず、POST には必ず application/json を付けるので、
// そこから外れるものはブラウザ(か取り決めを知らない相手)とみなしてよい。
func guardAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status, code := rejectBrowser(r); status != 0 {
			writeError(w, status, code)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// rejectBrowser は断る理由を HTTP のステータスとエラー名で返す。通すなら 0。
func rejectBrowser(r *http.Request) (int, string) {
	// ブラウザは他所のページからの要求に Origin を、今の版ではどの要求にも
	// Sec-Fetch-Site を付ける。どちらもページの側からは消せない。
	if len(r.Header.Values("Origin")) > 0 || len(r.Header.Values("Sec-Fetch-Site")) > 0 {
		return http.StatusForbidden, "forbidden"
	}
	// DNS rebinding では、ブラウザは攻撃者の名前を Host に入れてくる。
	// 本体は Host を送らない(送るとしても IP で呼ぶ)ので、名前の Host は断る。
	if r.Host != "" && !isIPHost(r.Host) {
		return http.StatusForbidden, "forbidden"
	}
	// no-cors の fetch やフォームの送信は application/json を付けられない
	// (付けると事前確認になり、それは上の Origin で断る)。
	if r.Method == http.MethodPost {
		mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mt != "application/json" {
			return http.StatusUnsupportedMediaType, "unsupported_media_type"
		}
	}
	return 0, ""
}

// isIPHost は Host の値が IP アドレスそのもの(:port 付き・[IPv6] も可)か。
func isIPHost(h string) bool {
	if host, _, err := net.SplitHostPort(h); err == nil {
		h = host
	} else {
		h = strings.TrimSuffix(strings.TrimPrefix(h, "["), "]")
	}
	_, err := netip.ParseAddr(h)
	return err == nil
}
