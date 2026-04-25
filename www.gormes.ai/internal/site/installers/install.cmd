@echo off
rem install.cmd - Windows CMD wrapper around install.ps1.
rem
rem Usage:
rem   install.cmd
rem
rem Equivalent to:
rem   powershell -ExecutionPolicy Bypass -NoProfile -Command "irm https://gormes.ai/install.ps1 | iex"
rem
rem Honors the same GORMES_* environment variables as install.ps1.

setlocal

set "GORMES_PS1_URL=%GORMES_PS1_URL%"
if "%GORMES_PS1_URL%"=="" set "GORMES_PS1_URL=https://gormes.ai/install.ps1"

where powershell.exe >nul 2>&1
if errorlevel 1 (
  echo [gormes] error: powershell.exe is required to run install.cmd 1>&2
  exit /b 1
)

powershell.exe -ExecutionPolicy Bypass -NoProfile -Command "$ErrorActionPreference='Stop'; iex (irm '%GORMES_PS1_URL%')"
exit /b %ERRORLEVEL%
