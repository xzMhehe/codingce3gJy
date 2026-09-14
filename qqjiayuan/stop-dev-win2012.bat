@echo off
REM =====================================================================
REM  QQ JiaYuan - STOP DEV SERVICES for Windows Server 2012 R2
REM
REM  Stops whatever start-dev-win2012.bat started:
REM    QQJY-backend   :8080
REM    QQJY-web       :8000
REM    QQJY-admin     :8001
REM
REM  Three passes, in this order (order matters):
REM    1. kill by console window title - cleanest, kills the whole tree
REM       (cmd.exe + air + tmp\server.exe) in one shot
REM    2. kill air.exe - MUST come before the port pass, otherwise the
REM       watcher rebuilds and respawns the backend right after you kill it
REM    3. kill by listening port - catches anything still holding 8080/8000/8001
REM
REM  It does NOT touch the production service. To stop that use:
REM    nssm stop qqjiayuan
REM
REM  Run this as Administrator, otherwise taskkill may be denied.
REM  ASCII-only + CRLF on purpose, same as start-dev-win2012.bat.
REM =====================================================================

setlocal
chcp 65001 >nul
title QQ JiaYuan Dev - STOP
cd /d "%~dp0"

set "PORTS=8080 8000 8001"

echo ============================================================
echo   QQ JiaYuan  -  STOP DEV SERVICES
echo ============================================================
echo.

net session >nul 2>&1
if errorlevel 1 (
  echo   [WARN] not running as Administrator.
  echo          If the services were started elevated, taskkill will be
  echo          denied. Re-run this .bat with "Run as administrator".
  echo.
)

REM ---------------- pass 1: by console window title ---------------------
echo [1/3] by console window title
for %%T in (QQJY-backend QQJY-web QQJY-admin) do (
  taskkill /FI "WINDOWTITLE eq %%T" /T /F >nul 2>&1
  if errorlevel 1 (echo   %%T  - window not found) else (echo   %%T  - killed)
)
echo.

REM ---------------- pass 2: kill the air watcher ------------------------
REM  Must run BEFORE the port pass: if air is still alive it will rebuild
REM  and respawn tmp\server.exe immediately after you kill the port.
echo [2/3] go hot reload watcher
taskkill /IM air.exe /F >nul 2>&1
if errorlevel 1 (echo   air.exe  - not running) else (echo   air.exe  - killed)
echo.

REM ---------------- pass 3: by listening port ---------------------------
echo [3/3] by listening port
for %%P in (%PORTS%) do call :KILLPORT %%P
echo.

REM ---------------- verify ----------------------------------------------
echo ------------------------------------------------------------
echo   port check
set "BUSY=0"
for %%P in (%PORTS%) do (
  netstat -ano | findstr /r /c:":%%P .*LISTENING" >nul 2>&1
  if errorlevel 1 (
    echo     %%P  free
  ) else (
    echo     %%P  STILL LISTENING  -^> see the note below
    set "BUSY=1"
  )
)
echo ------------------------------------------------------------
echo.
if "%BUSY%"=="1" (
  echo   Some ports are still in use. Most likely causes:
  echo     - this .bat was not run as Administrator
  echo     - the port is held by something else ^(check: netstat -ano ^| findstr ":8080"^)
  echo     - the production service is running ^(stop it: nssm stop qqjiayuan^)
) else (
  echo   All dev services stopped.
)
echo.
echo   Any leftover console window can just be closed with the X button.
echo.
pause
endlocal
exit /b 0

REM =====================================================================
:KILLPORT
set "P=%~1"
set "HIT="
for /f "tokens=5" %%a in ('netstat -ano ^| findstr /r /c:":%P% .*LISTENING"') do (
  set "HIT=1"
  echo   port %P%  -^>  PID %%a  - killing
  taskkill /PID %%a /T /F >nul 2>&1
)
if not defined HIT echo   port %P%  -^>  not listening
exit /b 0
