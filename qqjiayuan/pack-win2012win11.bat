@echo off
REM ============================================================
REM  pack-win2012win11.bat - one-click package builder (Windows 10/11)
REM  Windows port of pack-win2012.sh
REM
REM  Output: WindowsServer2012bushu\{server,web,admin-web} + sibling zip
REM    server\server.exe      fresh Go build of the backend
REM    server\dbinit.exe      db init + seed tool
REM    server\config.yaml     kept if exists, template generated if missing
REM    web\dist               fresh build (includes static assets)
REM    admin-web\dist         fresh build
REM
REM  Usage: pack-win2012win11.bat                    (deploy dir default C:/Users/Public/Code)
REM         pack-win2012win11.bat D:/qqjiayuan       (deploy dir, only affects config template)
REM ============================================================
setlocal EnableExtensions EnableDelayedExpansion
cd /d "%~dp0"

set "DEPLOY_DIR=%~1"
if "%DEPLOY_DIR%"=="" if defined DEPLOY_DIR set "DEPLOY_DIR=%DEPLOY_DIR%"
if "%DEPLOY_DIR%"=="" set "DEPLOY_DIR=C:/Users/Public/Code"
set "ROOT=%CD%"
set "PKG=%ROOT%\WindowsServer2012bushu"
set "ZIP=%ROOT%\WindowsServer2012bushu.zip"
set "NPMREG=%NPM_REGISTRY%"
if "%NPMREG%"=="" set "NPMREG=https://registry.npmmirror.com"
if "%GOPROXY%"=="" set "GOPROXY=https://goproxy.cn,direct"

call :log "STEP 0/5 toolchain check"

where node >nul 2>nul
if errorlevel 1 goto :err_node
for /f "delims=" %%v in ('node -v') do set "NODEV=%%v"
echo   [pack] node uses %NODEV%
set "NODE_OPTIONS="
set "NM=%NODEV:v=%"
for /f "tokens=1 delims=." %%a in ("%NM%") do if %%a GEQ 17 set "NODE_OPTIONS=--openssl-legacy-provider"

set "GO_BIN="
where go >nul 2>nul && for /f "delims=" %%g in ('where go') do if not defined GO_BIN set "GO_BIN=%%g"
if not defined GO_BIN (
    for %%g in ("%ProgramFiles%\Go\bin\go.exe" "%LOCALAPPDATA%\Programs\Go\bin\go.exe" "%USERPROFILE%\sdk\go1.26.0\bin\go.exe" "%USERPROFILE%\sdk\go1.27.0\bin\go.exe") do (
        if exist "%%~g" if not defined GO_BIN set "GO_BIN=%%~g"
    )
)
if not defined GO_BIN goto :err_go
for /f "tokens=2" %%v in ('"%GO_BIN%" version') do echo   [pack] go uses %%v

call :log "STEP 1/5 build web"

if not exist "%ROOT%\web\node_modules" (
    echo   [pack] node_modules missing, running npm install ^(slow on first run^)
    pushd "%ROOT%\web"
    call npm install --registry="%NPMREG%"
    if errorlevel 1 (
        popd
        goto :err_generic
    )
    popd
)
if exist "%ROOT%\web\dist" rmdir /s /q "%ROOT%\web\dist"
pushd "%ROOT%\web"
call npm run build
set "RC=%errorlevel%"
popd
if not %RC%==0 goto :err_generic
if not exist "%ROOT%\web\dist\index.html" goto :err_generic
echo   [pack] web\dist done

call :log "STEP 2/5 build admin-web"

if not exist "%ROOT%\admin-web\node_modules" (
    echo   [pack] node_modules missing, running npm install ^(slow on first run^)
    pushd "%ROOT%\admin-web"
    call npm install --registry="%NPMREG%"
    if errorlevel 1 (
        popd
        goto :err_generic
    )
    popd
)
if exist "%ROOT%\admin-web\dist" rmdir /s /q "%ROOT%\admin-web\dist"
pushd "%ROOT%\admin-web"
call npm run build
set "RC=%errorlevel%"
popd
if not %RC%==0 goto :err_generic
if not exist "%ROOT%\admin-web\dist\index.html" goto :err_generic
echo   [pack] admin-web\dist done

call :log "STEP 3/5 compile server.exe / dbinit.exe"

if not exist "%PKG%\server" mkdir "%PKG%\server"
pushd "%ROOT%\server"
set "CGO_ENABLED=0"
"%GO_BIN%" build -trimpath -ldflags "-s -w" -o "%PKG%\server\server.exe" .
if errorlevel 1 (
    popd
    goto :err_build
)
"%GO_BIN%" build -trimpath -ldflags "-s -w" -o "%PKG%\server\dbinit.exe" .\cmd\dbinit
if errorlevel 1 (
    popd
    goto :err_build
)
popd
set "CGO_ENABLED="
if not exist "%PKG%\server\server.exe" goto :err_build
del /q "%PKG%\server\*-2012r2.exe" >nul 2>nul
echo   [pack] server.exe / dbinit.exe done

call :log "STEP 4/5 assemble package"

if exist "%PKG%\web" rmdir /s /q "%PKG%\web"
if exist "%PKG%\admin-web" rmdir /s /q "%PKG%\admin-web"
mkdir "%PKG%\web"
mkdir "%PKG%\admin-web"
robocopy "%ROOT%\web\dist" "%PKG%\web\dist" /e /nfl /ndl /njh >nul
robocopy "%ROOT%\admin-web\dist" "%PKG%\admin-web\dist" /e /nfl /ndl /njh >nul

if exist "%PKG%\server\config.yaml" (
    echo   [pack] existing server\config.yaml kept ^(manual edits kept^)
    goto :zipstep
)
set "SECRET=qq-jiayuan-secret-change-me"
for /f "delims=" %%s in ('powershell -NoProfile -Command "($x=1..32|ForEach-Object{'{0:x2}' -f (Get-Random -Maximum 256)}) -join ''" 2^>nul') do set "SECRET=%%s"
(
    echo # qqjiayuan Windows Server 2012 R2 deploy config
    echo # The folder is assumed to be placed at %DEPLOY_DIR% , otherwise edit web_dir below ^(use forward slashes^)
    echo server:
    echo   port: 8080
    echo   web_dir: "%DEPLOY_DIR%/web/dist"
    echo   admin_web_dir: "%DEPLOY_DIR%/admin-web/dist"
    echo.
    echo mysql:
    echo   host: 127.0.0.1
    echo   port: 3306
    echo   user: root
    echo   password: "CHANGE-TO-SERVER-MYSQL-ROOT-PASSWORD"
    echo   dbname: qq_jiayuan
    echo.
    echo jwt:
    echo   secret: "%SECRET%"
    echo   expire_hours: 168
) > "%PKG%\server\config.yaml"
echo   [pack] template server\config.yaml generated ^(deploy dir %DEPLOY_DIR%^, remember to set the MySQL password^)

:zipstep
call :log "STEP 5/5 zip and verify"

if exist "%ZIP%" del /f /q "%ZIP%"
powershell -NoProfile -Command "Compress-Archive -Path '%PKG:\=\\%' -DestinationPath '%ZIP:\=\\%' -Force" >nul
if errorlevel 1 (
    echo   [pack] Compress-Archive failed, skipping zip ^(folder package still valid^)
) else (
    echo   [pack] zip: %ZIP%
)

set "FAIL=0"
for %%f in ("%PKG%\server\server.exe" "%PKG%\server\dbinit.exe" "%PKG%\server\config.yaml" "%PKG%\web\dist\index.html" "%PKG%\admin-web\dist\index.html") do (
    if exist "%%f" (echo   OK  %%f) else (echo   MISSING %%f & set "FAIL=1")
)
if not %FAIL%==0 goto :err_incomplete

call :end "package done -> %PKG%"
endlocal
exit /b 0

:err_incomplete
call :fatal "package incomplete, check output above"
endlocal
exit /b 1

:err_build
call :die "backend build failed"
endlocal
exit /b 1

:err_generic
call :die "front-end build failed"
endlocal
exit /b 1

:err_go
call :die "Go >= 1.26 not found. Install Go or run: set GO_BIN=C:\Go\bin\go.exe"
endlocal
exit /b 1

:err_node
call :die "node not found. Please install Node 16+ and add it to PATH"
endlocal
exit /b 1

REM ---------------- helpers ----------------
:log
echo ============================================================
echo   %~1
echo ============================================================
exit /b 0

:end
echo ============================================================
echo   %~1
echo ============================================================
exit /b 0

:die
echo.
echo   [FATAL] %~1
echo.
exit /b 0
