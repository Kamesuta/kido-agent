#!/bin/sh
# 再起動する。インストール時に shutdown だけをパスワードなしで許す設定
# (/etc/sudoers.d/kido-agent)を入れていれば、そのまま再起動できる。
# 入れていなければ System Events に頼む。こちらは保存の確認を出したアプリに止められる。
sudo -n /sbin/shutdown -r now 2>/dev/null ||
	osascript -e 'tell application "System Events" to restart'
