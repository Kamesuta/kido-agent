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
[ "$(find "$HOME/KidoButtons" -mindepth 1 -maxdepth 1 -type d | wc -l)" -eq 4 ] || fail "同梱の操作が 4 つ無い"
[ -f "$HOME/KidoButtons/使いかた.txt" ] || fail "使いかた.txt が無い"

# before_login なしの試験用の操作。手足役がいないと動かせないことをまず確かめる。
mkdir "$HOME/KidoButtons/90_sh"
printf 'touch "%s/kido-ci-sh"\n' "$HOME" >"$HOME/KidoButtons/90_sh/touch.sh"
python3 ci/hub_stub.py hello | grep -q '"session": false' || fail "手足役がいないのに session が true"
python3 ci/hub_stub.py run 90_sh 2>&1 | grep -q needs_login || fail "手足役なしで needs_login にならない"
[ -f "$HOME/kido-ci-sh" ] && fail "ログインなしで動いてしまった"

# 手足役(2 つ目)を起こす。これがログイン中の役の代わり。
"$bin" serve >/tmp/kido-helper.log 2>&1 &
helper=$!
i=0
while [ $i -lt 50 ]; do
	python3 ci/hub_stub.py hello 2>/dev/null | grep -q '"session": true' && break
	sleep 0.2
	i=$((i + 1))
done
python3 ci/hub_stub.py hello | grep -q '"session": true' || fail "手足役がいるのに session が false"

# 手足役経由で .sh が実際に動く
python3 ci/hub_stub.py list
python3 ci/hub_stub.py run 90_sh
sleep 2
[ -f "$HOME/kido-ci-sh" ] || fail ".sh が動いていない(手足役経由)"
kill "$helper" 2>/dev/null || true
"$bin" check || fail "check が失敗した"

# 更新(入れ直し)では組み直さない
KIDO_AGENT_TARBALL=$tarball sh install/install.sh </dev/null
python3 ci/hub_stub.py list || fail "入れ直しで鍵が消えた"

"$bin" uninstall
sleep 2
[ ! -e "$bin" ] || fail "実行ファイルが残っている"
curl -fsS -m 2 http://127.0.0.1:47821/v1/hello && fail "常駐アプリが止まっていない"
[ -d "$HOME/KidoButtons" ] || fail "操作フォルダは残すはず"
echo "✓ 通し試験に通りました"
