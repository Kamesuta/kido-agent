#!/bin/sh
# シャットダウンする。-i は、止めるのを引き止めるアプリがあっても進めるため
# (押す前にスマホで確認を出している)。
exec systemctl poweroff -i
