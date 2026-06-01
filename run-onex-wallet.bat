@echo off
setlocal
cd /d "%~dp0"
title OneX Wallet Bridge

if not exist "bin\onexd.exe" (
  echo Building OneX...
  call "%~dp0build-onex.bat"
)

REM Start blockchain node if API is down
powershell -NoProfile -Command "try { (Invoke-WebRequest -Uri 'http://127.0.0.1:8545/health' -UseBasicParsing -TimeoutSec 2).StatusCode } catch { exit 1 }" >nul 2>&1
if errorlevel 1 (
  echo Starting OneX node...
  start "OneX Node" /MIN "bin\onexd.exe" -datadir "%~dp0data" -api :8545 -listen :30303
  timeout /t 3 >nul
)

if not exist "bin\onex-bridge.exe" (
  echo Building bridge...
  go build -o bin\onex-bridge.exe ./cmd/onex-bridge
)

REM Start bridge if not running
powershell -NoProfile -Command "try { (Invoke-WebRequest -Uri 'http://127.0.0.1:9338/bridge/status' -UseBasicParsing -TimeoutSec 2).StatusCode } catch { exit 1 }" >nul 2>&1
if errorlevel 1 (
  echo Starting OneX Wallet bridge...
  start "OneX Bridge" /MIN "bin\onex-bridge.exe"
  timeout /t 2 >nul
)

start http://127.0.0.1:9338/wallet/
echo OneX Wallet opened. Bridge links your wallet to the local node.
exit /b 0
