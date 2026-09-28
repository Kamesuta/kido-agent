# 起動丸エージェントを Windows に入れる(入れ直し・更新にも使える)。
#
#   irm https://pc.kido.page/install.ps1 | iex
#
# 手元の zip やフォルダから入れるとき(リリース前の確かめなど):
#   $env:KIDO_AGENT_ZIP = "C:\path\kido-agent-windows-amd64.zip"
#   Get-Content -Raw -Encoding UTF8 .\install\install.ps1 | iex
# (Windows PowerShell 5.1 の -File は BOM の無いファイルを Shift_JIS として読んで
#  壊すので、手元のファイルは UTF-8 と指定して読ませる。PowerShell 7 なら -File -From でもよい)
#
# 管理者の権限は要らない。すべて利用者のフォルダとレジストリ(HKCU)に置く。
param(
    [string]$From = $env:KIDO_AGENT_ZIP,
    [switch]$Decoded
)

# 本体は下のヒア文字列に入れてある。Windows PowerShell 5.1 の irm は文字コードの
# 指定が無い応答を UTF-8 として読まず、日本語が化けて文字列の区切りまで壊れることがある。
# ヒア文字列の中なら化けても構文は壊れないので、化けていたら UTF-8 で読み直してやり直す。
# 進み具合の表示を止める。Windows Terminal では Expand-Archive などの進み具合の表示と
# 重なって日本語が二重に崩れるうえ、5.1 では取得もとても遅くなる。Expand-Archive は
# モジュールの関数で呼び出し元の変数を見ないので、global に書き、最後に元へ戻す。
$KidoSavedProgress = $global:ProgressPreference
$global:ProgressPreference = 'SilentlyContinue'

$Main = @'
param([string]$From)
$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [Text.Encoding]::UTF8
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$Dest = Join-Path $env:LOCALAPPDATA 'Programs\kido-agent'
$Kido = Join-Path $env:USERPROFILE 'KidoButtons'
$UninstallKey = 'Software\Microsoft\Windows\CurrentVersion\Uninstall\KidoAgent'

function Get-Arch {
    if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64' -or $env:PROCESSOR_ARCHITEW6432 -eq 'ARM64') { return 'arm64' }
    if ([Environment]::Is64BitOperatingSystem) { return 'amd64' }
    throw '32 ビット版の Windows には対応していません'
}

# Get-Package は入手元(GitHub のリリースか、手元の zip・フォルダ)から exe のあるフォルダを返す。
function Get-Package([string]$Work) {
    $src = $From
    if (-not $src) {
        $url = "https://kido-agent/releases/latest/download/kido-agent-windows-$(Get-Arch).zip"
        Write-Host "ダウンロードしています: $url"
        $src = Join-Path $Work 'kido-agent.zip'
        try { Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $src }
        catch { throw "ダウンロードできませんでした($($_.Exception.Message))。公開前は `$env:KIDO_AGENT_ZIP に手元の zip を指定してください" }
    }
    if (-not (Test-Path -LiteralPath $src -PathType Container)) {
        $out = Join-Path $Work 'unzipped'
        Expand-Archive -LiteralPath $src -DestinationPath $out -Force
        $src = $out
    }
    $daemon = Get-ChildItem -LiteralPath $src -Recurse -Filter 'kido-agentd.exe' | Select-Object -First 1
    if (-not $daemon -or -not (Test-Path (Join-Path $daemon.DirectoryName 'kido-agent.exe'))) {
        throw "kido-agent.exe と kido-agentd.exe が見つかりません: $src"
    }
    return $daemon.DirectoryName
}

# Stop-Agent は動いている常駐アプリを止める。動いている exe は上書きできないため。
function Stop-Agent {
    try {
        Invoke-WebRequest -UseBasicParsing -Method Post -Uri 'http://127.0.0.1:47822/control/stop' `
            -Headers @{ 'X-Kido-Control' = '1' } -TimeoutSec 3 | Out-Null
    } catch {}
    foreach ($i in 1..30) {
        if (-not (Get-Process -Name 'kido-agentd' -ErrorAction SilentlyContinue)) { return }
        Start-Sleep -Milliseconds 100
    }
    Get-Process -Name 'kido-agentd' -ErrorAction SilentlyContinue | Stop-Process -Force
    Start-Sleep -Milliseconds 300
}

# Open-UserKey は HKCU のキーを開く。無ければ途中の階層ごと作る(新しい利用者には
# Run すら無いことがある)。New-Item -Force は既存のキーの値を消してしまうので使わない。
function Open-UserKey([string]$Path) {
    return [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey($Path)
}

# Add-UserPath は kido-agent を、新しく開く PowerShell からも名前だけで呼べるようにする。
# [Environment]::SetEnvironmentVariable で書くと %USERPROFILE% などが展開されない形に
# 変わって他のパスが壊れるので、展開前の値に書き足して REG_EXPAND_SZ で書く。
function Add-UserPath {
    $key = Open-UserKey 'Environment'
    try {
        $raw = [string]$key.GetValue('Path', '', 'DoNotExpandEnvironmentNames')
        if (($raw -split ';') -contains $Dest) { return }
        $new = if ($raw) { "$raw;$Dest" } else { $Dest }
        $key.SetValue('Path', $new, 'ExpandString')
    } finally { $key.Close() }
    # 変わったことを開いているエクスプローラーに知らせる(変数を書いて消すと通知が飛ぶ)
    [Environment]::SetEnvironmentVariable('KIDO_AGENT_PATH_REFRESH', '1', 'User')
    [Environment]::SetEnvironmentVariable('KIDO_AGENT_PATH_REFRESH', $null, 'User')
}

function Register-Agent([string]$Version) {
    $run = Open-UserKey 'Software\Microsoft\Windows\CurrentVersion\Run'
    $run.SetValue('KidoAgent', "`"$Dest\kido-agentd.exe`"")
    $run.Close()
    $un = Open-UserKey $UninstallKey
    $un.SetValue('DisplayName', '起動丸エージェント')
    $un.SetValue('DisplayVersion', $Version)
    $un.SetValue('DisplayIcon', "$Dest\kido-agentd.exe")
    $un.SetValue('Publisher', 'Kido')
    $un.SetValue('InstallLocation', $Dest)
    $un.SetValue('UninstallString', "`"$Dest\kido-agent.exe`" uninstall")
    $un.SetValue('NoModify', 1, 'DWord')
    $un.SetValue('NoRepair', 1, 'DWord')
    $un.Close()
    Add-UserPath
}

# New-FolderShortcut はスタートメニューから操作フォルダを開けるようにする。
# ~/KidoButtons は常駐アプリが最初の起動で作るので、できるまで少し待ってから作る。
# WScript.Shell は日本語でない Windows だと日本語のファイル名で保存できないので、
# 英字の名前で保存してから付け替える。
function New-FolderShortcut {
    foreach ($i in 1..50) { if (Test-Path $Kido) { break }; Start-Sleep -Milliseconds 100 }
    $programs = Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs'
    $tmp = Join-Path $programs 'kido-agent-folder.lnk'
    $link = (New-Object -ComObject WScript.Shell).CreateShortcut($tmp)
    $link.TargetPath = $Kido
    $link.Save()
    Move-Item -LiteralPath $tmp -Destination (Join-Path $programs '起動丸の操作フォルダ.lnk') -Force
}

# Wait-Agent は常駐アプリが答えるまで待ち、その状態を返す(起動しなければ $null)。
function Wait-Agent {
    foreach ($i in 1..50) {
        try {
            return Invoke-RestMethod -UseBasicParsing -Uri 'http://127.0.0.1:47822/control/status' `
                -Headers @{ 'X-Kido-Control' = '1' } -TimeoutSec 1
        } catch { Start-Sleep -Milliseconds 100 }
    }
    return $null
}

$work = Join-Path ([IO.Path]::GetTempPath()) ("kido-agent-" + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $work | Out-Null
try {
    Write-Host '起動丸エージェントを入れます。'
    $pkg = Get-Package $work
    Stop-Agent
    New-Item -ItemType Directory -Path $Dest -Force | Out-Null
    Copy-Item -Path (Join-Path $pkg 'kido-agent*.exe') -Destination $Dest -Force
    Get-ChildItem -LiteralPath $Dest -Filter '*.exe' | Unblock-File
    $version = ((& "$Dest\kido-agent.exe" version) -split ' ')[-1]
    Register-Agent $version
    $env:Path = "$env:Path;$Dest"
    Write-Host "入れました($version): $Dest"
    Write-Host ''
    Write-Host '常駐アプリを起動します。「Windows セキュリティ」の確認が出たら「許可」を押してください。'
    Write-Host '(起動丸の本体から、この PC に届くようにするためです)'
    Start-Process -FilePath "$Dest\kido-agentd.exe" -WorkingDirectory $Dest
    $status = Wait-Agent
    if (-not $status) { throw '常駐アプリが起動しませんでした。ログ: %USERPROFILE%\.kido-agent\kido-agent.log' }
    New-FolderShortcut
    Write-Host ''
    # 更新で入れ直したときは組み直さない(pair は古い鍵を消してしまう)。
    if ($status.paired) {
        Write-Host '起動丸の本体とはもう組めています。組み直すときは: kido-agent pair'
    } else {
        # 残り時間を同じ行で書き換えるので、出力を横取りせず、この画面にそのまま書かせる。
        Start-Process -FilePath "$Dest\kido-agent.exe" -ArgumentList 'pair' -NoNewWindow -Wait
    }
    & "$Dest\kido-agent.exe" open | Out-Null
    Write-Host ''
    Write-Host '操作フォルダ(~/KidoButtons)を開きました。ショートカットを入れたフォルダを作ると、スマホに操作が増えます。'
    Write-Host '困ったときは: kido-agent check'
} catch {
    Write-Host "インストールに失敗しました: $($_.Exception.Message)" -ForegroundColor Red
} finally {
    Remove-Item -LiteralPath $work -Recurse -Force -ErrorAction SilentlyContinue
}
'@

try {
    # 「起」(U+8D77)が 1 文字として読めていれば、文字コードは正しい。
    # ここから下は化けた状態でも動く必要があるので、文字列に日本語を書かない。
    if (-not $Main.Contains([string][char]0x8D77)) {
        if ($Decoded) { throw 'install.ps1: failed to read as UTF-8' }
        [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
        $bytes = (New-Object Net.WebClient).DownloadData('https://pc.kido.page/install.ps1')
        $text = [Text.Encoding]::UTF8.GetString($bytes)
        & ([scriptblock]::Create($text)) -From $From -Decoded
        return
    }
    & ([scriptblock]::Create($Main)) -From $From
} finally {
    $global:ProgressPreference = $KidoSavedProgress
}
