#!/bin/sh
# スリープにする。画面の前に座っている利用者なら、管理者の権限なしで使える。
exec systemctl suspend
