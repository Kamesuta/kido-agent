# 起動時タスク KidoAgent を登録する(管理者で実行される)。
# ログインしていなくても、電源を入れただけで待ち受け役が動くようにするため。
# S4U(パスワードの保存なしで本人として動く)・起動時・時間制限なし・電池でも止めない。
param([Parameter(Mandatory)][string]$Exe, [Parameter(Mandatory)][string]$User)
$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [Text.Encoding]::UTF8
$name = 'KidoAgent'

$action = New-ScheduledTaskAction -Execute $Exe
# 起動直後は混むので少し遅らせる。ネットワークが立ち上がる前だと本体とつながれない。
$trigger = New-ScheduledTaskTrigger -AtStartup
$trigger.Delay = 'PT30S'
# RunLevel は Limited を頼む(実際には High で動くが、プロセス側で権限を下げる)。
$principal = New-ScheduledTaskPrincipal -UserId $User -LogonType S4U -RunLevel Limited
$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries `
    -ExecutionTimeLimit ([TimeSpan]::Zero) `
    -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1) `
    -StartWhenAvailable
$task = New-ScheduledTask -Action $action -Trigger $trigger -Principal $principal -Settings $settings `
    -Description '起動丸エージェント: ログイン前でも操作を受け取れるようにします'
# 名指しで登録・置き換え(ワイルドカードは壊れた別タスクの XML で例外になる実績があるため使わない)。
Register-ScheduledTask -TaskName $name -InputObject $task -Force | Out-Null
Start-ScheduledTask -TaskName $name
