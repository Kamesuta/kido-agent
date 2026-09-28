# Windows 向けの zip を作る(CI とリリースで共用。PowerShell 7 ならどの OS でも動く)。
# 同じコードから exe を 2 つ作る。
#   kido-agentd.exe  GUI 版。ログオン時に起動しても黒い窓が出ない。常駐に使う
#   kido-agent.exe   コンソール版。PowerShell は GUI 版の終わりを待たず出力も
#                    受け取らないので、pair や check の表示にはこちらが要る
param(
    [Parameter(Mandatory)][string]$Version,
    [Parameter(Mandatory)][ValidateSet('amd64', 'arm64')][string]$Arch,
    [string]$Out = 'dist'
)
$ErrorActionPreference = 'Stop'
$stage = Join-Path $Out "windows-$Arch"
New-Item -ItemType Directory -Force -Path $stage | Out-Null
$env:GOOS = 'windows'; $env:GOARCH = $Arch; $env:CGO_ENABLED = '0'
$ld = "-s -w -X main.version=$Version"
go build -trimpath -ldflags $ld -o (Join-Path $stage 'kido-agent.exe') .
if ($LASTEXITCODE) { throw 'go build (console)' }
go build -trimpath -ldflags "$ld -H=windowsgui" -o (Join-Path $stage 'kido-agentd.exe') .
if ($LASTEXITCODE) { throw 'go build (gui)' }
Remove-Item Env:GOOS, Env:GOARCH
Compress-Archive -Force -Path (Join-Path $stage '*.exe') -DestinationPath (Join-Path $Out "kido-agent-windows-$Arch.zip")
