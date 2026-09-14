@echo off
REM =====================================================================
REM  QQ JiaYuan - DEV MODE launcher for Windows Server 2012 R2
REM
REM  Launches three services, each in its own window:
REM    QQJY-backend   :8080   go + air hot reload (falls back to go run .)
REM    QQJY-web       :8000   vue-cli dev server (user site)
REM    QQJY-admin     :8001   vue-cli dev server (/admin-ui/)
REM
REM  Run it from the qqjiayuan root. See Server2012.md for tool versions.
REM
REM  ---------------------------------------------------------------------
REM  WHY THIS FILE LOOKS THE WAY IT DOES (read before editing):
REM
REM  1. ASCII-ONLY. cmd.exe reads .bat files using the OEM codepage (936
REM     on Chinese Windows). UTF-8 Chinese inside a .bat gets garbled or
REM     breaks parsing. Do not add Chinese characters to this file.
REM
REM  2. NO "chcp 65001" AT THE TOP. On Windows 7 / 8 / 2008R2 / 2012 the
REM     code-page switch is known to make cmd.exe lose its read position
REM     in the batch file and silently stop executing the rest of it.
REM     All output here is English, so the switch is not needed.
REM     The backend window sets its own code page (see step 5) because
REM     the Go server prints Chinese to its console.
REM
REM  3. SELF RELAUNCH GUARD. The first thing this script does is re-run
REM     itself inside "cmd /k", so the console window can NEVER flash and
REM     disappear - even on a hard error you can read the message.
REM     The inner run is identified by the __run__ argument.
REM
REM  4. CRLF line endings. Keep them.
REM  ---------------------------------------------------------------------

if /i not "%~1"=="__run__" (
  cmd /k call "%~f0" __run__
  exit /b
)

setlocal
title QQ JiaYuan Dev 2012 R2
cd /d "%~dp0"

echo ============================================================
echo   QQ JiaYuan  -  DEV MODE
echo ============================================================
echo   script dir : %~dp0
echo   cmd dir    : %CD%
echo.

REM ---------------- 0. layout check ------------------------------------
echo [0/6] layout check
if not exist "server\main.go" (
  echo   [ERROR] server\main.go not found.
  echo           Put this .bat in the qqjiayuan root folder and run it there.
  goto :FAIL
)
if not exist "web\package.json" (
  echo   [ERROR] web\package.json not found.
  goto :FAIL
)
if not exist "admin-web\package.json" (
  echo   [ERROR] admin-web\package.json not found.
  goto :FAIL
)
if not exist "server\config.yaml" (
  echo   [WARN] server\config.yaml not found - backend will fail to start.
  echo          Create it first, see Server2012.md section 5.
)
echo   ok
echo.

REM ---------------- 1. admin rights ------------------------------------
net session >nul 2>&1
if errorlevel 1 (set "IS_ADMIN=0") else (set "IS_ADMIN=1")

REM ---------------- 2. toolchain ---------------------------------------
echo [1/6] toolchain check
set "MISSING=0"
where go   >nul 2>&1
if errorlevel 1 (echo   [ERROR] go   not found in PATH & set "MISSING=1")
where node >nul 2>&1
if errorlevel 1 (echo   [ERROR] node not found in PATH & set "MISSING=1")
where npm  >nul 2>&1
if errorlevel 1 (echo   [ERROR] npm  not found in PATH & set "MISSING=1")

if "%MISSING%"=="1" (
  echo.
  echo   Install Go 1.26.6 and Node 16.20.2 first, then open a NEW cmd
  echo   window before running this script again.
  goto :FAIL
)
for /f "delims=" %%v in ('go version') do echo   %%v
for /f "delims=" %%v in ('node -v')    do echo   node %%v
for /f "delims=" %%v in ('npm -v')     do echo   npm  %%v
echo.

REM ---------------- 3. mirrors + PATH ----------------------------------
set "GOPROXY=https://goproxy.cn,direct"
set "NPM_CONFIG_REGISTRY=https://registry.npmmirror.com"
for /f "delims=" %%p in ('go env GOPATH') do set "GOPATH_DIR=%%p"
if defined GOPATH_DIR set "PATH=%GOPATH_DIR%\bin;%PATH%"

REM ---------------- 4. frontend dependencies ---------------------------
echo [2/6] frontend dependencies
if not exist "web\node_modules" (
  echo   web\node_modules missing - running npm install, first run is slow
  pushd "web"
  call npm install --registry=%NPM_CONFIG_REGISTRY%
  if errorlevel 1 (echo   [ERROR] npm install failed in web & popd & goto :FAIL)
  popd
) else (
  echo   web        ok
)
if not exist "admin-web\node_modules" (
  echo   admin-web\node_modules missing - running npm install, first run is slow
  pushd "admin-web"
  call npm install --registry=%NPM_CONFIG_REGISTRY%
  if errorlevel 1 (echo   [ERROR] npm install failed in admin-web & popd & goto :FAIL)
  popd
) else (
  echo   admin-web  ok
)
echo.

REM ---------------- 5. air (go hot reload) -----------------------------
echo [3/6] go hot reload - air
set "HAVE_AIR=0"
where air >nul 2>&1
if not errorlevel 1 set "HAVE_AIR=1"

if "%HAVE_AIR%"=="0" (
  echo   air not found - installing github.com/air-verse/air@latest
  call go install github.com/air-verse/air@latest
  if errorlevel 1 (
    echo   [WARN] air install failed - backend will use "go run ." instead
  ) else (
    set "HAVE_AIR=1"
    echo   air installed
  )
) else (
  echo   air ok
)
echo.

REM ---------------- 6. firewall ----------------------------------------
echo [4/6] firewall - ports 8080 / 8000 / 8001
if "%IS_ADMIN%"=="1" (
  netsh advfirewall firewall show rule name="qqjiayuan-dev" >nul 2>&1
  if errorlevel 1 (
    netsh advfirewall firewall add rule name="qqjiayuan-dev" dir=in action=allow protocol=TCP localport=8080,8000,8001 >nul
    echo   rule qqjiayuan-dev added
  ) else (
    echo   rule qqjiayuan-dev already exists
  )
) else (
  echo   [WARN] not running as Administrator - firewall step skipped.
  echo          To reach the dev servers from another machine, run this .bat
  echo          as Administrator once, or run this command manually:
  echo            netsh advfirewall firewall add rule name="qqjiayuan-dev" dir=in action=allow protocol=TCP localport=8080,8000,8001
)
echo.

REM ---------------- 7. launch ------------------------------------------
echo [5/6] launching services
if "%HAVE_AIR%"=="1" (
  start "QQJY-backend" /d "%~dp0server" cmd /k "chcp 65001 && air"
) else (
  start "QQJY-backend" /d "%~dp0server" cmd /k "chcp 65001 && go run ."
)
timeout /t 3 /nobreak >nul

REM  --host 0.0.0.0 exposes the dev servers on all interfaces so you can
REM  open them from another machine. Remove it to bind to localhost only.
start "QQJY-web"   /d "%~dp0web"       cmd /k "npm run serve -- --host 0.0.0.0 --port 8000"
start "QQJY-admin" /d "%~dp0admin-web" cmd /k "npm run serve -- --host 0.0.0.0 --port 8001"
echo   three windows opened
echo.

echo [6/6] done
echo ============================================================
echo   All three services launched in separate windows.
echo   Close a window to stop that service.
echo ------------------------------------------------------------
echo   local                             remote - replace SERVER_IP
echo   http://127.0.0.1:8080             http://SERVER_IP:8080
echo   http://127.0.0.1:8000             http://SERVER_IP:8000
echo   http://127.0.0.1:8001/admin-ui/   http://SERVER_IP:8001/admin-ui/
echo ------------------------------------------------------------
echo   backend log : watch the QQJY-backend window
echo   config file : server\config.yaml
echo.
echo   If the QQJY-backend window shows an error instead of the startup
echo   banner, Go binaries cannot run on this system.
echo   See Server2012.md, appendix A.
echo.
echo   [SECURITY] Dev servers are exposed on 0.0.0.0 with no auth and
echo   with source maps on. For production use start-prod.bat instead.
echo ============================================================
echo.
goto :DONE

:FAIL
echo.
echo ============================================================
echo   SCRIPT STOPPED - see the [ERROR] line above.
echo   This window is kept open on purpose so you can read it.
echo ============================================================
echo.

:DONE
echo   Press any key to close this window.
pause
endlocal
exit /b 0
