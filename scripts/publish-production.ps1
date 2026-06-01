# Production publish: sync wallet → GitHub (+ Pages) → Gitea (+ Pages).
param(
    [string]$GitHubUser = "zaragoza444",
    [string]$RepoName = "onex-blockchain",
    [string]$GiteaUrl = "",
    [string]$BridgeUrl = ""
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root

function Read-RemotesEnv {
    param([string]$Key)
    $path = Join-Path $root "remotes.env"
    if (-not (Test-Path $path)) { return "" }
    foreach ($line in Get-Content $path) {
        if ($line -match "^\s*$Key=(.+)$") {
            return $matches[1].Trim().Trim('"').Trim("'")
        }
    }
    return ""
}

Write-Host "=== OneX production publish ===" -ForegroundColor Cyan

# Load remotes.env defaults
if (-not $GiteaUrl) { $GiteaUrl = Read-RemotesEnv "GITEA_URL" }
if (-not $BridgeUrl) { $BridgeUrl = Read-RemotesEnv "ONEX_BRIDGE_PUBLIC_URL" }
$envUser = Read-RemotesEnv "GITHUB_USER"
if ($envUser) { $GitHubUser = $envUser }
$envRepo = Read-RemotesEnv "REPO_NAME"
if ($envRepo) { $RepoName = $envRepo }

Write-Host "Syncing wallet static files to docs/wallet ..."
New-Item -ItemType Directory -Force -Path "$root\docs\wallet" | Out-Null
Copy-Item "$root\internal\bridge\static\wallet\*" "$root\docs\wallet\" -Recurse -Force

if ($BridgeUrl) {
    & "$root\scripts\set-bridge-url.ps1" -BridgeUrl $BridgeUrl
}

Write-Host "Building binaries (smoke test) ..."
go build -o "$root\bin\onex-bridge.exe" "$root\cmd\onex-bridge"
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$dirty = git status --porcelain
if ($dirty) {
    git add docs/wallet internal/bridge/static/wallet scripts remotes.env.example .gitea/workflows/pages.yml PRODUCTION.md 2>$null
    git add -u
    $pending = git status --porcelain
    if ($pending) {
        git commit -m "Production sync: wallet static files and publish config."
        Write-Host "Committed local sync changes."
    }
}

Write-Host ""
Write-Host "--- GitHub ---" -ForegroundColor Yellow
& "$root\scripts\create-repo.ps1" -GitHubUser $GitHubUser -RepoName $RepoName
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

& "$root\scripts\push-and-enable-pages.ps1" -GitHubUser $GitHubUser -RepoName $RepoName -BridgeUrl $BridgeUrl

Write-Host ""
Write-Host "--- Gitea ---" -ForegroundColor Yellow
if ($GiteaUrl -and $GiteaUrl -notmatch 'YOUR_|example\.com') {
    git remote remove gitea 2>$null
    git remote add gitea $GiteaUrl
    Write-Host "Pushing main to Gitea ..."
    git push -u gitea main
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Gitea push failed. Check URL and credentials." -ForegroundColor Red
    } else {
        Write-Host "Gitea remote: $GiteaUrl"
        Write-Host "Enable Pages: repo Settings -> Pages -> deploy from Actions (pages.yml)"
    }
} else {
    Write-Host "Skip Gitea (set GITEA_URL in remotes.env or pass -GiteaUrl)" -ForegroundColor DarkYellow
    if (-not (Test-Path "$root\remotes.env")) {
        Copy-Item "$root\remotes.env.example" "$root\remotes.env"
        Write-Host "Created remotes.env from example — edit GITEA_URL and re-run."
    }
}

Write-Host ""
Write-Host "=== Production URLs ===" -ForegroundColor Green
Write-Host "GitHub wallet:  https://${GitHubUser}.github.io/${RepoName}/wallet/"
Write-Host "GitHub repo:    https://github.com/${GitHubUser}/${RepoName}"
Write-Host "Render bridge:  deploy render.yaml -> set ONEX_BRIDGE_PUBLIC_URL"
if ($BridgeUrl) {
    Write-Host "Bridge:         $BridgeUrl"
    Write-Host "Connected:      https://${GitHubUser}.github.io/${RepoName}/wallet/?bridge=$BridgeUrl"
}
Write-Host ""
Write-Host "Done."
