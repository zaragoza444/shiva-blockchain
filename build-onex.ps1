$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $Root

function Ensure-Go {
    if (Get-Command go -ErrorAction SilentlyContinue) { return }
    Write-Host "Go not found. Trying winget install GoLang.Go ..."
    winget install -e --id GoLang.Go --accept-package-agreements --accept-source-agreements 2>$null
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Host "Install Go from https://go.dev/dl/ then re-run build-onex.bat"
        exit 1
    }
}

Ensure-Go
New-Item -ItemType Directory -Force -Path bin | Out-Null

Write-Host "==> building onexd"
go build -o bin\onexd.exe .\cmd\onexd

Write-Host "==> building onex CLI"
go build -o bin\onex.exe .\cmd\onex

Write-Host "==> building onex-bridge (wallet)"
go build -o bin\onex-bridge.exe .\cmd\onex-bridge

Write-Host "==> building onex-ai"
go build -o bin\onex-ai.exe .\cmd\onex-ai

Write-Host ""
Write-Host "Done:"
Write-Host "  bin\onexd.exe"
Write-Host "  bin\onex.exe"
Write-Host "  bin\onex-bridge.exe"
Write-Host "  bin\onex-ai.exe"
Write-Host "Run node: .\run-onex.bat"
Write-Host "Run wallet: .\run-onex-wallet.bat"
