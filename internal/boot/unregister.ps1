# 起動時タスク KidoAgent を消す(管理者で実行される)。
$ErrorActionPreference = 'SilentlyContinue'
[Console]::OutputEncoding = [Text.Encoding]::UTF8
# 名指しで消す。無ければ何もしない(Get-ScheduledTask のワイルドカードは使わない)。
Unregister-ScheduledTask -TaskName 'KidoAgent' -Confirm:$false
exit 0
