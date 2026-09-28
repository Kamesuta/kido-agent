#!/bin/sh
# Mac・Linux でインストールから取り除くまでを通しで試す(CI 用)。
# 自動起動(launchd・systemd --user)と実際の実行を、実物で確かめる。
set -eu
tarball=$1
fail() { echo "失敗: $*" >&2; exit 1; }

python3 ci/hub_stub.py pair &
hub=$!
KIDO_AGENT_TARBALL=$tarball sh install/install.sh </dev/null
wait $hub || fail "本体の代わりが組めなかった"

bin="$HOME/.local/bin/kido-agent"
[ -x "$bin" ] || fail "実行ファイルが無い"
[ "$(ls "$HOME/Kido" | wc -l)" -eq 4 ] || fail "同梱の操作が 4 つ無い"

mkdir "$HOME/Kido/90_sh"
printf 'touch "%s/kido-ci-sh"\n' "$HOME" >"$HOME/Kido/90_sh/touch.sh"
python3 ci/hub_stub.py list
python3 ci/hub_stub.py run 90_sh
sleep 2
[ -f "$HOME/kido-ci-sh" ] || fail ".sh が動いていない"
"$bin" check || fail "check が失敗した"

# 更新(入れ直し)では組み直さない
KIDO_AGENT_TARBALL=$tarball sh install/install.sh </dev/null
python3 ci/hub_stub.py list || fail "入れ直しで鍵が消えた"

"$bin" uninstall
sleep 2
[ ! -e "$bin" ] || fail "実行ファイルが残っている"
curl -fsS -m 2 http://127.0.0.1:47821/v1/hello && fail "常駐アプリが止まっていない"
[ -d "$HOME/Kido" ] || fail "操作フォルダは残すはず"
echo "✓ 通し試験に通りました"
