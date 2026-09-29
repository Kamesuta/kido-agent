#!/bin/sh
# マイクラのサーバーを tmux の中で起動する。あとから `tmux attach -t mc` で中を見られる。
# 下の2行を、自分のサーバーに合わせて書き換えてください
SERVER_DIR="$HOME/minecraft"
START="java -Xmx4G -jar server.jar nogui"

# 常駐アプリから起動すると PATH が短く、Homebrew の tmux や java が見つからないため
PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"
export PATH

cd "$SERVER_DIR" || exit 1
# 2回押しても2つ立ち上がらないように、動いていれば何もしない
tmux has-session -t mc 2>/dev/null && exit 0

if command -v systemd-run >/dev/null 2>&1; then
  # Linux: 常駐アプリを更新・再起動したときに、サーバーが道連れで止まらないよう別の枠で起こす
  exec systemd-run --user --scope --quiet tmux new-session -d -s mc "$START"
fi
exec tmux new-session -d -s mc "$START"
