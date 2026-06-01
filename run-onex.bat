@echo off
cd /d "%~dp0"
if not exist "bin\onexd.exe" (
  echo Building OneX node...
  call "%~dp0build-onex.bat"
)
if exist "bin\onexd.exe" (
  start "OneX Node" "bin\onexd.exe" -datadir "%~dp0data" -api :8545 -listen :30303
  timeout /t 2 >nul
  start http://localhost:8545/explorer/
  exit /b 0
)
echo OneX node not built in this folder yet.
echo.
echo Options:
echo   1. Install Go and build: go build -o bin\onexd.exe .\cmd\onexd
echo   2. Run from WSL: /home/ubuntu/onex-blockchain
echo.
start notepad "%~dp0README.txt"
pause
