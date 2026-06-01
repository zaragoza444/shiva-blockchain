$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$Wsh = New-Object -ComObject WScript.Shell
$Desktop = [Environment]::GetFolderPath("Desktop")
$Shortcut = $Wsh.CreateShortcut((Join-Path $Desktop "OneX Wallet.lnk"))
$Shortcut.TargetPath = Join-Path $Root "run-onex-wallet.bat"
$Shortcut.WorkingDirectory = $Root
$Shortcut.Description = "OneX Wallet - bridge to local OneX blockchain"
$Icon = Join-Path $Root "onex-icon.ico"
if (Test-Path $Icon) { $Shortcut.IconLocation = $Icon }
$Shortcut.Save()
Write-Host "Created: $($Shortcut.FullName)"
