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
$ok = $false
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

    # 試験用の操作。90・91 は require_login = true(手足役がいないと動かせない)、
    # 92 は設定なし(ログインしていなくても待ち受け役が画面なしで動かす)。
    New-Item -ItemType Directory "$kido\90_bat", "$kido\91_ps1", "$kido\92_free" | Out-Null
    Set-Content "$kido\90_bat\touch.bat" "echo ok> `"%TEMP%\kido-ci-bat.txt`"" -Encoding ASCII
    Set-Content "$kido\91_ps1\touch.ps1" "Set-Content `"`$env:TEMP\kido-ci-ps1.txt`" ok" -Encoding ASCII
    Set-Content "$kido\92_free\touch.bat" "echo ok> `"%TEMP%\kido-ci-free.txt`"" -Encoding ASCII
    Set-Content "$kido\90_bat\kido.toml" "require_login = true" -Encoding ASCII
    Set-Content "$kido\91_ps1\kido.toml" "require_login = true" -Encoding ASCII

    # 手足役はまだいない(待ち受け役だけ)。session は false、require_login の操作は
    # login が付き、run は needs_login で断られる。設定なしの操作は動く。
    (python ci/hub_stub.py hello) | Tee-Object -Variable helloOut | Write-Host
    Assert ($helloOut -match '"session": false') '手足役がいないのに session が true'
    (python ci/hub_stub.py list) | Tee-Object -Variable listOut | Write-Host
    Assert ($listOut -match '"id":"90_bat".*"login":true') 'login の印が無い'
    (python ci/hub_stub.py run 90_bat) | Tee-Object -Variable runOut | Write-Host
    Assert ($runOut -match 'needs_login') '手足役なしで needs_login にならない'
    Assert (-not (Test-Path "$env:TEMP\kido-ci-bat.txt")) 'ログインなしで動いてしまった'
    Assert (-not ($listOut -match '"id":"92_free"[^}]*"login":true')) '設定なしの操作に login が付いた'
    python ci/hub_stub.py run 92_free; Assert ($LASTEXITCODE -eq 0) 'run 92_free(ログインなし)'
    Start-Sleep 3
    Assert (Test-Path "$env:TEMP\kido-ci-free.txt") '設定なしの操作がログインなしで動いていない'
    # wait = true の操作は、終わるまで待って終了コードで答える(ログインなし=待ち受け役)。
    New-Item -ItemType Directory "$kido\93_wait" | Out-Null
    Set-Content "$kido\93_wait\fail.bat" "@echo off`r`nexit /b 3" -Encoding ASCII
    Set-Content "$kido\93_wait\kido.toml" "wait = true" -Encoding ASCII
    (python ci/hub_stub.py run 93_wait) | Tee-Object -Variable waitOut | Write-Host
    Assert ($waitOut -match '"code":3') 'wait の操作が終了コードを返さない(ログインなし)'
    # & ( ) と空白の入ったフォルダでも、cmd が区切りと読み違えずに動かす。
    $odd = "$kido\94_a&b (1)"
    New-Item -ItemType Directory -Path $odd | Out-Null
    Set-Content -LiteralPath "$odd\fail.bat" "@echo off`r`nexit /b 4" -Encoding ASCII
    Set-Content -LiteralPath "$odd\kido.toml" "wait = true" -Encoding ASCII
    (python ci/hub_stub.py run '94_a&b (1)') | Tee-Object -Variable oddOut | Write-Host
    Assert ($oddOut -match '"code":4') '& ( ) の入ったフォルダの wait が動かない'

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
    (python ci/hub_stub.py run 93_wait) | Tee-Object -Variable waitOut | Write-Host
    Assert ($waitOut -match '"code":3') 'wait の操作が終了コードを返さない(手足役経由)'

    & "$dest\kido-agent.exe" check
    Assert ($LASTEXITCODE -eq 0) 'check が失敗した'

    # このランナーの kido-agentd は昇格して動くので、権限降格の道が通っているはず。
    # 目印付きの子が立ち上がっていること(=無限ループになっていないこと)を確かめる。
    $plog = Get-Content -Raw -Encoding UTF8 "$env:USERPROFILE\.kido-agent\kido-agent.log"
    if ($plog -match '管理者だったので') {
        Assert ($plog -match '起動し直し済み') '権限降格の子が目印付きで起動していない(ループ防止)'
        Write-Host '✓ 権限降格(一度きり・普通のユーザーで起動し直し)を確認'
    } else {
        Write-Host '! この環境では昇格していないため、権限降格の確認は省きます'
    }

    # ログイン前対応(S4U 起動時タスク)。CI ランナーは管理者なので、boot on は
    # RunAs を挟まず同じプロセスで登録する。実際に登録→起動→権限降格→boot off まで通す。
    Write-Host '== ログイン前対応(boot on)'
    # ここまでの待ち受け役・手足役をいったん止めて、素の状態から試す。
    Get-Process kido-agentd -ErrorAction SilentlyContinue | Stop-Process -Force
    Start-Sleep 2
    & "$dest\kido-agent.exe" boot on 2>&1 | Write-Host
    Start-Sleep 6
    cmd /c "schtasks /query /tn KidoAgent >nul 2>nul"
    Assert ($LASTEXITCODE -eq 0) 'boot on で S4U 起動時タスクが登録されなかった'
    Write-Host 'S4U タスクの登録を確認'
    # タスクは管理者(High)で起こされ、プロセス側で権限を下げて起動し直す。
    # 起動し直した子が「普通のユーザー」で動いていることをログで確かめる。
    $log = ''
    foreach ($i in 1..30) {
        $log = Get-Content -Encoding UTF8 "$env:USERPROFILE\.kido-agent\kido-agent.log" -Raw
        if ($log -match '普通のユーザー') { break }
        Start-Sleep 1
    }
    Assert ($log -match '普通のユーザー') 'ログに権限降格(普通のユーザー)が見えない'
    # ログの文言だけでなく、実際にプロセスの整合性レベルが Medium まで下がっている
    # ことを確かめる(SAFER だけだと High のまま残る)。
    Add-Type -Namespace K -Name Tok -MemberDefinition @'
[DllImport("advapi32.dll",SetLastError=true)] public static extern bool OpenProcessToken(IntPtr h,uint acc,out IntPtr tok);
[DllImport("advapi32.dll",SetLastError=true)] public static extern bool GetTokenInformation(IntPtr tok,int cls,IntPtr buf,int len,out int need);
[DllImport("advapi32.dll",SetLastError=true)] public static extern bool ConvertSidToStringSidW(IntPtr sid,out System.IntPtr str);
'@
    function Get-Integrity($procId) {
        $h = (Get-Process -Id $procId).Handle
        $tok = [IntPtr]::Zero
        if (-not [K.Tok]::OpenProcessToken($h, 0x8, [ref]$tok)) { return '?' }
        $need = 0; [K.Tok]::GetTokenInformation($tok, 25, [IntPtr]::Zero, 0, [ref]$need) | Out-Null
        $buf = [Runtime.InteropServices.Marshal]::AllocHGlobal($need)
        [K.Tok]::GetTokenInformation($tok, 25, $buf, $need, [ref]$need) | Out-Null
        $sp = [IntPtr]::Zero
        [K.Tok]::ConvertSidToStringSidW([Runtime.InteropServices.Marshal]::ReadIntPtr($buf), [ref]$sp) | Out-Null
        $sid = [Runtime.InteropServices.Marshal]::PtrToStringUni($sp)
        [Runtime.InteropServices.Marshal]::FreeHGlobal($buf)
        switch ($sid) { 'S-1-16-8192' { 'Medium' } 'S-1-16-12288' { 'High' } 'S-1-16-4096' { 'Low' } default { $sid } }
    }
    $levels = @(Get-CimInstance Win32_Process -Filter "Name='kido-agentd.exe'" | ForEach-Object { Get-Integrity $_.ProcessId })
    Write-Host ("kido-agentd の整合性レベル: " + ($levels -join ', '))
    Assert (($levels -contains 'Medium') -and -not ($levels -contains 'High')) "権限が Medium まで下がっていない: $levels"
    Write-Host '✓ S4U タスクが Medium(普通のユーザー)の整合性で動いていることを確認'
    & "$dest\kido-agent.exe" boot off
    Start-Sleep 3
    cmd /c "schtasks /query /tn KidoAgent >nul 2>nul"
    Assert ($LASTEXITCODE -ne 0) 'boot off で起動時タスクが消えない'
    Write-Host '✓ ログイン前対応(S4U・権限降格・boot off)を確認'

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
    $ok = $true
} finally {
    cmd /c "schtasks /end /tn KidoAgent >nul 2>nul"
    cmd /c "schtasks /delete /tn KidoAgent /f >nul 2>nul"
    Get-Process kido-agentd -ErrorAction SilentlyContinue | Stop-Process -Force
    Write-Host '--- ログ ---'
    Get-Content -Encoding UTF8 "$env:USERPROFILE\.kido-agent\kido-agent.log" -ErrorAction SilentlyContinue
}
# 後片付けの schtasks は終了コードを汚すので、成否は $ok で決める。
if ($ok) { exit 0 } else { exit 1 }
