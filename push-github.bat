@echo off
cd /d "%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\push-now.ps1" -GitHub "https://github.com/zaragoza444/shiva-blockchain.git"
pause
