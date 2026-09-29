#!/bin/sh
# 配る物(各 OS の zip・tar.gz と SHA256SUMS)を dist/ に作る。
# リリースの CI と、手元から配布のページを出すときの両方で使う(作り方を 2 か所に書かない)。
#   sh ci/build-dist.sh <版>
set -eu
version=$1
out=dist
rm -rf "$out"
mkdir -p "$out"

# Windows は常駐用に窓の無い exe も要るので、専用の手順で zip にする
for arch in amd64 arm64; do
	pwsh -NoProfile -File ci/package-windows.ps1 -Version "$version" -Arch "$arch" -Out "$out"
done

for t in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64; do
	os=${t%/*}
	arch=${t#*/}
	stage=$out/$os-$arch
	mkdir -p "$stage"
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
		-ldflags "-s -w -X main.version=$version" -o "$stage/kido-agent" ./cmd/kido-agent
	tar -czf "$out/kido-agent-$os-$arch.tar.gz" -C "$stage" kido-agent
done

# Mac には sha256sum が無いことがある。出力の形は同じ
cd "$out"
if command -v sha256sum >/dev/null 2>&1; then
	sha256sum ./*.zip ./*.tar.gz | sed 's# \./# #' >SHA256SUMS
else
	shasum -a 256 ./*.zip ./*.tar.gz | sed 's# \./# #' >SHA256SUMS
fi
cat SHA256SUMS
