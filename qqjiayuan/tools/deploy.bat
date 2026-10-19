@echo off
chcp 936 >nul
setlocal enabledelayedexpansion
title  - 

rem ============================================================================
rem  deploy.bat   Linux Windows 
rem
rem   tools/deploy.sh  Win11 / Win2012 
rem  nodetools/ssh-run.js ?ssh/scp
rem
rem  /
rem    tools\deploy.bat                      ..\Linuxbushu.tar.gz 
rem    tools\deploy.bat -f D:\pkg.tar.gz    ?
rem    tools\deploy.bat --no-restart         + 
rem    tools\deploy.bat --dry-run            + 
rem    tools\deploy.bat --no-backup         
rem    tools\deploy.bat -h 1.2.3.4 -u root  /
rem    tools\deploy.bat -p               SSH 
rem
rem  -p  >  SSH_PASS >  > ssh-run.js 
rem
rem    qqjiayuan  .bat  .sh
rem    CMD:         tools\deploy.bat -p  -h 39.105.151.141 --no-backup
rem    PowerShell:  .\tools\deploy.bat -p  -h 39.105.151.141 --no-backup
rem    Git Bash:    SSH_PASS='' ./tools/deploy.bat -h 39.105.151.141 --no-backup
rem
rem  ?`SSH_PASS='' `  **bash ** PowerShell ?
rem     "SSH_PASS=..." cmdlet
rem     PowerShell   -p  `$env:SSH_PASS=''; .\tools\deploy.bat ...`
rem      `SSH_PASS=x ` 
rem
rem  ? Windows  .bat  .sh2026-10-07 
rem     deploy.sh  deploy.bat  index  LF core.autocrlf=true
rem    checkout ?CRLF.bat  CRLF
rem     .sh  CRLF Git Bash  `$'\r': command not found` 
rem     Windows  deploy.batdeploy.sh  Linux/macOS 
rem
rem  
rem    0.     /  md5 /  /  / 
rem    1.    ?/opt/Linuxbushu -> /opt/Linuxbushu.bak.<?.tar.gz
rem    2.    scp  /opt/Linuxbushu.tar.gzmd5 
rem    3.    chmod +x *.sh && ./stop.sh Text file busy
rem    4.    tar -xzf ... --exclude=server/config.yaml
rem    5.     config.yaml  server/
rem    6.    chmod +x server/* && ./start.sh
rem    7.    8080  + curl 200 + /api  JSON index.html 
rem
rem  
rem    chmod +x /opt/Linuxbushu/*.sh; /opt/Linuxbushu/stop.sh
rem    rm -rf /opt/Linuxbushu
rem    tar -xzf /opt/Linuxbushu.bak.<?.tar.gz -C /opt
rem    chmod +x /opt/Linuxbushu/server/*; /opt/Linuxbushu/start.sh
rem
rem  ?2026-09-28 
rem  1if / else ?***
rem         `()` `XX ()` ?`)`  cmd 
rem         ?if    if/else  
rem         cmd  "was unexpected at this time." ****
rem          ^( ^)  3 
rem          `(`?:rsh / :rsh_read / :usage 
rem  2 if/else ** !var! **
rem         `for ... do ( if not defined RH_OUT set "RH_OUT=%%i" )`   for 
rem         `if not defined RH_OUT` ?
rem         RH_OUT ****
rem         rsh_read  for 
rem         `set /p RH_OUT=<` 
rem  3`quit` ?cmd /   goto :eof
rem  4?
rem          echo ?PS  PowerShell ?
rem         ?
rem         
rem           '.js' is not recognized as an internal or external command
rem           'eploy.bat' is not recognized as an internal or external command
rem          bug WARN: ?
rem  ?2026-10-07 ** rem ** 
rem      cmd  if 
rem        ?
rem       cat / type  PS ?
rem      if 
rem     ?cat / type 
rem  5?if( ) **** %VAR%  call ?
rem         cmd  if ?RH_RC% 
rem         ****  `if not "%RH_RC%"=="0" goto fail` 
rem          `!RH_RC!` setlocal enabledelayedexpansion
rem          (1/7)?if  !RH_RC!?%RH_RC%
rem           // %RH_RC%?
rem  6  CRLF ** ASCII **
rem         ?`grep -c $'\r' deploy.bat`  cmd ?
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

rem ----------  ----------
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
echo   XX : %~1   --help 
exit /b 1

:parsed
if not exist "%SSH_RUN%" (
  echo   XX  %SSH_RUN%
  exit /b 1
)
rem  node  PATH  :rsh_read 
rem   SSH   
node -v >nul 2>&1
if errorlevel 1 (
  echo   XX  node ,  Node.js  node.exe  PATH
  exit /b 1
)

echo ==^>   %SSH_USER%@%SSH_HOST%:%REMOTE_DIR%
if "%DRY_RUN%"=="1" echo   WARN: DRY-RUN : , ?

rem =========================================================
rem 0. 
rem =========================================================
echo ==^> 0/7 

rem   if not defined  if "%SSH_PASS%"==""
rem   ^& ^| ^< ?  if defined ?
rem   
rem  ?^( ^)  if( ) ?
rem ? if  ^( ^) set /p 
rem     ?^  if + goto 
if defined SSH_PASS goto pass_ready
set /p "SSH_PASS=SSH  (%SSH_USER%@%SSH_HOST%, ): "
:pass_ready

if not exist "%PKG%" (
  echo   XX ? %PKG%
  exit /b 1
)
for %%A in ("%PKG%") do set "LOCAL_SIZE=%%~zA"

set "LOCAL_MD5="
for /f "skip=1 delims=" %%i in ('certutil -hashfile "%PKG%" MD5 2^>nul') do (
  if not defined LOCAL_MD5 set "LOCAL_MD5=%%i"
)
set "LOCAL_MD5=%LOCAL_MD5: =%"
if "%LOCAL_MD5%"=="" (
  echo   XX  md5certutil 
  exit /b 1
)

set /a LOCAL_MB=%LOCAL_SIZE%/1048576

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
echo   OK ?%PKG%  %LOCAL_MB% MB  md5=%LOCAL_MD5%

call :rsh_read "echo ping"
if "%RH_OUT%"=="ping" goto ssh_ok
echo   XX SSH  %SSH_USER%@%SSH_HOST%
echo       ssh-run.js ? ?
if exist "%TMPERR%" type "%TMPERR%"
if exist "%TMPOUT%" type "%TMPOUT%"
exit /b 1
:ssh_ok
echo   OK SSH 

call :rsh_read "df -P /opt | tail -1 | awk '{print $4}'"
set "DISK_AVAIL=%RH_OUT%"
if "%DISK_AVAIL%"=="" (
  echo   XX  /opt 
  exit /b 1
)
set /a AVAIL_MB=%DISK_AVAIL%/1024
set /a NEED_MB=%LOCAL_SIZE%/1048576+120
if %AVAIL_MB% LSS %NEED_MB% (
  echo   XX /opt  %AVAIL_MB%MB, ^( %NEED_MB%MB^)
  exit /b 1
)
echo   OK /opt  %AVAIL_MB%MB ^( %NEED_MB%MB^)

call :rsh_read "test -d '%REMOTE_DIR%' && echo yes || echo no"
set "HAS_DIR=%RH_OUT%"
if "%HAS_DIR%"=="yes" (
  call :rsh_read "cat %REMOTE_DIR%/logs/server.pid 2>/dev/null"
  set "OLD_PID=!RH_OUT!"
  call :rsh_read "ss -lntp 2>/dev/null | grep -c ':8080 ' || true"
  echo   OK , pid=!OLD_PID!^(=^), 8080 =!RH_OUT!
) else (
  echo   WARN:  %REMOTE_DIR%  ^(^)
)

if "%NO_RESTART%"=="1" echo   WARN: --no-restart:  + , 
if "%DRY_RUN%"=="1" echo ==^> DRY-RUN , :

rem ---------- 1.  ----------
set "BAK_CONFIG="
set "BAK_TGZ="
call :mktimestamp

if "%NO_RESTART%"=="0" if "%HAS_DIR%"=="yes" (
  echo ==^> 1/7 
  if "%NO_BACKUP%"=="1" (
    echo   WARN: --no-backup: ,  server/config.yaml
    call :rsh "cp -a %REMOTE_DIR%/server/config.yaml %REMOTE_DIR%/server/config.yaml.bak.%TS%"
    if not "!RH_RC!"=="0" (
      echo   XX  config.yaml 
      goto fail_clean
    )
  ) else (
    rem   logs/:  server.log,  tar  file changed as we read it
    rem  : tar ?1 , ,  deploy.sh
    call :rsh "tar -czf '/opt/Linuxbushu.bak.%TS%.tar.gz' --exclude='Linuxbushu/logs' -C /opt Linuxbushu; rc=$?; if [ $rc -gt 1 ]; then exit $rc; fi; [ -s '/opt/Linuxbushu.bak.%TS%.tar.gz' ] && ls -la '/opt/Linuxbushu.bak.%TS%.tar.gz'"
    if not "!RH_RC!"=="0" (
      echo   XX , ^(^)
      goto fail_clean
    )
    call :rsh "cp -a %REMOTE_DIR%/server/config.yaml %REMOTE_DIR%/server/config.yaml.bak.%TS%"
    if not "!RH_RC!"=="0" (
      echo   XX  config.yaml 
      goto fail_clean
    )
    set "BAK_TGZ=/opt/Linuxbushu.bak.%TS%.tar.gz"
    echo   OK : !BAK_TGZ! ^( logs/^)
  )
  set "BAK_CONFIG=%REMOTE_DIR%/server/config.yaml.bak.%TS%"
) else (
  echo ==^> 1/7   ^(^)
)

rem ---------- 2.  +  ----------
echo ==^> 2/7 ?
if "%DRY_RUN%"=="1" (
  echo      dry-run: scp %PKG% -^> %SSH_USER%@%SSH_HOST%:%REMOTE_TGZ%
) else (
  node "%SSH_RUN%" put "%PKG%" "%REMOTE_TGZ%"
  set "PUT_RC=!errorlevel!"
  chcp 936 >nul
  if not "!PUT_RC!"=="0" (
    echo   XX 
    goto fail_clean
  )
  call :rsh_read "md5sum '%REMOTE_TGZ%' | awk '{print $1}'"
  set "REMOTE_MD5=!RH_OUT!"
rem   stat -c %scall ?%s  s wc -c ?
call :rsh_read "wc -c < '%REMOTE_TGZ%'"
  set "REMOTE_SIZE=!RH_OUT!"
  if not "!REMOTE_MD5!"=="%LOCAL_MD5%" (
    echo   XX md5  =%LOCAL_MD5% =!REMOTE_MD5! ^(^)
    goto fail_clean
  )
  if not "!REMOTE_SIZE!"=="%LOCAL_SIZE%" (
    echo   XX  =%LOCAL_SIZE% =!REMOTE_SIZE! ^(^)
    goto fail_clean
  )
  echo   OK md5 +  ^(!REMOTE_MD5!^)
)

if "%NO_RESTART%"=="1" (
  echo ==^> ? --no-restart : , 
  echo     ? ^( --no-restart^): tools\deploy.bat
  goto done
)

rem ---------- 3.  ----------
echo ==^> 3/7 
rem  ?*.sh ,  chmod  Permission denied 
call :rsh "chmod +x %REMOTE_DIR%/*.sh 2>/dev/null; %REMOTE_DIR%/stop.sh; sleep 1; ss -lntp 2>/dev/null | grep ':8080 ' || echo '8080 '"
if not "%RH_RC%"=="0" echo   WARN: stop.sh  0^(^), 

rem ---------- 4.  ----------
echo ==^> 4/7  ^( config.yaml^)
rem  ?tar ?macOS ?*.sh  644), ?chmod
call :rsh "tar -xzf '%REMOTE_TGZ%' --exclude='Linuxbushu/server/config.yaml' -C /opt && echo 'extract ok'"
if not "%RH_RC%"=="0" (
  echo   XX   , : tar -xzf %BAK_TGZ% -C /opt
  goto fail_clean
)
call :rsh "chmod +x %REMOTE_DIR%/*.sh && echo 'chmod sh ok'"
if not "%RH_RC%"=="0" (
  echo   XX chmod *.sh 
  goto fail_clean
)
echo   OK ?+  *.sh 

rem ---------- 5.  ----------
echo ==^> 5/7 
if not "%BAK_CONFIG%"=="" goto restore_from_backup
call :rsh_read "test -f '%CONFIG_SEED%' && echo yes || echo no"
if "%RH_OUT%"=="yes" goto restore_from_seed
echo   WARN:  ( +  web_dir^), ?
goto after_restore

:restore_from_backup
call :rsh "cp -a '%BAK_CONFIG%' %REMOTE_DIR%/server/config.yaml && echo 'restored from backup'"
echo   OK 
goto after_restore

:restore_from_seed
call :rsh "cp -a '%CONFIG_SEED%' %REMOTE_DIR%/server/config.yaml"
echo   OK  %CONFIG_SEED% 

:after_restore
call :rsh "chown -R root:root %REMOTE_DIR%"
if not "%RH_RC%"=="0" echo   WARN: chown ^(macOS ?uid 501^), 

rem ---------- 6.  ----------
echo ==^> 6/7 
call :rsh "chmod +x %REMOTE_DIR%/*.sh %REMOTE_DIR%/server/server %REMOTE_DIR%/server/dbinit %REMOTE_DIR%/server/ezfymigrate 2>/dev/null; cd %REMOTE_DIR% && ./start.sh"
if not "%RH_RC%"=="0" (
  echo   XX , : %REMOTE_DIR%/logs/server.log
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log | iconv -f UTF-8 -t GBK -c 2>/dev/null"
  if exist "%TMPOUT%" type "%TMPOUT%"
  goto fail_clean
)

rem ---------- 7.  ----------
echo ==^> 7/7 
ping -n 7 127.0.0.1 >nul 2>&1
call :rsh_read "ss -lntp 2>/dev/null | grep ':8080 ' >/dev/null && echo LISTEN_OK || echo LISTEN_FAIL"
if not "%RH_OUT%"=="LISTEN_OK" (
  echo   XX 8080 
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log | iconv -f UTF-8 -t GBK -c 2>/dev/null"
  if exist "%TMPOUT%" type "%TMPOUT%"
  goto fail_clean
)
echo   OK 8080 

rem   curl -w '%{http_code}'call ? 
rem    -sI  HTTP/1.1 200 OK findstr  " 200 "
call :rsh_read "curl -sI --max-time 8 http://127.0.0.1:8080/ | head -1"
echo !RH_OUT! | findstr /c:" 200 " >nul
if errorlevel 1 (
  echo   XX  200 ^(!RH_OUT!^)
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log | iconv -f UTF-8 -t GBK -c 2>/dev/null"
  if exist "%TMPOUT%" type "%TMPOUT%"
  goto fail_clean
)
echo   OK  HTTP 200

rem : ?index.html  200,  JSON
call :rsh_read "curl -s --max-time 8 http://127.0.0.1:8080/api/games/ezfy/view | head -c 400 | iconv -f UTF-8 -t GBK -c 2>/dev/null"
findstr /c:"code" "%TMPOUT%" >nul
if errorlevel 1 goto verify_not_json
echo   OK  JSON  
goto after_verify

:verify_not_json
echo   WARN:  JSON^( index.html ^)  
echo %RH_OUT%

:after_verify
call :rsh_read "tail -n 8 %REMOTE_DIR%/logs/server.log | iconv -f UTF-8 -t GBK -c 2>/dev/null"
if exist "%TMPOUT%" type "%TMPOUT%"

echo.
echo ? http://%SSH_HOST%:8080
if not "%BAK_TGZ%"=="" (
  echo   : %BAK_TGZ%
  echo   : tar -xzf %BAK_TGZ% -C /opt
) else (
  echo   : 
)
goto done

rem =========================================================
rem 
rem =========================================================

rem :mktimestamp   TS
rem
rem ?
rem    `for /f "tokens=1-4 delims=/-. " %%a in ("%DATE% %TIME%")`
rem        Windows  %DATE% 2026/09/28 ?
rem      ?"" 
rem    `wmic OS get LocalDateTime`
rem       **Win11 24H2  wmic ** build 26200 
rem      
rem    ?
rem       cmd  + for 
rem
rem  **?*
rem    %RANDOM% 
rem    100% 
rem   %TIME% ?
rem      echo 
:mktimestamp
set "TS=%RANDOM%%RANDOM%"
goto :eof

rem :rsh ""   , DRY_RUN 
:rsh
if "%DRY_RUN%"=="1" (
  echo      dry-run: ssh %SSH_USER%@%SSH_HOST%: "%~1"
  set "RH_RC=0"
  goto :eof
)
node "%SSH_RUN%" exec "%~1"
set "RH_RC=%errorlevel%"
rem  node  Windows ?UTF-8
rem   ?echo    node  936
chcp 936 >nul
goto :eof

rem :rsh_read ""  , DRY_RUN ^()
rem  ?*** RH_OUT?RH_RC
rem   ?set /p2 for /f  if
:rsh_read
if exist "%TMPOUT%" del /q "%TMPOUT%" >nul 2>&1
if exist "%TMPERR%" del /q "%TMPERR%" >nul 2>&1
node "%SSH_RUN%" exec "%~1" >"%TMPOUT%" 2>"%TMPERR%"
set "RH_RC=%errorlevel%"
chcp 936 >nul
set "RH_OUT="
if exist "%TMPOUT%" set /p "RH_OUT="<"%TMPOUT%"
goto :eof

rem :parse_pass   -p/--pass <>
rem     `if ... ( set "SSH_PASS=%~2" & shift )`
rem     ?`)`  `(`?if 1
rem      `set "SSH_PASS=..."` 
:parse_pass
set "SSH_PASS=%~2"
shift
shift
goto parse

:usage
echo deploy.bat   Linux 
echo.
echo : qqjiayuan\tools\deploy.bat
echo   ? 
echo    git ?codingce3gJy, ?qqjiayuan\tools\, ?.\tools\deploy.bat
echo.
echo   tools\deploy.bat                      ..\Linuxbushu.tar.gz 
echo   tools\deploy.bat -f D:\pkg.tar.gz    ?
echo   tools\deploy.bat -h 1.2.3.4 -u root  /
echo   tools\deploy.bat -p               SSH 
echo   tools\deploy.bat --no-restart         + 
echo   tools\deploy.bat --no-backup         
echo   tools\deploy.bat --dry-run            + 
echo   tools\deploy.bat --help              
echo.
echo : -p  ^>  SSH_PASS ^>  ^> ssh-run.js 
echo.
echo ^( qqjiayuan ^):
echo   CMD:         tools\deploy.bat -p  -h 39.105.151.141 --no-backup
echo   PowerShell:  .\tools\deploy.bat -p  -h 39.105.151.141 --no-backup
echo   PowerShell:  $env:SSH_PASS=''; .\tools\deploy.bat -h 39.105.151.141 --no-backup
echo   Git Bash:    SSH_PASS='' ./tools/deploy.bat -h 39.105.151.141 --no-backup
echo.
echo : SSH_PASS=''  Git Bash ; PowerShell  CommandNotFoundException
echo       CMD / PowerShell  -p ,  $env:SSH_PASS=''
echo.
echo :  ^& ^| ^^ ,  -p "a^&b"
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
