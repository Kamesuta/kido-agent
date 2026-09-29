#!/bin/sh
# 配布のページ(案内・入れ方のスクリプト・配る物)を Cloudflare Workers に出す。
# 先に ci/build-dist.sh で dist/ を作っておく。
#   sh ci/deploy-site.sh                  → pc.kido.page
#   sh ci/deploy-site.sh <ドメイン> <名前>  → 試験用の出し先(ドメインはリポジトリに書かない)
# 出すのはリリースのときだけ。ページだけを出すと、配る物が消える。
set -eu
domain=${1:-pc.kido.page}
name=${2:-kido-agent-site}
base=https://$domain
out=.site

for f in dist/SHA256SUMS; do
	[ -f "$f" ] || { echo "先に ci/build-dist.sh で dist/ を作ってください" >&2; exit 1; }
done

rm -rf "$out"
mkdir -p "$out/dl"
cp site/index.html site/app.js site/style.css site/_headers "$out/"
cp install/install.ps1 install/install.sh "$out/"
# 入れ方の本物は install/ に 1 つだけ置き、出し先の URL だけをここで書き換える。
# スクリプトが自分を落とし直すとき・配る物を落とすときに、出した場所を向くように
for f in index.html install.ps1 install.sh; do
	sed "s#https://pc\.kido\.page#$base#g" "$out/$f" >"$out/$f.tmp"
	mv "$out/$f.tmp" "$out/$f"
done
cp dist/*.zip dist/*.tar.gz dist/SHA256SUMS "$out/dl/"

npx --yes wrangler@4 deploy --name "$name" --domain "$domain"
