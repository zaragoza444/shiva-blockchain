$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $Root

function Ensure-Go {
    if (Get-Command go -ErrorAction SilentlyContinue) { return }
    Write-Host "Go not found. Trying winget install GoLang.Go ..."
    winget install -e --id GoLang.Go --accept-package-agreements --accept-source-agreements 2>$null
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Host "Install Go from https://go.dev/dl/ then re-run build-shiva.bat"
        exit 1
    }
}

Ensure-Go
New-Item -ItemType Directory -Force -Path bin | Out-Null

Write-Host "==> building shivad"
go build -o bin\shivad.exe .\cmd\shivad

Write-Host "==> building shiva CLI"
go build -o bin\shiva.exe .\cmd\shiva

Write-Host "==> building shiva-bridge (wallet)"
go build -o bin\shiva-bridge.exe .\cmd\shiva-bridge

Write-Host "==> building shiva-ai"
go build -o bin\shiva-ai.exe .\cmd\shiva-ai

Write-Host ""
Write-Host "Done:"
Write-Host "  bin\shivad.exe"
Write-Host "  bin\shiva.exe"
Write-Host "  bin\shiva-bridge.exe"
Write-Host "  bin\shiva-ai.exe"
Write-Host "Run node: .\run-shiva.bat"
Write-Host "Run wallet: .\run-shiva-wallet.bat"
