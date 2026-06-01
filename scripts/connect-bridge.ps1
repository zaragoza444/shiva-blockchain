# Connect hosted wallet UI to your public onex-bridge URL.
param(
    [Parameter(Mandatory = $true)]
    [string]$BridgeUrl,
    [switch]$GitHubVariable,
    [string]$GitHubUser = "zaragoza444",
    [string]$RepoName = "onex-blockchain"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$url = $BridgeUrl.Trim().TrimEnd('/')

& "$root\scripts\set-bridge-url.ps1" -BridgeUrl $url

$internalCfg = Join-Path $root "internal\bridge\static\wallet\config.js"
$internal = Get-Content $internalCfg -Raw
if ($internal -notmatch "ONEX_BRIDGE_URL = '$url'") {
    Write-Host "Note: internal wallet uses auto-detect on localhost; Pages uses docs/wallet/config.js"
}

if ($GitHubVariable) {
    $gh = "$env:ProgramFiles\GitHub CLI\gh.exe"
    if (-not (Test-Path $gh)) { $gh = "gh" }
    & $gh auth status 2>&1 | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Run: gh auth login"
        exit 1
    }
    & $gh variable set ONEX_BRIDGE_PUBLIC_URL --body $url --repo "${GitHubUser}/${RepoName}"
    Write-Host "Set GitHub Actions variable ONEX_BRIDGE_PUBLIC_URL"
    & $gh workflow run "GitHub Pages" --repo "${GitHubUser}/${RepoName}" 2>&1 | Out-Null
}

Write-Host ""
Write-Host "Connected bridge: $url"
Write-Host ""
Write-Host "Local test (after push):"
Write-Host "  https://${GitHubUser}.github.io/${RepoName}/wallet/?bridge=$url"
Write-Host ""
Write-Host "Next:"
Write-Host "  git add docs/wallet/config.js"
Write-Host "  git commit -m 'Connect Pages wallet to bridge'"
Write-Host "  git push -u github main"
