@echo off
chcp 65001 >nul
setlocal EnableExtensions EnableDelayedExpansion
title WWII deployer

rem ============================================================================
rem  deploy-wwii.bat - deploy WWII (ezfy) independent package to one Linux host
rem
rem  Twin of tools/deploy.bat, but targets the WWII independent package
rem  (WWIIbushu.tar.gz) -> /opt/WWIIbushu, port 8081 (zone 1). Home server is
rem  NEVER touched, so updating WWII does not restart home (no downtime).
rem  Optional --zone2 clones a zone-2 instance (/opt/wwii-2, port 8082,
rem  independent DB qq_ezzt2, bootstrap=fresh) and starts it.
rem
rem  Usage (from repo root qqjiayuan):
rem    tools\deploy-wwii.bat -p 'Pass..' -h 124.222.208.92 --no-backup
rem    tools\deploy-wwii.bat -p 'Pass..' -h 124.222.208.92 --zone2 --no-backup
rem    tools\deploy-wwii.bat --dry-run -h 39.105.151.141
rem
rem  Password priority: -p arg > env SSH_PASS > interactive prompt
rem ============================================================================

set "SCRIPT_DIR=%~dp0"
set "SSH_RUN=%SCRIPT_DIR%ssh-run.js"
set "PROJ_DIR=%SCRIPT_DIR%.."

if "%SSH_HOST%"=="" set "SSH_HOST=39.105.151.141"
if "%SSH_USER%"=="" set "SSH_USER=root"

set "PKG=%PROJ_DIR%\WWIIbushu.tar.gz"
set "REMOTE_DIR=/opt/WWIIbushu"
set "REMOTE_TGZ=/opt/WWIIbushu.tar.gz"
set "CONFIG_SEED=/opt/config.yaml.wwii"
set "WWII_PORT=8081"
set "ZONE2_DIR=/opt/wwii-2"
set "ZONE2_PORT=8082"

set "NO_RESTART=0"
set "NO_BACKUP=0"
set "DRY_RUN=0"
set "ZONE2=0"
set "TMPOUT=%TEMP%\wwii_out_%RANDOM%%RANDOM%.txt"
set "TMPERR=%TEMP%\wwii_err_%RANDOM%%RANDOM%.txt"

rem ---------- parse ----------
:parse
if "%~1"=="" goto parsed
if /i "%~1"=="-f"         ( set "PKG=%~2"      & shift & shift & goto parse )
if /i "%~1"=="--file"     ( set "PKG=%~2"      & shift & shift & goto parse )
if /i "%~1"=="-h"         ( set "SSH_HOST=%~2" & shift & shift & goto parse )
if /i "%~1"=="--host"     ( set "SSH_HOST=%~2" & shift & shift & goto parse )
if /i "%~1"=="-u"         ( set "SSH_USER=%~2" & shift & shift & goto parse )
if /i "%~1"=="--user"     ( set "SSH_USER=%~2" & shift & shift & goto parse )
if /i "%~1"=="-p"         goto parse_pass
if /i "%~1"=="--pass"     goto parse_pass
if /i "%~1"=="--no-restart" ( set "NO_RESTART=1" & shift & goto parse )
if /i "%~1"=="--no-backup"  ( set "NO_BACKUP=1"  & shift & goto parse )
if /i "%~1"=="--dry-run"    ( set "DRY_RUN=1"    & shift & goto parse )
if /i "%~1"=="--zone2"      ( set "ZONE2=1"      & shift & goto parse )
if /i "%~1"=="--help"       goto usage
if /i "%~1"=="-?"           goto usage
echo   XX unknown arg: %~1   try --help
exit /b 1

:parse_pass
set "SSH_PASS=%~2"
shift
shift
goto parse

:parsed
if not exist "%SSH_RUN%" (
  echo   XX missing %SSH_RUN%
  exit /b 1
)
node -v >nul 2>&1
if errorlevel 1 (
  echo   XX node not found, install Node.js or add node.exe to PATH
  exit /b 1
)

echo ==^> target  %SSH_USER%@%SSH_HOST%:%REMOTE_DIR%
if "%DRY_RUN%"=="1" echo   WARN: DRY-RUN, plan only, no action

rem =========================================================
rem 0/7 preflight
rem =========================================================
echo ==^> 0/7 preflight

if defined SSH_PASS goto pass_ready
set /p "SSH_PASS=SSH password (%SSH_USER%@%SSH_HOST%): "
:pass_ready

if not exist "%PKG%" (
  echo   XX package not found: %PKG%  ^(run pack-linxu.bat first^)
  exit /b 1
)
for %%A in ("%PKG%") do set "LOCAL_SIZE=%%~zA"

set "LOCAL_MD5="
for /f "skip=1 delims=" %%i in ('certutil -hashfile "%PKG%" MD5 2^>nul') do (
  if not defined LOCAL_MD5 set "LOCAL_MD5=%%i"
)
set "LOCAL_MD5=%LOCAL_MD5: =%"
if "%LOCAL_MD5%"=="" (
  echo   XX md5 failed - certutil unavailable
  exit /b 1
)
set /a LOCAL_MB=%LOCAL_SIZE%/1048576
echo   OK package %PKG% ~%LOCAL_MB% MB md5=%LOCAL_MD5%

rem 2026-10-19 hardening: local package integrity pre-check.
rem Windows tar.exe can crash (0xC0000005) and leave a TRUNCATED tar.gz
rem that md5 checks cannot detect; abort before uploading a bad package.
where python >nul 2>nul
if errorlevel 1 (
  echo   WARN: python not found, skip local package integrity check
) else (
  python -c "import tarfile,sys; t=tarfile.open(sys.argv[1],'r:gz'); n=sum(1 for _ in t); t.close(); assert n>0" "%PKG%" >nul 2>&1
  if errorlevel 1 (
    echo   XX local package integrity FAILED ^(truncated/corrupt^): %PKG%
    echo       rerun pack-linxu.bat to rebuild, then deploy again
    goto fail_clean
  )
  echo   OK local package integrity check passed
)

call :rsh_read "echo ping"
if "%RH_OUT%"=="ping" goto ssh_ok
echo   XX SSH unreachable %SSH_USER%@%SSH_HOST%
exit /b 1
:ssh_ok
echo   OK SSH ok

call :rsh_read "df -P /opt | tail -1 | awk '{print $4}'"
set "DISK_AVAIL=%RH_OUT%"
if "%DISK_AVAIL%"=="" (
  echo   XX cannot read /opt free space
  exit /b 1
)
set /a AVAIL_MB=%DISK_AVAIL%/1024
set /a NEED_MB=%LOCAL_SIZE%/1048576+120
if %AVAIL_MB% LSS %NEED_MB% (
  echo   XX /opt free %AVAIL_MB%MB, need ^(~%NEED_MB%MB^)
  exit /b 1
)
echo   OK /opt free %AVAIL_MB%MB ^(need ~%NEED_MB%MB^)

call :rsh_read "test -d '%REMOTE_DIR%' && echo yes || echo no"
set "HAS_DIR=%RH_OUT%"
if "%HAS_DIR%"=="yes" (
  call :rsh_read "ss -lntp 2>/dev/null | grep -c ':%WWII_PORT% ' || true"
  echo   OK remote dir exists, WWII %WWII_PORT% listeners=!RH_OUT!
) else (
  echo   WARN: remote has no %REMOTE_DIR% - fresh install ^(nothing to back up^)
)

if "%DRY_RUN%"=="1" echo ==^> DRY-RUN ends here
if "%NO_RESTART%"=="1" echo   WARN: --no-restart: upload + verify only, stop here

rem ---------- 1/7 backup config ----------
set "BAK_CONFIG="
call :mktimestamp
if "%NO_RESTART%"=="0" if "%HAS_DIR%"=="yes" (
  echo ==^> 1/7 backup live config
  call :rsh "cp -a %REMOTE_DIR%/server/config.yaml %REMOTE_DIR%/server/config.yaml.bak.%TS%"
  if not "!RH_RC!"=="0" (
    echo   XX backup config.yaml failed
    goto fail_clean
  )
  set "BAK_CONFIG=%REMOTE_DIR%/server/config.yaml.bak.%TS%"
  echo   OK backup: !BAK_CONFIG!
) else (
  echo ==^> 1/7 skip backup ^(fresh / --no-backup^)
)

rem ---------- 2/7 upload + verify ----------
echo ==^> 2/7 upload
if "%DRY_RUN%"=="1" (
  echo      dry-run: scp %PKG% -^> %SSH_USER%@%SSH_HOST%:%REMOTE_TGZ%
) else (
  node "%SSH_RUN%" put "%PKG%" "%REMOTE_TGZ%"
  set "PUT_RC=!errorlevel!"
  chcp 65001 >nul
  if not "!PUT_RC!"=="0" (
    echo   XX upload failed
    goto fail_clean
  )
  call :rsh_read "md5sum '%REMOTE_TGZ%' | awk '{print $1}'"
  set "REMOTE_MD5=!RH_OUT!"
  call :rsh_read "wc -c < '%REMOTE_TGZ%'"
  set "REMOTE_SIZE=!RH_OUT!"
  if not "!REMOTE_MD5!"=="%LOCAL_MD5%" (
    echo   XX md5 mismatch local=%LOCAL_MD5% remote=!REMOTE_MD5!
    goto fail_clean
  )
  if not "!REMOTE_SIZE!"=="%LOCAL_SIZE%" (
    echo   XX size mismatch local=%LOCAL_SIZE% remote=!REMOTE_SIZE!
    goto fail_clean
  )
  echo   OK md5 + size match
)

if "%NO_RESTART%"=="1" (
  echo ==^> done --no-restart mode: package in place, server untouched
  goto done
)

rem ---------- 3/7 stop ----------
echo ==^> 3/7 stop WWII server
call :rsh "chmod +x %REMOTE_DIR%/*.sh 2>/dev/null; %REMOTE_DIR%/stop.sh 2>/dev/null; sleep 1; ss -lntp 2>/dev/null | grep ':%WWII_PORT% ' || echo '%WWII_PORT% free'"
if not "%RH_RC%"=="0" echo   WARN: stop.sh rc!=0 ^(maybe never started^), continuing

rem ---------- 4/7 extract ----------
if "%HAS_DIR%"=="yes" (
  echo ==^> 4/7 extract ^(exclude server/config.yaml, keep live config^)
  call :rsh "tar -xzf '%REMOTE_TGZ%' --exclude='WWIIbushu/server/config.yaml' -C /opt && echo 'extract ok'"
) else (
  echo ==^> 4/7 extract ^(fresh install, include template config.yaml^)
  call :rsh "tar -xzf '%REMOTE_TGZ%' -C /opt && echo 'extract ok'"
)
if not "%RH_RC%"=="0" (
  echo   XX extract failed, check tar output above
  goto fail_clean
)
call :rsh "chmod +x %REMOTE_DIR%/*.sh %REMOTE_DIR%/server/server && echo 'chmod ok'"
if not "%RH_RC%"=="0" (
  echo   XX chmod failed
  goto fail_clean
)
echo   OK extracted + exec bits set

rem ---------- 5/7 restore config ----------
echo ==^> 5/7 restore live config
REM Every branch below (backup / seed / fresh template) then runs wwii-patch-cfg.sh,
REM so MySQL host+password AND server.home_url (home server public URL) always end
REM up filled no matter which config path was restored. The patch is idempotent.
REM WWII mode connects to the home DB read-only; ezfy_mysql inherits host/user/password
REM from the mysql block, so patching host+password is enough (DB may be a remote RDS).
REM All quoting lives inside the bash script - cmd/bat can never pass a literal "
REM through call :rsh (a " inside the quoted arg closes the quote early and corrupts
REM the sed program / exposes pipes), so the patch logic MUST live in an uploaded file.
node "%SSH_RUN%" put "%SCRIPT_DIR%wwii-patch-cfg.sh" "%REMOTE_DIR%/wwii-patch-cfg.sh"
set "PUT_RC=!errorlevel!"
if not "!PUT_RC!"=="0" (
    echo   XX patch script upload failed
    goto fail_clean
)
call :rsh_read "sed -n 's/^[[:space:]]*port:[[:space:]]*\([0-9][0-9]*\).*/\1/p' /opt/Linuxbushu/server/config.yaml 2>/dev/null | head -n1"
set "HOME_PORT=!RH_OUT!"
if "!HOME_PORT!"=="" set "HOME_PORT=8080"

if not "%BAK_CONFIG%"=="" goto restore_from_backup
call :rsh_read "test -f '%CONFIG_SEED%' && echo yes || echo no"
if "%RH_OUT%"=="yes" goto restore_from_seed
REM no backup / no seed: fresh template config.yaml is already in place
goto run_patch

:restore_from_backup
call :rsh "cp -a '%BAK_CONFIG%' %REMOTE_DIR%/server/config.yaml && echo 'restored from backup'"
if not "%RH_RC%"=="0" goto fail_clean
echo   OK restored live config from backup
goto run_patch

:restore_from_seed
call :rsh "cp -a '%CONFIG_SEED%' %REMOTE_DIR%/server/config.yaml && echo 'restored from seed'"
if not "%RH_RC%"=="0" goto fail_clean
echo   OK restored live config from %CONFIG_SEED%

:run_patch
call :rsh "bash %REMOTE_DIR%/wwii-patch-cfg.sh %REMOTE_DIR%/server/config.yaml /opt/Linuxbushu/server/config.yaml http://%SSH_HOST%:!HOME_PORT!"
if "!RH_RC!"=="0" (
    echo   OK MySQL host/password + home_url filled from home config ^(http://%SSH_HOST%:!HOME_PORT!^)
) else (
    echo   WARN: patch script failed - edit %REMOTE_DIR%\server\config.yaml to set MySQL host/password before start
)

:after_restore

rem ---------- 6/7 start ----------
echo ==^> 6/7 start WWII server
call :rsh "cd %REMOTE_DIR% && ./start.sh"
if not "%RH_RC%"=="0" (
  echo   XX start script failed, log: %REMOTE_DIR%/logs/server.log
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log"
  echo !RH_OUT!
  goto fail_clean
)

rem ---------- 7/7 verify ----------
echo ==^> 7/7 verify
call :wait_port "%WWII_PORT%"
if not "!WP_RESULT!"=="OK" (
  echo   WARN: WWII %WWII_PORT% not listening yet ^(WWII cold start seeds the new DB, can take 60s+^)
  call :rsh_read "tail -n 8 %REMOTE_DIR%/logs/server.log"
  echo !RH_OUT!
  echo   NOTE: continuing anyway - re-run deploy-wwii.bat later to verify
) else (
  echo   OK WWII %WWII_PORT% listening
)
call :rsh_read "curl -s --max-time 8 http://127.0.0.1:%WWII_PORT%/api/ezfy/zone | head -c 400"
findstr /c:"code" "%TMPOUT%" >nul
if errorlevel 1 (
  echo   WARN: zone api not JSON ^(maybe still seeding^), check later
) else (
  echo   OK zone api JSON: !RH_OUT!
)

rem ---------- optional zone 2 ----------
if "%ZONE2%"=="1" (
  echo ==^> --zone2: clone zone 2 ^(%ZONE2_DIR%, port %ZONE2_PORT%, db qq_ezzt2, fresh^)
  call :rsh_read "test -d '%ZONE2_DIR%' && echo yes || echo no"
  if "!RH_OUT!"=="yes" (
    echo   %ZONE2_DIR% exists - syncing new binary/web/scripts ^(config.yaml kept^), restarting zone 2
    call :rsh "cp -f %REMOTE_DIR%/server/server %ZONE2_DIR%/server/server && cp -R %REMOTE_DIR%/web/dist/. %ZONE2_DIR%/web/dist/ && cp -f %REMOTE_DIR%/start.sh %REMOTE_DIR%/stop.sh %ZONE2_DIR%/ && chmod +x %ZONE2_DIR%/*.sh %ZONE2_DIR%/server/server"
    call :rsh "bash %REMOTE_DIR%/wwii-patch-cfg.sh %ZONE2_DIR%/server/config.yaml /opt/Linuxbushu/server/config.yaml http://%SSH_HOST%:!HOME_PORT! >/dev/null 2>&1; %ZONE2_DIR%/stop.sh 2>/dev/null; sleep 1; %ZONE2_DIR%/start.sh"
    if not "!RH_RC!"=="0" (
      echo   XX zone2 sync/restart failed
      call :rsh_read "tail -n 15 %ZONE2_DIR%/logs/server.log 2>/dev/null"
      echo !RH_OUT!
      goto fail_clean
    )
    ping -n 2 127.0.0.1 >nul 2>&1
    call :wait_port "%ZONE2_PORT%"
    if not "!WP_RESULT!"=="OK" (
      echo   WARN: zone2 %ZONE2_PORT% not listening yet - check %ZONE2_DIR%/logs/server.log
    ) else (
      echo   OK zone2 %ZONE2_PORT% listening ^(updated^)
    )
  ) else (
    call :rsh "cd %REMOTE_DIR% && bash ./zone2.sh && chmod +x %ZONE2_DIR%/*.sh %ZONE2_DIR%/server/server && %ZONE2_DIR%/start.sh"
    if not "!RH_RC!"=="0" (
      echo   XX zone2 clone/start failed
      call :rsh_read "tail -n 15 %ZONE2_DIR%/logs/server.log 2>/dev/null"
      echo !RH_OUT!
      goto fail_clean
    )
    ping -n 2 127.0.0.1 >nul 2>&1
    call :wait_port "%ZONE2_PORT%"
    if not "!WP_RESULT!"=="OK" (
      echo   WARN: zone2 %ZONE2_PORT% not listening yet - check %ZONE2_DIR%/logs/server.log
    ) else (
      echo   OK zone2 %ZONE2_PORT% listening
    )
  )
)

echo.
echo   WWII deployed:  http://%SSH_HOST%:%WWII_PORT%   ^(zone 1^)
if "%ZONE2%"=="1" echo   Zone 2:          http://%SSH_HOST%:%ZONE2_PORT%
echo   Next: home admin 'game hall' WWII zone entries - host use ":%WWII_PORT%" / ":%ZONE2_PORT%"
echo         (port-only = follow the visitor's domain; correct on EVERY machine sharing the same home DB)
goto done

rem =========================================================
rem subroutines
rem =========================================================
:mktimestamp
set "TS=%RANDOM%%RANDOM%"
goto :eof

:rsh
if "%DRY_RUN%"=="1" (
  echo      dry-run: ssh %SSH_USER%@%SSH_HOST%: "%~1"
  set "RH_RC=0"
  goto :eof
)
node "%SSH_RUN%" exec "%~1"
set "RH_RC=%errorlevel%"
chcp 65001 >nul
goto :eof

:rsh_read
if exist "%TMPOUT%" del /q "%TMPOUT%" >nul 2>&1
if exist "%TMPERR%" del /q "%TMPERR%" >nul 2>&1
node "%SSH_RUN%" exec "%~1" >"%TMPOUT%" 2>"%TMPERR%"
set "RH_RC=%errorlevel%"
chcp 65001 >nul
set "RH_OUT="
if exist "%TMPOUT%" set /p "RH_OUT="<"%TMPOUT%"
goto :eof

rem ---------- wait_port <port> [max_rounds] ----------
rem Polls the remote port until listening (WWII startup can take ~45s:
rem AutoMigrate + seed on first run). Sets WP_RESULT=OK|TIMEOUT.
:wait_port
set "WP_PORT=%~1"
set "WP_MAX=%~2"
if "%WP_MAX%"=="" set "WP_MAX=22"
set "WP_N=0"
set "WP_RESULT=TIMEOUT"
:wp_loop
set /a WP_N+=1
ping -n 3 127.0.0.1 >nul 2>&1
call :rsh_read "ss -lntp 2>/dev/null | grep ':%WP_PORT% ' >/dev/null && echo LISTEN_OK || echo LISTEN_FAIL"
if "!RH_OUT!"=="LISTEN_OK" (
  set "WP_RESULT=OK"
  goto :eof
)
if %WP_N% GEQ %WP_MAX% goto :eof
goto wp_loop

:usage
echo deploy-wwii.bat - deploy WWII independent package to a Linux host
echo.
echo   tools\deploy-wwii.bat -p PASS -h HOST ^(options^)
echo     -f FILE        package, default ..\WWIIbushu.tar.gz
echo     -h HOST        ssh host, default 39.105.151.141
echo     -u USER        ssh user, default root
echo     -p PASS        ssh password ^(or env SSH_PASS^)
echo     --no-restart   upload + verify only
echo     --no-backup    skip config backup
echo     --zone2        also clone + start zone 2 ^(/opt/wwii-2, 8082, qq_ezzt2 fresh^)
echo     --dry-run      plan only
echo.
echo   Examples:
echo     tools\deploy-wwii.bat -p 'Admin123mzd..' -h 124.222.208.92 --no-backup
echo     tools\deploy-wwii.bat -p 'Admin123mzd..' -h 124.222.208.92 --zone2 --no-backup
goto done

:fail_clean
if exist "%TMPOUT%" del /q "%TMPOUT%" >nul 2>&1
if exist "%TMPERR%" del /q "%TMPERR%" >nul 2>&1
endlocal
exit /b 1

:done
if exist "%TMPOUT%" del /q "%TMPOUT%" >nul 2>&1
if exist "%TMPERR%" del /q "%TMPERR%" >nul 2>&1
endlocal
exit /b 0
