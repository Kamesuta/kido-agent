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

# require_login = true の試験用の操作。
mkdir "$HOME/KidoButtons/90_sh"
printf 'touch "%s/kido-ci-sh"\n' "$HOME" >"$HOME/KidoButtons/90_sh/touch.sh"
printf 'require_login = true\n' >"$HOME/KidoButtons/90_sh/kido.toml"
# 設定なしの操作は、手足役がいなくても待ち受け役が動かす
mkdir "$HOME/KidoButtons/92_free"
printf 'touch "%s/kido-ci-free"\n' "$HOME" >"$HOME/KidoButtons/92_free/touch.sh"
python3 ci/hub_stub.py run 92_free || fail "設定なしの操作が手足役なしで動かない"
sleep 2
[ -f "$HOME/kido-ci-free" ] || fail "設定なしの操作がログインなしで動いていない"

# Mac・Linux では手足役は生まれない。ログイン中かは OS で決まる。Mac は常駐アプリ
# (LaunchAgent)が動いていればログイン中。Linux は logind に画面のあるログインが
# あるか(CI のランナーにあるとは限らないので、どちらでも筋が通るかを見る)。
if python3 ci/hub_stub.py hello | grep -q '"session": true'; then
	logged_in=1
else
	logged_in=0
fi
if [ "$(uname -s)" = Darwin ] && [ $logged_in = 0 ]; then
	fail "Mac では常駐アプリが動いていればログイン中のはず"
fi
if [ $logged_in = 1 ]; then
	# ログイン中なら、手足役がいなくても require_login の操作を待ち受け役が動かす
	python3 ci/hub_stub.py run 90_sh || fail "ログイン中なのに require_login の操作が動かない"
	sleep 2
	[ -f "$HOME/kido-ci-sh" ] || fail "require_login の操作が動いていない(手足役なし)"
	rm -f "$HOME/kido-ci-sh"
else
	python3 ci/hub_stub.py run 90_sh 2>&1 | grep -q needs_login || fail "ログインなしで needs_login にならない"
	[ -f "$HOME/kido-ci-sh" ] && fail "ログインなしで動いてしまった"
fi

# 2 つ目のプロセスは手足役になる(役の取り合いは Windows と同じ作り)。
# ログインしていない Linux では手足役を経て動き、ログイン中なら待ち受け役が動かす。
"$bin" serve >/tmp/kido-helper.log 2>&1 &
helper=$!
i=0
while [ $i -lt 50 ]; do
	grep -q '手足役として動きます' /tmp/kido-helper.log 2>/dev/null && break
	sleep 0.2
	i=$((i + 1))
done
sleep 1 # 手足役が 1 回目を取りに来るまで
python3 ci/hub_stub.py hello | grep -q '"session": true' || fail "手足役がいるのに session が false"

# 手足役経由で .sh が実際に動く
python3 ci/hub_stub.py list
python3 ci/hub_stub.py run 90_sh
sleep 2
[ -f "$HOME/kido-ci-sh" ] || fail ".sh が動いていない(手足役経由)"
# ログイン中(OS の決まり)なら待ち受け役が自分で動かし、手足役には流さない
if [ $logged_in = 0 ]; then
	grep -q '手足役: 実行しました 90_sh' /tmp/kido-helper.log || fail "手足役を経ていない"
else
	grep -q '手足役: 実行しました' /tmp/kido-helper.log && fail "ログイン中なのに手足役に流した"
fi
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
