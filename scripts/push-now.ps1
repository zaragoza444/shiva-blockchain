# One-shot: GitHub login (if needed), create repo, push. Optional Gitea mirror.
param(
    [string]$GitHubUser = "zaragoza444",
    [string]$RepoName = "onex-blockchain",
    [string]$GiteaUrl = ""
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root

$gh = "$env:ProgramFiles\GitHub CLI\gh.exe"
if (-not (Test-Path $gh)) { $gh = "gh" }

$auth = & $gh auth status 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "GitHub login required. Complete the browser window, then run this script again."
    Write-Host ""
    & $gh auth login -h github.com -p https -w
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

$repo = "${GitHubUser}/${RepoName}"
$exists = & $gh repo view $repo 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "Creating GitHub repo $repo ..."
    & $gh repo create $repo --public --description "OneX blockchain: PoW node, OKX-style wallet, DeFi bridge, OneX Swap AMM" --source . --remote github --push
} else {
    git remote remove github 2>$null
    git remote add github "https://github.com/${repo}.git"
    git push -u github main
}

Write-Host ""
Write-Host "GitHub: https://github.com/$repo"

if ($GiteaUrl) {
    git remote remove gitea 2>$null
    git remote add gitea $GiteaUrl
    git push -u gitea main
    Write-Host "Gitea: $GiteaUrl"
} elseif (Test-Path "$root\remotes.env") {
    Get-Content "$root\remotes.env" | ForEach-Object {
        if ($_ -match '^\s*GITEA_URL=(.+)$') {
            $url = $matches[1].Trim().Trim('"')
            if ($url -and $url -notmatch 'YOUR_') {
                git remote remove gitea 2>$null
                git remote add gitea $url
                git push -u gitea main
                Write-Host "Gitea: $url"
            }
        }
    }
}

Write-Host "Done."
