@echo off
setlocal EnableDelayedExpansion
cd /d "%~dp0"
title OneX Token Lab

echo Building OneX Token Lab...
go build -o bin\bsc-launcher.exe ./bsc-launcher/server
if errorlevel 1 (
  echo Build failed. Install Go from https://go.dev/dl/ then retry.
  pause
  exit /b 1
)

REM Local dev overrides (safe defaults for run-onex-wallet style local use)
set "BSC_LAUNCHER_ROOT=%~dp0bsc-launcher"
set "BSC_LAUNCHER_ENV=development"
set "BSC_LAUNCHER_DATA_DIR=%~dp0bsc-launcher\data"
set "BSC_LAUNCHER_API_KEY="
set "BSC_LAUNCHER_CORS_ORIGINS=*"

if exist "bsc-launcher\.env" (
  for /f "usebackq eol=# tokens=1,* delims==" %%a in ("bsc-launcher\.env") do (
    if not "%%a"=="" (
      set "%%a=%%b"
    )
  )
)

REM Re-apply local overrides so production .env does not break localhost
set "BSC_LAUNCHER_ROOT=%~dp0bsc-launcher"
set "BSC_LAUNCHER_ENV=development"
set "BSC_LAUNCHER_DATA_DIR=%~dp0bsc-launcher\data"
if "%BSC_LAUNCHER_API_KEY%"=="" set "BSC_LAUNCHER_API_KEY="
set "BSC_LAUNCHER_CORS_ORIGINS=*"

if not exist "bsc-launcher\data" mkdir "bsc-launcher\data"

taskkill /IM bsc-launcher.exe /F >nul 2>&1
echo Starting OneX Token Lab on :9340...
start "OneX Token Lab" /MIN cmd /c "cd /d "%~dp0" && set BSC_LAUNCHER_ROOT=%BSC_LAUNCHER_ROOT%&& set BSC_LAUNCHER_ENV=%BSC_LAUNCHER_ENV%&& set BSC_LAUNCHER_DATA_DIR=%BSC_LAUNCHER_DATA_DIR%&& set BSC_LAUNCHER_API_KEY=%BSC_LAUNCHER_API_KEY%&& set BSC_LAUNCHER_CORS_ORIGINS=%BSC_LAUNCHER_CORS_ORIGINS%&& set BSCSCAN_API_KEY=%BSCSCAN_API_KEY%&& set BSC_RPC_URL=%BSC_RPC_URL%&& set BSC_DEPLOYER_PRIVATE_KEY=%BSC_DEPLOYER_PRIVATE_KEY%&& bin\bsc-launcher.exe"
ping -n 4 127.0.0.1 >nul

powershell -NoProfile -Command "try { (Invoke-WebRequest -Uri 'http://127.0.0.1:9340/health' -UseBasicParsing -TimeoutSec 5).StatusCode } catch { exit 1 }" >nul 2>&1
if errorlevel 1 (
  echo ERROR: Server did not start. Check port 9340 is free.
  pause
  exit /b 1
)

start http://127.0.0.1:9340/
echo OneX Token Lab OK — http://127.0.0.1:9340/
exit /b 0
