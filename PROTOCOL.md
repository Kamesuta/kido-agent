# kido-agent プロトコル v1

Kido の本体(スマホからの操作を、同じ LAN の PC に取り次ぐ機器)と、PC の常駐アプリ `kido-agent` の間の取り決め。

## 通信

- **HTTP/1.0 over TCP、ポート 47821**(`0.0.0.0` で待ち受け)。
  - 本体が要求を送り、常駐アプリが答える
  - 要求は `Connection: close` にして `Content-Length` を付ける。応答は、本文を読み切ったら接続を閉じる
  - 本文は UTF-8 の JSON
- **POST には必ず `Content-Type: application/json` を付ける**(`; charset=utf-8` などの付け足しは可)
- 要求に **`Host` を付けない**か、付けるなら IP アドレス(`192.168.1.20:47821` `[fe80::1]:47821` など)にする。
  `Origin` と `Sec-Fetch-Site` は付けない
- ブラウザの中のページからの要求を止めるため、常駐アプリは上から外れる要求を、中身を見る前に断る
  - `Origin` か `Sec-Fetch-Site` がある、または `Host` が IP アドレスでない(名前): 403 `{"ok":false,"error":"forbidden"}`
  - POST の `Content-Type` が `application/json` でない・無い: 415 `{"ok":false,"error":"unsupported_media_type"}`
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
{"v":1,"state":"paired","nonce":"<32 hex>","session":true}
```

- 誰でも呼べる(認証なし)
- 本体はこれで「常駐アプリがいる」「組めるか」「次の要求の nonce」を知る
- `session` は、いま誰かがログインしていて、操作がログイン中の画面で動くか。`false` のとき
  (ログイン前など)は、操作は画面のないところで動き、`require_login` の操作は押せない
  (下の `login`・`needs_login`)

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
  {"id":"10_sleep","name":"スリープ","icon":"moon"},
  {"id":"40_shutdown","name":"シャットダウン","icon":"power","confirm":"開いているアプリはすべて閉じられます。\n保存していないデータは消えてしまいます。"},
  {"id":"50_Minecraft起動","name":"Minecraft起動","broken":true},
  {"id":"20_lock","name":"画面ロック","icon":"lock","login":true}
]}
```

- 応答には **`X-Kido-Sig: <HMAC(key, "list-ok\n" + nonce + "\n" + 本文のバイト列)>`** ヘッダを付ける。本体は本文を JSON として読む前に、生のバイト列で確かめる
- `id` はフォルダ名そのもの。`name` はフォルダ名から先頭の番号と `_` を除いたもの
  - `10_sleep` → `sleep`。同梱の操作は `kido.toml` の `name` で日本語名を付ける(下)
- `icon` は `kido.toml` の `icon`。無ければ省く
- `confirm` は `kido.toml` の `confirm`。無ければ省く。空文字なら `""` を送る(本文なしの確認)
- `broken` は設定が壊れていて押せない操作(実行できるファイルが無い・2つ以上ある・`kido.toml` が読めない)だけに `true` を付ける
- `login` は「いまは押せない(ログインが要る)」印。誰もログインしていない(`session` が
  `false`)ときに、`require_login` の操作へ付ける。ログイン中は付けない
- 並びはフォルダ名の昇順
- 失敗
  - nonce が無い・失効・署名違い: 401 `{"ok":false,"error":"unauthorized"}`
  - `paired` でない: 409 `{"ok":false,"error":"not_paired"}`

### `POST /v1/run`

要求: `{"nonce":"…","id":"10_sleep","sig":"<HMAC(key, "run\n" + nonce + "\n" + id)>"}`

- 200 `{"ok":true}` + `X-Kido-Sig: <HMAC(key, "run-ok\n" + nonce + "\n" + 本文のバイト列)>`
  - **応答を送り切ってから実行する**(少なくとも 300ms 待つ)。先に寝ると、本体に返事が届かない
- `kido.toml` に `wait = true` がある操作は、**実行して終わるまで応答を保留し**、終了コードで答える。
  署名はどれも上と同じ(`run-ok`、生の本文のバイト列に対して)
  - 終了コード 0: 200 `{"ok":true}`(待たない操作の成功と同じ)
  - 0 以外: 200 `{"ok":false,"err":"exit","code":3}`(`code` は終了コード)
  - 起動できなかった・終わりを見届けられなかった(手足役に渡せなかったなど): 200 `{"ok":false,"err":"start"}`
  - 常駐アプリは時間切れを持たない(どれだけ待つかは、スクリプトの側で決める)。本体が
    接続を切ったら、応答は捨てる(動かしたものは止めない)。本体は 3 秒の時間切れより
    長く待たないと、この応答を受け取れない
- 失敗
  - 401 unauthorized
  - 409 not_paired
  - 404 `{"ok":false,"error":"unknown_action"}`: その ID の操作が無い
  - 409 `{"ok":false,"error":"broken_action"}`: その操作の設定が壊れている
  - 409 `{"ok":false,"error":"needs_login"}`: ログインが要る操作を、誰もログインしていないときに押した(`login` の操作)

## 上限(常駐アプリが守る。本体も超えたら切り詰める)

| 項目 | 上限 |
|---|---|
| 操作の数 | 24 |
| `id` | 64 バイト(UTF-8)。`/` `\` 制御文字を含まない。`.` で始まらない |
| `name` | 32 文字 |
| `icon` | 40 文字、`[a-z0-9-]` のみ(違えば省く) |
| `confirm` | 200 文字 |

## 操作フォルダ(常駐アプリの中の話)

- `~/KidoButtons/<フォルダ>/` が 1 つの操作。`.` で始まるフォルダは無視する
- 実行できるファイル
  - Windows: `.ps1` `.bat` `.cmd` `.exe` `.lnk`
  - Mac: `.sh` `.command` `.app`
  - Linux: `.sh`、実行権限の付いたファイル、`.desktop`
- `kido.toml`(任意、UTF-8、BOM 可)
  - `name`(表示名。無ければフォルダ名から作る)
  - `icon`
  - `confirm`
  - `run`(実行するファイル名。フォルダの中だけ)
  - `require_login`(bool、既定 false)。true の操作は、誰もログインしていないときは
    一覧で `login` が付き、`run` すると `needs_login` になる。false(既定)の操作は、
    ログインしていなければ画面のないところで動く(窓のあるアプリは見えない)
  - `wait`(bool、既定 false)。true の操作は `run` で終わるまで待ち、終了コードで答える(上)。
    終了コードが取れる種類(Windows: `.ps1` `.bat` `.cmd` `.exe`、Mac: `.sh` `.command`、
    Linux: `.desktop` 以外)だけに付けられる。ショートカットやアプリに付けると `broken`
  - 知らないキーは `kido-agent check` で警告する(動作は止めない)
