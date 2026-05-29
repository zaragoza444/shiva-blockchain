# Full deploy: commit check → GitHub push → Pages → Render instructions.
param(
    [string]$GitHubUser = "zaragoza444",
    [string]$RepoName = "shiva-blockchain",
    [string]$BridgeUrl = ""
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root

Write-Host "=== Shiva deploy ===" -ForegroundColor Cyan
Write-Host ""

if (git status --porcelain) {
    Write-Host "Uncommitted changes detected. Commit first, then re-run deploy." -ForegroundColor Yellow
    git status -sb
    exit 1
}

& "$root\scripts\push-and-enable-pages.ps1" -GitHubUser $GitHubUser -RepoName $RepoName -BridgeUrl $BridgeUrl
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host ""
Write-Host "=== Render (bridge + node) ===" -ForegroundColor Cyan
Write-Host "1. Open https://dashboard.render.com/blueprints"
Write-Host "2. New Blueprint Instance → connect GitHub repo $GitHubUser/$RepoName"
Write-Host "3. Apply render.yaml (creates shiva-node + shiva-bridge)"
Write-Host "4. Copy shiva-bridge public URL when live"
Write-Host ""
if (-not $BridgeUrl) {
    Write-Host "Then set bridge URL:" -ForegroundColor Yellow
    Write-Host "  .\scripts\push-and-enable-pages.ps1 -BridgeUrl https://YOUR-bridge.onrender.com"
    Write-Host "  OR wallet URL: https://${GitHubUser}.github.io/${RepoName}/wallet/?bridge=https://YOUR-bridge"
} else {
    Write-Host "Bridge variable set. Wallet:" -ForegroundColor Green
    Write-Host "  https://${GitHubUser}.github.io/${RepoName}/wallet/"
}
