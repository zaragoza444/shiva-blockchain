param(
    [Parameter(Mandatory = $true)]
    [string]$GitHub,
    [Parameter(Mandatory = $true)]
    [string]$Gitea,
    [string]$Branch = "main"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root

if (-not (Test-Path .git)) {
    git init -b $Branch
    git add -A
    git commit -m "Initial commit: OneX blockchain production stack"
}

function Set-Remote($name, $url) {
    git remote remove $name 2>$null
    git remote add $name $url
    Write-Host "Remote $name -> $url"
}

Set-Remote "github" $GitHub
Set-Remote "gitea" $Gitea

git push -u github $Branch
git push -u gitea $Branch

Write-Host ""
Write-Host "Published branch '$Branch' to GitHub and Gitea."
