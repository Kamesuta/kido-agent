#!/bin/sh
# サーバーのコンソールに stop を打ち、ワールドを保存してから止める。
# 強制終了はワールドが壊れることがあるので使わない
PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"
export PATH

# tmux が無いと、サーバーが動いているかも確かめられない。「止まった」と返さず、
# 理由を残して失敗としてスマホに知らせる
if ! command -v tmux >/dev/null 2>&1; then
  echo "tmux が見つかりません。Mac は brew install tmux、Ubuntu は sudo apt install tmux で入れてください" >&2
  exit 1
fi

# もう止まっていれば何もしない
tmux has-session -t mc 2>/dev/null || exit 0
# 中を見たまま離れてスクロール中だったり、打ちかけの文字があったりすると stop が届かない
tmux send-keys -t mc -X cancel 2>/dev/null
tmux send-keys -t mc C-u
tmux send-keys -t mc stop Enter

# 12 秒待っても止まらなければ、失敗としてスマホに知らせる(サーバーは止めずに残す)
i=0
while tmux has-session -t mc 2>/dev/null; do
  i=$((i + 1))
  [ "$i" -ge 12 ] && exit 1
  sleep 1
done
