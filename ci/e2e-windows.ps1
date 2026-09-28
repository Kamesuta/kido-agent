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
$kido = Join-Path $env:USERPROFILE 'Kido'
$uninstall = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\KidoAgent'

# 途中で落ちても常駐アプリを残さない。残ると CI の手順が出力の管を握られて終わらなくなる。
try {
    # pair の待ちを本体の代わりに終わらせる
    $hub = Start-Process python -ArgumentList 'ci/hub_stub.py', 'pair' -PassThru -NoNewWindow
    $null = $hub.Handle # 取っておかないと、終わった後に ExitCode が読めない
    $env:KIDO_AGENT_ZIP = (Resolve-Path $Zip).Path
    Write-Host '== インストール'
    Get-Content -Raw -Encoding UTF8 install/install.ps1 | Invoke-Expression
    $hub.WaitForExit()
    Assert ($hub.ExitCode -eq 0) '本体の代わりが組めなかった'

    Assert (Test-Path "$dest\kido-agentd.exe") 'インストール先に exe が無い'
    Assert ((Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run').KidoAgent) '自動起動が登録されていない'
    Assert ((Get-ItemProperty $uninstall).DisplayName -eq '起動丸エージェント') '「アプリ」一覧に無い'
    Assert (Test-Path "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\起動丸の操作フォルダ.lnk") 'ショートカットが無い'
    Assert ((Get-ChildItem $kido -Directory).Count -eq 4) '同梱の操作が 4 つ無い'

    # .bat(ShellExecute)と .ps1(窓なしの PowerShell)を実際に動かす
    New-Item -ItemType Directory "$kido\90_bat", "$kido\91_ps1" | Out-Null
    Set-Content "$kido\90_bat\touch.bat" "echo ok> `"%TEMP%\kido-ci-bat.txt`"" -Encoding ASCII
    Set-Content "$kido\91_ps1\touch.ps1" "Set-Content `"`$env:TEMP\kido-ci-ps1.txt`" ok" -Encoding ASCII
    python ci/hub_stub.py list; Assert ($LASTEXITCODE -eq 0) 'list'
    python ci/hub_stub.py run 90_bat; Assert ($LASTEXITCODE -eq 0) 'run 90_bat'
    python ci/hub_stub.py run 91_ps1; Assert ($LASTEXITCODE -eq 0) 'run 91_ps1'
    Start-Sleep 5
    Assert (Test-Path "$env:TEMP\kido-ci-bat.txt") '.bat が動いていない'
    Assert (Test-Path "$env:TEMP\kido-ci-ps1.txt") '.ps1 が動いていない'

    & "$dest\kido-agent.exe" check
    Assert ($LASTEXITCODE -eq 0) 'check が失敗した'

    # 更新(入れ直し)では組み直さない
    Get-Content -Raw -Encoding UTF8 install/install.ps1 | Invoke-Expression
    python ci/hub_stub.py list; Assert ($LASTEXITCODE -eq 0) '入れ直しで鍵が消えた'

    & "$dest\kido-agent.exe" uninstall
    Start-Sleep 5
    Assert (-not (Test-Path $uninstall)) '「アプリ」一覧に残っている'
    Assert (-not (Get-Process kido-agentd -ErrorAction SilentlyContinue)) '常駐アプリが止まっていない'
    Assert (-not (Test-Path $dest)) 'インストール先が残っている'
    Assert (Test-Path $kido) '~/Kido は残すはず'
    Write-Host '✓ Windows の通し試験に通りました'
} finally {
    Get-Process kido-agentd -ErrorAction SilentlyContinue | Stop-Process -Force
    Write-Host '--- ログ ---'
    Get-Content -Encoding UTF8 "$env:APPDATA\kido-agent\kido-agent.log" -ErrorAction SilentlyContinue
}
