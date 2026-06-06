@echo off
cd /d "%~dp0"
set BSC_LAUNCHER_ROOT=%~dp0
set BSC_LAUNCHER_LISTEN=:9340
echo OneX Token Lab — http://127.0.0.1:9340
echo Uses bsc-launcher\.env (BSC_LAUNCHER_ENV=production)
echo Paste API key from .env into Settings in the browser.
cd /d "%~dp0.."
go run ./bsc-launcher/server
