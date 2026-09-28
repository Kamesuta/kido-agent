#!/bin/sh
# シャットダウンする。インストール時に shutdown だけをパスワードなしで許す設定
# (/etc/sudoers.d/kido-agent)を入れていれば、そのまま切れる。
# 入れていなければ System Events に頼む。こちらは保存の確認を出したアプリに止められる。
sudo -n /sbin/shutdown -h now 2>/dev/null ||
	osascript -e 'tell application "System Events" to shut down'
