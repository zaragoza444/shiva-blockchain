# Create GitHub repo zaragoza444/onex-blockchain and push main.
param(
    [string]$GitHubUser = "zaragoza444",
    [string]$RepoName = "onex-blockchain",
    [ValidateSet("public", "private")]
    [string]$Visibility = "public"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root

$gh = "$env:ProgramFiles\GitHub CLI\gh.exe"
if (-not (Test-Path $gh)) { $gh = "gh" }

try { & $gh auth status 2>&1 | Out-Null } catch {}
if ($LASTEXITCODE -ne 0) {
    Write-Host "Sign in as $GitHubUser (zardashtways44@gmail.com) ..."
    & $gh auth login -h github.com -p https -w
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

$repo = "${GitHubUser}/${RepoName}"
$exists = $false
try {
    & $gh repo view $repo 2>&1 | Out-Null
    $exists = ($LASTEXITCODE -eq 0)
} catch { $exists = $false }

if (-not $exists) {
    Write-Host "Creating $repo ($Visibility) ..."
    & $gh repo create $repo --$Visibility --description "OneX blockchain: PoW node, wallet, DeFi bridge, OneX Swap AMM" --source . --remote github
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
} else {
    Write-Host "Repo exists: https://github.com/$repo"
    git remote remove github 2>$null
    git remote add github "https://github.com/${repo}.git"
}

Write-Host "Pushing main ..."
git push -u github main
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host ""
Write-Host "Done: https://github.com/$repo"
Write-Host "Next: .\scripts\deploy.ps1"
