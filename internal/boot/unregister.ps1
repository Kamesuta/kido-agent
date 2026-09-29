# 起動時タスク KidoAgent を消す(管理者で実行される)。
# Unregister-ScheduledTask はこの PC の別タスクの壊れた XML で例外になる実績があるので、
# CIM を通さない schtasks で名指しで消す。無ければ何もしない(exit 0)。
[Console]::OutputEncoding = [Text.Encoding]::UTF8
cmd /c "schtasks /delete /tn KidoAgent /f >nul 2>nul"
exit 0
