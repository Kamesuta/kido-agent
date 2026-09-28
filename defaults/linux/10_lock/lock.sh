#!/bin/sh
# 画面をロックする。常駐アプリはログインのセッションの外で動いているので、
# セッションを名指ししないと loginctl がどれをロックするか決められない。
if [ -n "$XDG_SESSION_ID" ]; then
	exec loginctl lock-session
fi
exec loginctl lock-session "$(loginctl show-user "$(id -un)" -p Display --value)"
