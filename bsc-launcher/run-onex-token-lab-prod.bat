@echo off
cd /d "%~dp0"
set BSC_LAUNCHER_ROOT=%~dp0
if not exist "%~dp0.env" (
  echo Missing .env — copy .env.production.example to .env first.
  copy /Y "%~dp0.env.production.example" "%~dp0.env" >nul
  pause
)
echo OneX Token Lab PRODUCTION — http://127.0.0.1:9340
echo API key is in bsc-launcher\.env — paste into Settings in the UI.
cd /d "%~dp0.."
go build -o "%~dp0bsc-launcher.exe" ./bsc-launcher/server
if errorlevel 1 pause & exit /b 1
"%~dp0bsc-launcher.exe"
