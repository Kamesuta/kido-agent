# Windows でインストールから取り除くまでを通しで試す(CI 用)。
# レジストリ・スタートメニュー・GUI 版の常駐・ShellExecute での実行は、
# 手元の Mac では確かめられないので、ここで実物を動かして確かめる。
# Windows PowerShell 5.1 で動かすため、BOM の無いこのファイルは
#   Get-Content -Raw -Encoding UTF8 ci/e2e-windows.ps1 | Invoke-Expression
# で読ませる(-File だと日本語が化ける)。そのため引数ではなく環境変数で受ける。
$Zip = $env:KIDO_CI_ZIP
$ErrorActionPreference = 'Stop'
function Assert($cond, $msg) { if (-not $cond) { throw "失敗: $msg" } }

$dest = Join-Path $env:LOCALAPPDATA 'Programs\kido-agent'
$kido = Join-Path $env:USERPROFILE 'KidoButtons'
$uninstall = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\KidoAgent'

# 途中で落ちても常駐アプリを残さない。残ると CI の手順が出力の管を握られて終わらなくなる。
try {
    # pair の待ちを本体の代わりに終わらせる
    $hub = Start-Process python -ArgumentList 'ci/hub_stub.py', 'pair' -PassThru -NoNewWindow
    $null = $hub.Handle # 取っておかないと、終わった後に ExitCode が読めない
    $env:KIDO_AGENT_ZIP = (Resolve-Path $Zip).Path
    $env:KIDO_AGENT_BOOT = 'skip' # 通しの試験では、ログイン前対応は後で明示的に試す
    Write-Host '== インストール'
    Get-Content -Raw -Encoding UTF8 install/install.ps1 | Invoke-Expression
    $hub.WaitForExit()
    Assert ($hub.ExitCode -eq 0) '本体の代わりが組めなかった'

    Assert (Test-Path "$dest\kido-agentd.exe") 'インストール先に exe が無い'
    Assert ((Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run').KidoAgent) '自動起動が登録されていない'
    Assert ((Get-ItemProperty $uninstall).DisplayName -eq '起動丸エージェント') '「アプリ」一覧に無い'
    Assert (Test-Path "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\起動丸の操作フォルダ.lnk") 'ショートカットが無い'
    Assert ((Get-ChildItem $kido -Directory).Count -eq 4) '同梱の操作が 4 つ無い'
    Assert (Test-Path "$kido\使いかた.txt") '使いかた.txt が無い'
    Assert ((Get-Item -Force "$env:USERPROFILE\.kido-agent").Attributes -band [IO.FileAttributes]::Hidden) '鍵の置き場が隠れていない'

    # 試験用の操作(before_login なし)。手足役がいないと動かせないことを先に確かめる。
    New-Item -ItemType Directory "$kido\90_bat", "$kido\91_ps1" | Out-Null
    Set-Content "$kido\90_bat\touch.bat" "echo ok> `"%TEMP%\kido-ci-bat.txt`"" -Encoding ASCII
    Set-Content "$kido\91_ps1\touch.ps1" "Set-Content `"`$env:TEMP\kido-ci-ps1.txt`" ok" -Encoding ASCII

    # 手足役はまだいない(待ち受け役だけ)。session は false、before_login でない操作は
    # login が付き、run は needs_login で断られる。
    (python ci/hub_stub.py hello) | Tee-Object -Variable helloOut | Write-Host
    Assert ($helloOut -match '"session": false') '手足役がいないのに session が true'
    (python ci/hub_stub.py list) | Tee-Object -Variable listOut | Write-Host
    Assert ($listOut -match '"id":"90_bat".*"login":true') 'login の印が無い'
    (python ci/hub_stub.py run 90_bat) | Tee-Object -Variable runOut | Write-Host
    Assert ($runOut -match 'needs_login') '手足役なしで needs_login にならない'
    Assert (-not (Test-Path "$env:TEMP\kido-ci-bat.txt")) 'ログインなしで動いてしまった'

    # 手足役(2 つ目の kido-agentd)を起こす。これはログイン中の画面を持つ役。
    Start-Process -FilePath "$dest\kido-agentd.exe" -WorkingDirectory $dest
    foreach ($i in 1..50) {
        if ((python ci/hub_stub.py hello) -match '"session": true') { break }
        Start-Sleep -Milliseconds 200
    }
    (python ci/hub_stub.py hello) | Tee-Object -Variable helloOut | Write-Host
    Assert ($helloOut -match '"session": true') '手足役がいるのに session が false'

    # 手足役経由で、.bat(ShellExecute)と .ps1(窓なしの PowerShell)が実際に動く。
    python ci/hub_stub.py list; Assert ($LASTEXITCODE -eq 0) 'list'
    python ci/hub_stub.py run 90_bat; Assert ($LASTEXITCODE -eq 0) 'run 90_bat'
    python ci/hub_stub.py run 91_ps1; Assert ($LASTEXITCODE -eq 0) 'run 91_ps1'
    Start-Sleep 5
    Assert (Test-Path "$env:TEMP\kido-ci-bat.txt") '.bat が動いていない(手足役経由)'
    Assert (Test-Path "$env:TEMP\kido-ci-ps1.txt") '.ps1 が動いていない(手足役経由)'

    & "$dest\kido-agent.exe" check
    Assert ($LASTEXITCODE -eq 0) 'check が失敗した'

    # ログイン前対応(S4U 起動時タスク)。CI ランナーは管理者なので UAC は素通りする想定。
    # 環境によって昇格できないときは、登録の確認だけ省いて先へ進む。
    Write-Host '== ログイン前対応(boot on)'
    # ここまでの待ち受け役・手足役をいったん止めて、素の状態から試す。
    Get-Process kido-agentd -ErrorAction SilentlyContinue | Stop-Process -Force
    Start-Sleep 2
    & "$dest\kido-agent.exe" boot on 2>&1 | Write-Host
    Start-Sleep 5
    & schtasks /query /tn KidoAgent *> $null
    if ($LASTEXITCODE -eq 0) {
        Write-Host 'S4U タスクの登録を確認'
        $log = Get-Content -Encoding UTF8 "$env:USERPROFILE\.kido-agent\kido-agent.log" -Raw
        Assert ($log -match '普通のユーザー') 'ログに権限降格が見えない'
        & "$dest\kido-agent.exe" boot off
        Start-Sleep 3
        & schtasks /query /tn KidoAgent *> $null
        Assert ($LASTEXITCODE -ne 0) 'boot off で起動時タスクが消えない'
        Write-Host '✓ ログイン前対応(S4U・権限降格)を確認'
    } else {
        Write-Host '! この環境では起動時タスクを登録できませんでした(昇格不可のためスキップ)'
    }

    # 更新(入れ直し)では組み直さない
    Get-Content -Raw -Encoding UTF8 install/install.ps1 | Invoke-Expression
    python ci/hub_stub.py list; Assert ($LASTEXITCODE -eq 0) '入れ直しで鍵が消えた'

    & "$dest\kido-agent.exe" uninstall
    Start-Sleep 5
    Assert (-not (Test-Path $uninstall)) '「アプリ」一覧に残っている'
    Assert (-not (Get-Process kido-agentd -ErrorAction SilentlyContinue)) '常駐アプリが止まっていない'
    Assert (-not (Test-Path $dest)) 'インストール先が残っている'
    Assert (Test-Path $kido) '~/KidoButtons は残すはず'
    Write-Host '✓ Windows の通し試験に通りました'
} finally {
    & schtasks /end /tn KidoAgent *> $null
    & schtasks /delete /tn KidoAgent /f *> $null
    Get-Process kido-agentd -ErrorAction SilentlyContinue | Stop-Process -Force
    Write-Host '--- ログ ---'
    Get-Content -Encoding UTF8 "$env:USERPROFILE\.kido-agent\kido-agent.log" -ErrorAction SilentlyContinue
}
