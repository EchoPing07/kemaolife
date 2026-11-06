@echo off
rem Launcher for uninstall.ps1 - bypasses the default PowerShell execution policy
rem without changing machine or user policy (process scope only).
setlocal
cd /d "%~dp0"

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0uninstall.ps1" %*
set RC=%ERRORLEVEL%

echo.
if not "%RC%"=="0" (
  echo Uninstall failed with exit code %RC%. Check the messages above.
)
pause
endlocal
exit /b %RC%
