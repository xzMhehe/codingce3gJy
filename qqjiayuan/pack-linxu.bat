@echo off
REM ============================================================
REM  pack-linxu.bat - one-click Linux(amd64) package builder
REM  Windows port of pack-linxu.sh
REM
REM  Output: Linuxbushu\{server,web,admin-web} + sibling Linuxbushu.tar.gz
REM    server\server        fresh Go build, GOOS=linux GOARCH=amd64 CGO_ENABLED=0
REM    server\dbinit        db init + seed tool, same target
REM    server\config.yaml   kept if exists, template generated if missing
REM    web\dist             fresh build (includes static assets)
REM    admin-web\dist       fresh build
REM
REM  Deploy docs and helper scripts (deploy flow txt, start.sh, stop.sh,
REM  init-db.sh, qqjiayuan.service) are never modified or deleted, except
REM  CRLF -> LF normalization so they do not break on Linux.
REM
REM  Usage: pack-linxu.bat                      (deploy dir default /opt/qqjiayuan)
REM         pack-linxu.bat /srv/qqjiayuan       (deploy dir, only affects config template)
REM
REM  NOTE: a tar built on Windows carries no unix exec bit. On the server run
REM        chmod +x server/server server/dbinit *.sh  (see deploy flow txt)
REM        ./start.sh and ./init-db.sh also chmod by themselves.
REM ============================================================
setlocal EnableExtensions EnableDelayedExpansion
cd /d "%~dp0"

REM deploy dir: 1st arg wins, else a pre-set DEPLOY_DIR env var, else default
set "ARG_DEPLOY=%~1"
if not "%ARG_DEPLOY%"=="" set "DEPLOY_DIR=%ARG_DEPLOY%"
if "%DEPLOY_DIR%"=="" set "DEPLOY_DIR=/opt/qqjiayuan"

set "ROOT=%CD%"
set "PKG=%ROOT%\Linuxbushu"
set "TARBALL=%ROOT%\Linuxbushu.tar.gz"
set "ZIP=%ROOT%\Linuxbushu.zip"
set "WWIIPKG=%ROOT%\WWIIbushu"
set "WWII_TARBALL=%ROOT%\WWIIbushu.tar.gz"
set "WWII_ZIP=%ROOT%\WWIIbushu.zip"
set "NPMREG=%NPM_REGISTRY%"
if "%NPMREG%"=="" set "NPMREG=https://registry.npmmirror.com"
if "%GOPROXY%"=="" set "GOPROXY=https://goproxy.cn,direct"

call :log "STEP 0/6 toolchain check"

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
    for %%g in ("%ProgramFiles%\Go\bin\go.exe" "%LOCALAPPDATA%\Programs\Go\bin\go.exe" "D:\Go\bin\go.exe" "%USERPROFILE%\sdk\go1.26.0\bin\go.exe" "%USERPROFILE%\sdk\go1.27.0\bin\go.exe") do (
        if exist "%%~g" if not defined GO_BIN set "GO_BIN=%%~g"
    )
)
if not defined GO_BIN goto :err_go
for /f "tokens=3" %%v in ('"%GO_BIN%" version') do echo   [pack] go uses %%v

call :log "STEP 1/6 build web"

REM NOTE: check .bin\vue-cli-service.cmd, not just node_modules/ -- an interrupted
REM       npm install leaves node_modules/ present but without .bin, and the build
REM       then dies silently ("'vue-cli-service' is not recognized").
if not exist "%ROOT%\web\node_modules\.bin\vue-cli-service.cmd" (
    echo   [pack] web deps missing/incomplete, running npm install ^(slow on first run^)
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

call :log "STEP 2/6 build admin-web"

if not exist "%ROOT%\admin-web\node_modules\.bin\vue-cli-service.cmd" (
    echo   [pack] admin-web deps missing/incomplete, running npm install ^(slow on first run^)
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

call :log "STEP 3/6 cross compile server / dbinit (linux/amd64)"

if not exist "%PKG%\server" mkdir "%PKG%\server"
pushd "%ROOT%\server"
set "GOOS=linux"
set "GOARCH=amd64"
set "CGO_ENABLED=0"
"%GO_BIN%" build -trimpath -ldflags "-s -w" -o "%PKG%\server\server" .
if errorlevel 1 (
    popd
    goto :err_build
)
"%GO_BIN%" build -trimpath -ldflags "-s -w" -o "%PKG%\server\dbinit" .\cmd\dbinit
if errorlevel 1 (
    popd
    goto :err_build
)
popd
set "GOOS="
set "GOARCH="
set "CGO_ENABLED="
if not exist "%PKG%\server\server" goto :err_build
if not exist "%PKG%\server\dbinit" goto :err_build

REM make sure we produced a Linux ELF, not a stray windows exe
call :check_elf "%PKG%\server\server"
call :check_elf "%PKG%\server\dbinit"
echo   [pack] server / dbinit done (linux/amd64 ELF verified)

call :log "STEP 4/6 assemble package"

if exist "%PKG%\web" rmdir /s /q "%PKG%\web"
if exist "%PKG%\admin-web" rmdir /s /q "%PKG%\admin-web"
mkdir "%PKG%\web\dist"
mkdir "%PKG%\admin-web\dist"
robocopy "%ROOT%\web\dist" "%PKG%\web\dist" /e /nfl /ndl /njh >nul
robocopy "%ROOT%\admin-web\dist" "%PKG%\admin-web\dist" /e /nfl /ndl /njh >nul

if exist "%PKG%\server\config.yaml" (
    echo   [pack] existing server\config.yaml kept ^(manual edits kept^)
    goto :lfstep
)
set "SECRET=qq-jiayuan-secret-change-me"
for /f "delims=" %%s in ('powershell -NoProfile -Command "($x=1..32^|ForEach-Object{'{0:x2}' -f (Get-Random -Maximum 256)}) -join ''" 2^>nul') do set "SECRET=%%s"
(
    echo # qqjiayuan Linux deploy config
    echo # The folder is assumed to be placed at %DEPLOY_DIR% , otherwise edit web_dir below ^(absolute path required^)
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
    echo # WW2 ezfy uses its own database. Omit this whole block for single-database mode.
    echo # host / port / user / password default to the mysql block above.
    echo ezfy_mysql:
    echo   dbname: qq_ezzt
    echo.
    echo jwt:
    echo   secret: "%SECRET%"
    echo   expire_hours: 168
) > "%PKG%\server\config.yaml"
echo   [pack] template server\config.yaml generated ^(deploy dir %DEPLOY_DIR%^, remember to set the MySQL password^)

:lfstep
REM CRLF -> LF for shell scripts, service unit and docs (else "bad interpreter" on Linux)
powershell -NoProfile -Command "$n=0; Get-ChildItem -Path '%PKG%\*' -Include *.sh,*.service,*.txt -File | ForEach-Object { $c=[IO.File]::ReadAllText($_.FullName); $d=$c.Replace([string][char]13+[string][char]10,[string][char]10); if($d -ne $c){ [IO.File]::WriteAllText($_.FullName,$d); $n++ } }; if($n -gt 0){ Write-Output ('  [pack] CRLF to LF normalized: ' + $n + ' file(s)') }"
if not exist "%PKG%\*.txt" echo   [pack] warning: deploy doc (.txt) not found, doc not packaged

REM ================= WWII ezfy independent deploy package =================
REM   1st-zone template config + open-zone tool + docs. Same binary as home.
call :log "STEP 5/6 WWII independent package (WWIIbushu)"

if exist "%WWIIPKG%\server" rmdir /s /q "%WWIIPKG%\server"
if exist "%WWIIPKG%\web" rmdir /s /q "%WWIIPKG%\web"
mkdir "%WWIIPKG%\server"
mkdir "%WWIIPKG%\web\dist"
copy /y "%PKG%\server\server" "%WWIIPKG%\server\server" >nul
robocopy "%ROOT%\web\dist" "%WWIIPKG%\web\dist" /e /nfl /ndl /njh >nul
copy /y "%PKG%\start.sh" "%WWIIPKG%\start.sh" >nul
copy /y "%PKG%\stop.sh" "%WWIIPKG%\stop.sh" >nul
REM ASCII-only names for WWII extras (cmd parses this UTF-8 bat as GBK, so any Chinese
REM literal in a path would garble and the copy would silently skip - use zone2.sh /
REM wwii-readme.txt, which pack-linxu.sh keeps in sync on the Mac side too).
copy /y "%ROOT%\WWIIbushu\zone2.sh" "%WWIIPKG%\zone2.sh" >nul
copy /y "%ROOT%\WWIIbushu\wwii-readme.txt" "%WWIIPKG%\wwii-readme.txt" >nul

REM WWII zone-1 config is generated inline (UTF-8 via powershell, base64-embedded so this
REM bat stays ASCII / encoding-proof). No external template file needed. jwt.secret must
REM match the home package (SSO + admin reverse-proxy depend on it). MySQL password stays
REM a placeholder here - deploy-wwii fills it from the server's home config /opt/config.yaml.
set "HSEC="
findstr /b /c:"  secret:" "%PKG%\server\config.yaml" > "%TEMP%\hsec.tmp" 2>nul
if exist "%TEMP%\hsec.tmp" (
    for /f "tokens=2 delims=: " %%s in (%TEMP%\hsec.tmp) do set "HSEC=%%s"
    del /f /q "%TEMP%\hsec.tmp" 2>nul
)
set "HSEC=!HSEC:"=!"
powershell -NoProfile -Command "$e=$env:HSEC; $p='%WWIIPKG:\=\\%\server\config.yaml'; $b='IyBXV0lJIGV6ZnkgaW5kZXBlbmRlbnQgZGVwbG95IGNvbmZpZyAoem9uZSAxLCBnZW5lcmF0ZWQgYnkgcGFjay1saW54dSkKc2VydmVyOgogIG1vZGU6IHd3aWkKICBwb3J0OiA4MDgxCiAgd2ViX2RpcjogIi9vcHQvV1dJSWJ1c2h1L3dlYi9kaXN0IgogIGdhbWVfbmFtZTogIuS6jOaImOW+gemAlCIKICB6b25lX2lkOiAxCiAgem9uZV9uYW1lOiAiMeWMuiIKICB6b25lX3N1YjogIue6ouiJsuitpuaIkiIKICAjIGhvbWUgVVJMIG9mIHRoZSBob21lIHNlcnZlciAoZm9yIHRoZSBXV0lJICJiYWNrIHRvIGhvbWUiIGJ1dHRvbiwgZmlsbGVkIGJ5IGRlcGxveS13d2lpKQogIGhvbWVfdXJsOiAiIgpteXNxbDoKICBob3N0OiAxMjcuMC4wLjEKICBwb3J0OiAzMzA2CiAgdXNlcjogcm9vdAogIHBhc3N3b3JkOiAiQ0hBTkdFLVRPLVNFUlZFUi1NWVNRTC1ST09ULVBBU1NXT1JEIgogIGRibmFtZTogcXFfamlheXVhbgplemZ5X215c3FsOgogIGRibmFtZTogcXFfZXp6dAogIGJvb3RzdHJhcDogIm1pZ3JhdGUiCmp3dDoKICBzZWNyZXQ6ICJfX0pXVF9TRUNSRVRfXyIKICBleHBpcmVfaG91cnM6IDE2OA=='; $c=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($b)); if($e){$c=$c.Replace('__JWT_SECRET__',$e)}; [IO.File]::WriteAllText($p,$c,(New-Object System.Text.UTF8Encoding($false)))"
if defined HSEC (
    echo   [pack] WWII jwt.secret aligned with home package ^(!HSEC:~0,8!...^)
) else (
    echo   [pack] WARNING: no jwt.secret in home config - __JWT_SECRET__ placeholder kept, set it before deploy!
)
echo   [pack] WWII server\config.yaml generated ^(zone 1 template; MySQL password filled by deploy-wwii from server /opt/config.yaml^)

REM CRLF -> LF for WWII scripts / docs (copied from source, may be CRLF)
powershell -NoProfile -Command "$n=0; Get-ChildItem -Path '%WWIIPKG%' -Include *.sh,*.txt -File | ForEach-Object { $c=[IO.File]::ReadAllText($_.FullName); $d=$c.Replace([string][char]13+[string][char]10,[string][char]10); if($d -ne $c){ [IO.File]::WriteAllText($_.FullName,$d); $n++ } }; if($n -gt 0){ Write-Output ('  [pack] WWII CRLF to LF normalized: ' + $n + ' file(s)') }"

call :log "STEP 6/6 tar.gz / zip and verify"

if exist "%TARBALL%" del /f /q "%TARBALL%"
if exist "%WWII_TARBALL%" del /f /q "%WWII_TARBALL%"
where tar >nul 2>nul
if errorlevel 1 (
    echo   [pack] tar not found, skipping tar.gz ^(folder package still valid^)
    goto :ziptry
)
pushd "%ROOT%"
tar -czf "%TARBALL%" Linuxbushu
set "RC=%errorlevel%"
popd
if not %RC%==0 (
    echo   [pack] tar failed, skipping tar.gz
) else (
    echo   [pack] tar.gz: %TARBALL%
    echo   [pack] NOTE: Windows-built tar has no unix exec bit - on the server run
    echo   [pack]       chmod +x server/server server/dbinit *.sh
    echo   [pack]       ./start.sh and ./init-db.sh also chmod by themselves.
)
pushd "%ROOT%"
tar -czf "%WWII_TARBALL%" WWIIbushu
set "RC=%errorlevel%"
popd
if not %RC%==0 (
    echo   [pack] WWII tar failed, skipping WWIIbushu.tar.gz
) else (
    echo   [pack] WWII tar.gz: %WWII_TARBALL%
)

:ziptry
if exist "%ZIP%" del /f /q "%ZIP%"
powershell -NoProfile -Command "Compress-Archive -Path '%PKG:\=\\%' -DestinationPath '%ZIP:\=\\%' -Force" >nul 2>nul
if errorlevel 1 (
    echo   [pack] Compress-Archive failed, skipping zip
) else (
    echo   [pack] zip: %ZIP%
)
if exist "%WWII_ZIP%" del /f /q "%WWII_ZIP%"
powershell -NoProfile -Command "Compress-Archive -Path '%WWIIPKG:\=\\%' -DestinationPath '%WWII_ZIP:\=\\%' -Force" >nul 2>nul
if errorlevel 1 (
    echo   [pack] WWII Compress-Archive failed, skipping zip
) else (
    echo   [pack] WWII zip: %WWII_ZIP%
)

set "FAIL=0"
for %%f in ("%PKG%\server\server" "%PKG%\server\dbinit" "%PKG%\server\config.yaml" "%PKG%\web\dist\index.html" "%PKG%\admin-web\dist\index.html" "%PKG%\start.sh") do (
    if exist "%%f" (echo   OK  %%f) else (echo   MISSING %%f & set "FAIL=1")
)
if not exist "%PKG%\*.txt" set "FAIL=1"
if not %FAIL%==0 goto :err_incomplete

set "WFAIL=0"
for %%f in ("%WWIIPKG%\server\server" "%WWIIPKG%\server\config.yaml" "%WWIIPKG%\web\dist\ezfy.html" "%WWIIPKG%\start.sh" "%WWIIPKG%\stop.sh" "%WWIIPKG%\zone2.sh" "%WWIIPKG%\wwii-readme.txt") do (
    if exist "%%f" (echo   OK  %%f) else (echo   WWII MISSING %%f & set "WFAIL=1")
)
if not %WFAIL%==0 goto :err_incomplete

echo.
echo ============================================================
echo   package done -^> %PKG%  +  %WWIIPKG%
echo ============================================================
echo   On the server (minimal steps):
echo     tar -xzf Linuxbushu.tar.gz -C /opt ^&^& cd /opt/Linuxbushu
echo     chmod +x server/server server/dbinit *.sh
echo     vi server/config.yaml                  ^(set the MySQL password^)
echo     ./init-db.sh ^&^& ./start.sh
echo.
echo   WWII independent deploy ^(does not touch home, no restart of home server^):
echo     tar -xzf WWIIbushu.tar.gz -C /opt ^&^& cd /opt/WWIIbushu
echo     chmod +x *.sh server/server
echo     vi server/config.yaml                  ^(set MySQL password, keep jwt.secret = home^)
echo     ./start.sh                             ^(WWII zone 1, port 8081^)
echo     ./zone2.sh                              ^(clone zone 2: port 8082, db qq_ezzt2, fresh^)
echo.
endlocal
exit /b 0

REM ---------------- errors ----------------
:err_incomplete
call :fatal "package incomplete, check output above"
endlocal
exit /b 1

:err_elf
call :fatal "built binary is not a Linux ELF, check GOOS/GOARCH"
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
:check_elf
powershell -NoProfile -Command "if(-not (Test-Path '%~1')){ exit 1 }; $fs=[IO.File]::OpenRead('%~1'); $b=New-Object byte[] 4; $null=$fs.Read($b,0,4); $fs.Close(); if($b[0] -ne 0x7f -or $b[1] -ne 0x45 -or $b[2] -ne 0x4c -or $b[3] -ne 0x46){ exit 1 }; exit 0"
if errorlevel 1 goto :err_elf
exit /b 0

:log
echo ============================================================
echo   %~1
echo ============================================================
exit /b 0

REM NOTE: pause here -- when the .bat is double-clicked the console closes on exit
REM       and the whole reason for the failure scrolls away unseen.
REM       Set PACK_NOPAUSE=1 to skip the pause (for CI / scripted runs).
:die
echo.
echo   [FATAL] %~1
echo.
if not defined PACK_NOPAUSE pause
exit /b 0

:fatal
echo.
echo   [FATAL] %~1
echo.
if not defined PACK_NOPAUSE pause
exit /b 0
