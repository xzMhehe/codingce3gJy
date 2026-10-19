@echo off
chcp 65001 >nul
setlocal EnableExtensions EnableDelayedExpansion
title Home deployer

rem ============================================================================
rem  deploy.bat - deploy the Home (qqjiayuan) Linux package to one Linux host
rem
rem  Pairs with tools/deploy-wwii.bat (WWII independent package). This one
rem  targets the Home package (Linuxbushu.tar.gz) -> /opt/Linuxbushu, port 8080,
rem  and restarts the home server (brief downtime). WWII updates do NOT touch
rem  home, so prefer deploy-wwii.bat when only WWII changed.
rem
rem  Usage (from repo root qqjiayuan):
rem    tools\deploy.bat -p PASS -h HOST (options)
rem      -f FILE        package, default ..\Linuxbushu.tar.gz
rem      -h HOST        ssh host, default 39.105.151.141
rem      -u USER        ssh user, default root
rem      -p PASS        ssh password (or env SSH_PASS)
rem      --no-restart   upload + verify only, do NOT stop/restart the server
rem      --no-backup    skip full tar backup (still backs up server/config.yaml)
rem      --dry-run      plan only, no remote changes
rem
rem  Password priority: -p arg > env SSH_PASS > interactive prompt
rem
rem  Encoding note: this bat is pure ASCII on purpose. The original Chinese
rem  comments were destroyed by a double-mojibake corruption, so the file was
rem  rewritten in English (same style as deploy-wwii.bat) to be encoding-proof
rem  in any console / codepage.
rem ============================================================================

set "SCRIPT_DIR=%~dp0"
set "SSH_RUN=%SCRIPT_DIR%ssh-run.js"
set "PROJ_DIR=%SCRIPT_DIR%.."

if "%SSH_HOST%"=="" set "SSH_HOST=39.105.151.141"
if "%SSH_USER%"=="" set "SSH_USER=root"

set "PKG=%PROJ_DIR%\Linuxbushu.tar.gz"
set "REMOTE_DIR=/opt/Linuxbushu"
set "REMOTE_TGZ=/opt/Linuxbushu.tar.gz"
set "CONFIG_SEED=/opt/config.yaml"

set "NO_RESTART=0"
set "NO_BACKUP=0"
set "DRY_RUN=0"
set "TMPOUT=%TEMP%\deploy_out_%RANDOM%%RANDOM%.txt"
set "TMPERR=%TEMP%\deploy_err_%RANDOM%%RANDOM%.txt"

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

rem 2026-10-19 hardening: STRONG local package integrity pre-check via
rem tools/tgz_check.py (full gzip decompression + member enumeration).
rem Windows tar.exe can crash and leave a TRUNCATED tar.gz that md5 cannot
rem detect; the previous weak check (tarfile member count) also missed a
rem broken gzip footer.
where python >nul 2>nul
if errorlevel 1 (
  echo   WARN: python not found, skip local package integrity check
) else (
  python "%SCRIPT_DIR%tgz_check.py" "%PKG%" >nul 2>&1
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
  call :rsh_read "cat %REMOTE_DIR%/logs/server.pid 2>/dev/null"
  set "OLD_PID=!RH_OUT!"
  call :rsh_read "ss -lntp 2>/dev/null | grep -c ':8080 ' || true"
  echo   OK remote dir exists, pid=!OLD_PID!^(empty=never started^), 8080 listeners=!RH_OUT!
) else (
  echo   WARN: no %REMOTE_DIR% on remote - fresh install ^(nothing to back up^)
)

if "%NO_RESTART%"=="1" echo   WARN: --no-restart: upload + verify only, server untouched
if "%DRY_RUN%"=="1" echo ==^> DRY-RUN ends here

rem ---------- 1/7 backup ----------
set "BAK_CONFIG="
set "BAK_TGZ="
call :mktimestamp

if "%NO_RESTART%"=="0" if "%HAS_DIR%"=="yes" (
  echo ==^> 1/7 backup
  if "%NO_BACKUP%"=="1" (
    echo   WARN: --no-backup: full tar backup skipped, backing up server/config.yaml only
    call :rsh "cp -a %REMOTE_DIR%/server/config.yaml %REMOTE_DIR%/server/config.yaml.bak.%TS%"
    if not "!RH_RC!"=="0" (
      echo   XX backup config.yaml failed
      goto fail_clean
    )
  ) else (
    rem exclude logs/: server.log grows while tar runs; tar exits 1 when a file
    rem changed during read, so tolerate rc=1 and fail only on rc>1.
    call :rsh "tar -czf '/opt/Linuxbushu.bak.%TS%.tar.gz' --exclude='Linuxbushu/logs' -C /opt Linuxbushu; rc=$?; if [ $rc -gt 1 ]; then exit $rc; fi; [ -s '/opt/Linuxbushu.bak.%TS%.tar.gz' ] && ls -la '/opt/Linuxbushu.bak.%TS%.tar.gz'"
    if not "!RH_RC!"=="0" (
      echo   XX full backup failed ^(rc=!RH_RC!^)
      goto fail_clean
    )
    call :rsh "cp -a %REMOTE_DIR%/server/config.yaml %REMOTE_DIR%/server/config.yaml.bak.%TS%"
    if not "!RH_RC!"=="0" (
      echo   XX backup config.yaml failed
      goto fail_clean
    )
    set "BAK_TGZ=/opt/Linuxbushu.bak.%TS%.tar.gz"
    echo   OK full backup: !BAK_TGZ! ^(logs/ excluded^)
  )
  set "BAK_CONFIG=%REMOTE_DIR%/server/config.yaml.bak.%TS%"
) else (
  echo ==^> 1/7 skip backup ^(fresh install / --no-restart^)
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
  echo   OK md5 + size match ^(!REMOTE_MD5!^)
)

if "%NO_RESTART%"=="1" (
  echo ==^> done --no-restart mode: package in place, server untouched
  echo       apply later by re-running: tools\deploy.bat
  goto done
)

rem ---------- 3/7 stop ----------
echo ==^> 3/7 stop home server
call :rsh "chmod +x %REMOTE_DIR%/*.sh 2>/dev/null; %REMOTE_DIR%/stop.sh; sleep 1; ss -lntp 2>/dev/null | grep ':8080 ' || echo '8080 free'"
if not "%RH_RC%"=="0" echo   WARN: stop.sh rc!=0 ^(maybe never started^), continuing

rem ---------- 4/7 extract ----------
echo ==^> 4/7 extract ^(exclude server/config.yaml, keep live config^)
call :rsh "tar -xzf '%REMOTE_TGZ%' --exclude='Linuxbushu/server/config.yaml' -C /opt && echo 'extract ok'"
if not "%RH_RC%"=="0" (
  echo   XX extract failed, rollback hint: tar -xzf %BAK_TGZ% -C /opt
  goto fail_clean
)
call :rsh "chmod +x %REMOTE_DIR%/*.sh && echo 'chmod sh ok'"
if not "%RH_RC%"=="0" (
  echo   XX chmod *.sh failed
  goto fail_clean
)
echo   OK extracted + exec bits set

rem ---------- 5/7 restore config ----------
echo ==^> 5/7 restore live config
if not "%BAK_CONFIG%"=="" goto restore_from_backup
call :rsh_read "test -f '%CONFIG_SEED%' && echo yes || echo no"
if "%RH_OUT%"=="yes" goto restore_from_seed
echo   WARN: no backup and no seed - template config.yaml kept ^(edit web_dir + MySQL password^)
goto after_restore

:restore_from_backup
call :rsh "cp -a '%BAK_CONFIG%' %REMOTE_DIR%/server/config.yaml && echo 'restored from backup'"
if not "%RH_RC%"=="0" goto fail_clean
echo   OK restored live config from backup
goto after_restore

:restore_from_seed
call :rsh "cp -a '%CONFIG_SEED%' %REMOTE_DIR%/server/config.yaml"
if not "%RH_RC%"=="0" goto fail_clean
echo   OK restored live config from %CONFIG_SEED%

:after_restore
call :rsh "chown -R root:root %REMOTE_DIR%"
if not "%RH_RC%"=="0" echo   WARN: chown failed ^(macOS-built tar may carry uid 501^), continuing

rem ---------- 6/7 start ----------
echo ==^> 6/7 start home server
call :rsh "chmod +x %REMOTE_DIR%/*.sh %REMOTE_DIR%/server/server %REMOTE_DIR%/server/dbinit %REMOTE_DIR%/server/ezfymigrate 2>/dev/null; cd %REMOTE_DIR% && ./start.sh"
if not "%RH_RC%"=="0" (
  echo   XX start failed, log: %REMOTE_DIR%/logs/server.log
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log"
  echo !RH_OUT!
  goto fail_clean
)

rem ---------- 7/7 verify ----------
echo ==^> 7/7 verify
call :wait_port "8080"
if not "!WP_RESULT!"=="OK" (
  echo   XX port 8080 not listening
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log"
  echo !RH_OUT!
  goto fail_clean
)
echo   OK 8080 listening

call :rsh_read "curl -sI --max-time 8 http://127.0.0.1:8080/ | head -1"
echo !RH_OUT! | findstr /c:" 200 " >nul
if errorlevel 1 (
  echo   XX HTTP not 200 ^(!RH_OUT!^)
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log"
  echo !RH_OUT!
  goto fail_clean
)
echo   OK HTTP 200

call :rsh_read "curl -s --max-time 8 http://127.0.0.1:8080/api/games/ezfy/view | head -c 400"
findstr /c:"code" "%TMPOUT%" >nul
if errorlevel 1 goto verify_not_json
echo   OK API JSON
goto after_verify

:verify_not_json
echo   WARN: API not JSON ^(maybe serving index.html^), check manually
echo   !RH_OUT!

:after_verify
call :rsh_read "tail -n 8 %REMOTE_DIR%/logs/server.log"
if exist "%TMPOUT%" type "%TMPOUT%"

echo.
echo   Home deployed:  http://%SSH_HOST%:8080
if not "%BAK_TGZ%"=="" (
  echo   Backup:  %BAK_TGZ%
  echo   Rollback: tar -xzf %BAK_TGZ% -C /opt
) else (
  echo   No full backup created ^(--no-backup / fresh install^)
)
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
rem Polls the remote port until listening. Sets WP_RESULT=OK|TIMEOUT.
:wait_port
set "WP_PORT=%~1"
set "WP_MAX=%~2"
if "%WP_MAX%"=="" set "WP_MAX=15"
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
echo deploy.bat - deploy Home (qqjiayuan) Linux package to a Linux host
echo.
echo   tools\deploy.bat -p PASS -h HOST ^(options^)
echo     -f FILE        package, default ..\Linuxbushu.tar.gz
echo     -h HOST        ssh host, default 39.105.151.141
echo     -u USER        ssh user, default root
echo     -p PASS        ssh password ^(or env SSH_PASS^)
echo     --no-restart   upload + verify only, do NOT restart home server
echo     --no-backup    skip full tar backup ^(still backs up server/config.yaml^)
echo     --dry-run      plan only
echo.
echo   Examples:
echo     CMD:         tools\deploy.bat -p PASS -h 39.105.151.141 --no-backup
echo     PowerShell:  .\tools\deploy.bat -p 'Admin123mzd..' -h 124.222.208.92 --no-backup
echo     Git Bash:    SSH_PASS='' ./tools/deploy.bat -h 39.105.151.141 --no-backup
echo.
echo   Password priority: -p arg ^> env SSH_PASS ^> interactive prompt
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
