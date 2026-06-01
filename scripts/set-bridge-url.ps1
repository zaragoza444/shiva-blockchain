param(
    [Parameter(Mandatory = $true)]
    [string]$BridgeUrl
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$src = Join-Path $root "internal\bridge\static\wallet\config.js"
$cfg = Join-Path $root "docs\wallet\config.js"
$url = $BridgeUrl.Trim().TrimEnd('/')

Copy-Item $src $cfg -Force
$content = Get-Content $cfg -Raw
$content = $content -replace "window\.__ONEX_BRIDGE_DEPLOY__ = window\.__ONEX_BRIDGE_DEPLOY__ \|\| '';", "window.__ONEX_BRIDGE_DEPLOY__ = '$url';"
Set-Content -Path $cfg -Value $content -Encoding utf8 -NoNewline
Write-Host "Updated $cfg with bridge $url"
