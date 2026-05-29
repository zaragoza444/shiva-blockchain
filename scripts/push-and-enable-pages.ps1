# Push main to GitHub and enable GitHub Pages (Actions / workflow deployment).
param(
    [string]$GitHubUser = "zaragoza444",
    [string]$RepoName = "shiva-blockchain",
    [string]$BridgeUrl = ""
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root

$gh = "$env:ProgramFiles\GitHub CLI\gh.exe"
if (-not (Test-Path $gh)) { $gh = "gh" }

$authOk = $false
try {
    & $gh auth status 2>&1 | Out-Null
    $authOk = ($LASTEXITCODE -eq 0)
} catch { $authOk = $false }
if (-not $authOk) {
    Write-Host "GitHub login required — complete the browser prompt."
    & $gh auth login -h github.com -p https -w
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

$repo = "${GitHubUser}/${RepoName}"
$remoteUrl = "https://github.com/${repo}.git"

git remote remove github 2>$null
git remote add github $remoteUrl

Write-Host "Pushing main to $repo ..."
git push -u github main
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

if ($BridgeUrl) {
    $url = $BridgeUrl.Trim().TrimEnd('/')
    Write-Host "Setting SHIVA_BRIDGE_PUBLIC_URL = $url"
    & $gh variable set SHIVA_BRIDGE_PUBLIC_URL --body $url --repo $repo
}

Write-Host "Enabling GitHub Pages (GitHub Actions source) ..."
& $gh api --method PUT "repos/$repo/pages" -f build_type=workflow 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host "Note: Pages may already be enabled, or need repo admin access."
}

Write-Host "Triggering Pages deploy workflow ..."
& $gh workflow run "GitHub Pages" --repo $repo 2>&1 | Out-Null

Start-Sleep -Seconds 2
$run = & $gh run list --repo $repo --workflow "GitHub Pages" --limit 1 --json url,status,conclusion 2>&1 | ConvertFrom-Json

Write-Host ""
Write-Host "Repository: https://github.com/$repo"
Write-Host "Wallet URL:   https://${GitHubUser}.github.io/${RepoName}/wallet/"
if ($run) {
    Write-Host "Workflow:     $($run[0].status) — $($run[0].url)"
}
Write-Host ""
Write-Host "In GitHub: Settings → Pages → Build and deployment should show 'GitHub Actions'."
Write-Host "Optional: set SHIVA_BRIDGE_PUBLIC_URL (Actions variable) to your bridge HTTPS URL for live swaps."
