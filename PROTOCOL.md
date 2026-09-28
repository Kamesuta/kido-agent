# kido-agent プロトコル v1

Kido の本体(スマホからの操作を、同じ LAN の PC に取り次ぐ機器)と、PC の常駐アプリ `kido-agent` の間の取り決め。

## 通信

- **HTTP/1.0 over TCP、ポート 47821**(`0.0.0.0` で待ち受け)。
  - 本体が要求を送り、常駐アプリが答える
  - 要求は `Connection: close` にして `Content-Length` を付ける。応答は、本文を読み切ったら接続を閉じる
  - 本文は UTF-8 の JSON
- 応答の本文は **4096 バイト以内**。常駐アプリは、超えないように一覧を切り詰める(下の「上限」)
- 本体は 1 回の要求に 3 秒の時間切れを持つ

## 状態

常駐アプリの状態は3つ。

| state | 意味 |
|---|---|
| `unpaired` | 鍵を持っていない。組める時間も開いていない |
| `pairing` | 組める時間(`kido-agent pair` から 10 分)が開いている |
| `paired` | 鍵を持っている |

## 鍵と署名

- 鍵は 32 バイト。本体が作って `pair` で渡す。PC ごとに別の鍵にする
- 署名は `HMAC-SHA256(key, message)` を小文字の 16 進(64 文字)にしたもの
- メッセージは、下の各所で決めた文字列を `\n` でつないだ UTF-8 のバイト列
- **nonce は常駐アプリが `hello` で出す**。16 バイトの乱数を 16 進にしたもの(32 文字)
  - 1 回きりで、署名の検証に使った時点で(成否を問わず)捨てる
  - 出してから 30 秒で失効する。同時に持つのは 32 個までで、超えたら古いものから捨てる

## エンドポイント

### `GET /v1/hello`

```json
{"v":1,"state":"paired","nonce":"<32 hex>"}
```

- 誰でも呼べる(認証なし)
- 本体はこれで「常駐アプリがいる」「組めるか」「次の要求の nonce」を知る

### `POST /v1/pair`

要求: `{"key":"<64 hex>"}`

- `state` が `pairing` のときだけ受け付ける。鍵を保存し、組める時間を閉じ、`paired` になる
  - 200 `{"ok":true}`
- それ以外は 409 `{"ok":false,"error":"not_pairing"}`
- 形が違えば 400 `{"ok":false,"error":"bad_request"}`
- **`paired` のときに `pair` をやり直すと、古い鍵は `kido-agent pair` の時点で消える**。つまり組める時間を開くと、それまでの鍵は使えなくなる

### `POST /v1/list`

要求: `{"nonce":"<hello の nonce>","sig":"<HMAC(key, "list\n" + nonce)>"}`

応答 200:

```json
{"actions":[
  {"id":"00_sleep","name":"スリープ","icon":"moon"},
  {"id":"30_shutdown","name":"シャットダウン","icon":"power","confirm":"開いているアプリはすべて閉じられます。\n保存していないデータは消えてしまいます。"},
  {"id":"40_Minecraft起動","name":"Minecraft起動","broken":true}
]}
```

- 応答には **`X-Kido-Sig: <HMAC(key, "list-ok\n" + nonce + "\n" + 本文のバイト列)>`** ヘッダを付ける。本体は本文を JSON として読む前に、生のバイト列で確かめる
- `id` はフォルダ名そのもの。`name` はフォルダ名から先頭の番号と `_` を除いたもの
  - `00_sleep` → `sleep`。同梱の操作は `kido.toml` の `name` で日本語名を付ける(下)
- `icon` は `kido.toml` の `icon`。無ければ省く
- `confirm` は `kido.toml` の `confirm`。無ければ省く。空文字なら `""` を送る(本文なしの確認)
- `broken` は設定が壊れていて押せない操作(実行できるファイルが無い・2つ以上ある・`kido.toml` が読めない)だけに `true` を付ける
- 並びはフォルダ名の昇順
- 失敗
  - nonce が無い・失効・署名違い: 401 `{"ok":false,"error":"unauthorized"}`
  - `paired` でない: 409 `{"ok":false,"error":"not_paired"}`

### `POST /v1/run`

要求: `{"nonce":"…","id":"00_sleep","sig":"<HMAC(key, "run\n" + nonce + "\n" + id)>"}`

- 200 `{"ok":true}` + `X-Kido-Sig: <HMAC(key, "run-ok\n" + nonce + "\n" + 本文のバイト列)>`
  - **応答を送り切ってから実行する**(少なくとも 300ms 待つ)。先に寝ると、本体に返事が届かない
- 失敗
  - 401 unauthorized
  - 409 not_paired
  - 404 `{"ok":false,"error":"unknown_action"}`: その ID の操作が無い
  - 409 `{"ok":false,"error":"broken_action"}`: その操作の設定が壊れている

## 上限(常駐アプリが守る。本体も超えたら切り詰める)

| 項目 | 上限 |
|---|---|
| 操作の数 | 24 |
| `id` | 64 バイト(UTF-8)。`/` `\` 制御文字を含まない。`.` で始まらない |
| `name` | 32 文字 |
| `icon` | 40 文字、`[a-z0-9-]` のみ(違えば省く) |
| `confirm` | 200 文字 |

## 操作フォルダ(常駐アプリの中の話)

- `~/Kido/<フォルダ>/` が 1 つの操作。`.` で始まるフォルダは無視する
- 実行できるファイル
  - Windows: `.ps1` `.bat` `.cmd` `.exe` `.lnk`
  - Mac: `.sh` `.command` `.app`
  - Linux: `.sh`、実行権限の付いたファイル、`.desktop`
- `kido.toml`(任意、UTF-8、BOM 可)
  - `name`(表示名。無ければフォルダ名から作る)
  - `icon`
  - `confirm`
  - `run`(実行するファイル名。フォルダの中だけ)
  - 知らないキーは `kido-agent check` で警告する(動作は止めない)
