# 起動丸エージェント(kido-agent)

起動丸(Kido)で、スマホからこの PC をスリープ・画面ロック・再起動・シャットダウンできるようにする常駐アプリです。
ゲームの起動など、自分で作った操作も足せます。
Windows・Mac・Linux に対応しています(Linux は Ubuntu で動作確認)。

案内のページ: https://pc.kido.page

## インストール

管理者の権限は要りません。入れ直すと更新になります(組み合わせはそのまま)。

Windows(「ターミナル」または「Windows PowerShell」に貼り付け):

```powershell
irm https://pc.kido.page/install.ps1 | iex
```

Mac・Linux(ターミナルに貼り付け):

```sh
curl -fsSL https://pc.kido.page/install.sh | sh
```

手順の中で次のことが起きます。

1. 常駐アプリを入れ、ログイン時に自動で起動するようにする
   - Windows: `%LOCALAPPDATA%\Programs\kido-agent\`、スタートメニューに「起動丸の操作フォルダ」、「アプリ」一覧に「起動丸エージェント」
   - Mac: `~/.local/bin/kido-agent` と LaunchAgent(`page.kido.agent`)。再起動・シャットダウン用に `/sbin/shutdown` だけをパスワードなしで許す設定(`/etc/sudoers.d/kido-agent`)を入れるか聞く
   - Linux: `~/.local/bin/kido-agent` と `systemd --user` の `kido-agent.service`
2. ファイアウォールの確認(Windows は「Windows セキュリティ」)が出たら「許可」を押す
3. `kido-agent pair` が動き、起動丸の本体が見つけに来るのを 10 分のあいだ待つ。起動ページにこの PC をまだ登録していなければ、そのあいだにスマホで登録する
4. 操作フォルダ `~/KidoButtons` を開く

### 公開前(リポジトリが非公開のあいだ)の入れ方

リリースから落とせないので、手元に置いた zip・tar.gz を指定します。

```powershell
# Windows(Windows PowerShell 5.1 は BOM の無い .ps1 を -File で読むと日本語が化けるので、UTF-8 と指定して読ませる)
$env:KIDO_AGENT_ZIP = "C:\path\kido-agent-windows-amd64.zip"   # 展開したフォルダでもよい
Get-Content -Raw -Encoding UTF8 .\install\install.ps1 | iex
```

```sh
# Mac・Linux
KIDO_AGENT_TARBALL=/path/kido-agent-darwin-arm64.tar.gz sh install/install.sh
```

## 操作フォルダ

`~/KidoButtons/<フォルダ>/` が 1 つの操作です。最初の起動で、スリープ・画面ロック・再起動・シャットダウンの 4 つを置きます
(`~/KidoButtons` そのものが無いときだけ。消したものは戻しません)。
フォルダの使いかたを書いた `使いかた.txt` も一緒に置きます(Windows ではメモ帳で開けるよう BOM 付き・CRLF)。

- フォルダ名が操作の ID で、表示名はフォルダ名から先頭の番号と `_` を除いたもの(`50_マイクラ` → `マイクラ`)。並びはフォルダ名の順
- 同梱の操作は `10_sleep` `20_lock` `30_restart` `40_shutdown`。10 刻みにしてあるので、間に入れたければ `15_` のように番号を選ぶ
- フォルダに実行できるファイルを **1 つだけ** 置く
  - Windows: `.lnk` `.bat` `.cmd` `.ps1` `.exe`(`.ps1` は窓を出さずに動かし、ほかはダブルクリックと同じ)
  - Mac: `.app` `.sh` `.command`(`.sh` は窓を出さずに動かし、ほかは `open`)
  - Linux: `.sh` `.desktop` と、実行権限の付いたファイル
- `.` で始まるフォルダは無視する
- 本体からの要求のたびに読み直すので、足したり直したりしても再起動は要らない
- 一覧に載るのは 24 個まで。フォルダ名は 64 バイトまで

### kido.toml

任意。UTF-8(BOM 付きでもよい)。

| キー | 意味 |
|---|---|
| `name` | 表示名(32 文字まで) |
| `icon` | アイコン名。[lucide](https://lucide.dev/icons) の名前(英小文字・数字・`-`、40 文字まで)。違えば省く |
| `confirm` | 押したときの確認の文章(200 文字まで)。`""` なら文章なしで確認だけ出す |
| `run` | 実行するファイル名。2 つ以上あるときに選ぶ。フォルダの中のファイル名だけ |
| `require_login` | `true` にすると、ログインしているときだけ押せる(ログイン前は押せない)。窓の要るアプリ向け。既定 `false` |

```toml
name = "マイクラ起動"
icon = "gamepad-2"
confirm = "本当に起動しますか?"
run = "start.bat"
```

知らないキーは `kido-agent check` で警告します(動作は止めません)。
実行できるファイルが無い・2 つ以上ある・`kido.toml` が読めないときは、スマホに押せない操作として出ます。

## ログイン前(電源を入れただけ)でも使う

`kido-agent boot on` にすると、サインインしていなくても操作を受け取れるようになります
(電源を入れてゲームのサーバーを立ち上げる、スリープ・再起動・シャットダウンなど)。
`kido-agent boot off` で元に戻します。引数なしの `kido-agent boot` で今の状態が分かります。
インストールの最後にも尋ねます。

- Windows: 起動時に動くタスクを登録します(登録には管理者の確認 UAC が出ます)。
  サインイン中はこれまでどおり、ログイン中の画面・窓で操作が動きます
- Linux(Ubuntu で動作確認): `loginctl enable-linger` を使います
- Mac: 起動時のディスクの暗号化(FileVault)がパスワードを求めるため、ログイン前には動かせません

ログインしていない間に押した操作は、画面のないところで動きます。スクリプトやサーバーは
そのまま動きますが、**窓のあるアプリは見えないところで起動します**。窓が要る操作は
`kido.toml` に `require_login = true` を書くと、ログインしているときだけ押せるようになります
(ログイン前はスマホで「ログインが必要」と出ます)。同梱の画面ロックはこの設定です。
ショートカットやアプリなのに `require_login` が無い操作は、`kido-agent check` が知らせます。

## コマンド

| コマンド | 内容 |
|---|---|
| `kido-agent serve` | 常駐する(普段は自動で起動するので打たない) |
| `kido-agent pair` | 本体と組む。10 分だけ受け付ける。組み直すと古い組み合わせは使えなくなる |
| `kido-agent check` | 操作フォルダを確かめ、✓ / ✗(押せない。理由と直し方)/ !(警告)で表示する。組めているかも出す |
| `kido-agent open` | 操作フォルダを開く |
| `kido-agent boot` | ログイン前(電源を入れただけ)でも使えるようにする(`on`/`off`、引数なしで状態) |
| `kido-agent uninstall` | 取り除く。`~/KidoButtons` は残す |
| `kido-agent version` | 版を出す |

鍵とログの置き場: どの OS も `~/.kido-agent/`(Windows では隠しフォルダ)(ログは `kido-agent.log`、1MB で 1 世代回す)。

## 開発

本体とのやり取りは [PROTOCOL.md](PROTOCOL.md) が取り決めです。これに合わせて作ります。

| 場所 | 中身 |
|---|---|
| `cmd/kido-agent/` | 入口とコマンド(pair・check・open・uninstall) |
| `internal/auth/` | 鍵・nonce・署名と、組める時間 |
| `internal/actions/` | 操作フォルダを読む・kido.toml・上限・一覧の本文 |
| `internal/server/` | 本体からの要求と CLI からの要求を受ける常駐の本体 |
| `internal/control/` | CLI から常駐アプリへの問い合わせ |
| `internal/launch/` | OS ごとの実行のしかた |
| `internal/defaults/` | 同梱の操作(`data/<os>/`)と書き出し |
| `internal/uninstall/` | OS ごとの片付け |
| `internal/paths/` `internal/logfile/` | 置き場所とログ |
| `install/` | インストールの手順(`install.ps1` `install.sh`) |
| `ci/` `site/` | CI の道具と、案内のページ |

```sh
go vet ./...
go test ./...
# 全部の OS・CPU 向け(cgo は使わない)
for t in windows/amd64 windows/arm64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64; do
  GOOS=${t%/*} GOARCH=${t#*/} CGO_ENABLED=0 go build ./...
done
# Windows の zip(PowerShell 7 ならどの OS でも)
pwsh ci/package-windows.ps1 -Version dev -Arch amd64 -Out dist
```

- Windows では同じコードから exe を 2 つ作る。`kido-agentd.exe` は GUI 版(`-H=windowsgui`)で、ログオン時に黒い窓を出さずに常駐する。
  引数なしで起動すると `serve` になる。`kido-agent.exe` はコンソール版で、PowerShell は GUI 版の終わりを待たず出力も受け取らないので、CLI にはこちらを使う
- 動いている常駐アプリの隣で試すときは、待ち受けを `KIDO_AGENT_API_ADDR` `KIDO_AGENT_CONTROL_ADDR`、鍵とログの置き場を `KIDO_AGENT_CONFIG_DIR` で変えられる
- `ci/hub_stub.py` は本体の代わりをする試験用の道具。`pair` `list` `run <id>` を送る
- CI(`.github/workflows/ci.yml`)は 3 つの OS でテストし、Windows・Mac・Linux ではインストールから取り除くまでを実物で通す
- 同梱の操作を入れ直したい(新しい同梱の操作を試したい)ときは、`~/KidoButtons` を消してから `kido-agent serve` を起動する。
  同梱の操作は `~/KidoButtons` そのものが無いときだけ書き出すので、あるうちは何度起動しても変わらない

### リリース

`v` で始まるタグを push すると、`release.yml` が 6 つの成果物(Windows は zip、Mac・Linux は tar.gz)と `SHA256SUMS` を GitHub Releases に上げます。

```sh
git tag v0.1.0 && git push origin v0.1.0
```

### 案内のページ

`site/` を GitHub Pages(`pc.kido.page`)に出します。出すときに `install/` の `install.ps1` `install.sh` を写すので、
`https://pc.kido.page/install.ps1` が手順の本物と同じになります。
リポジトリが非公開のあいだは、無料の契約では Pages を使えないため `pages.yml` は失敗します(公開後に通ります)。
