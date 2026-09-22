@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion
title 家园社区 - 部署到线上

rem ============================================================================
rem  deploy.bat —— 一键推送并部署「家园社区」到线上 Linux 服务器（Windows 版）
rem
rem  与 tools/deploy.sh 等价，逻辑与步骤完全一致，给 Win11 / Win2012 用。
rem  依赖：node（tools/ssh-run.js 会调用 ssh/scp）+ curl(可选，仅本地预检用)
rem
rem  用法（在本目录或项目根目录双击/命令行执行都行）：
rem    tools\deploy.bat                     默认用 ..\Linuxbushu.tar.gz 全量部署
rem    tools\deploy.bat -f D:\pkg.tar.gz    指定部署包
rem    tools\deploy.bat --no-restart        只上传 + 校验，不停服不解压
rem    tools\deploy.bat --dry-run           只打印计划 + 预检，不改任何东西
rem    tools\deploy.bat --no-backup         跳过备份（省空间，不推荐）
rem    tools\deploy.bat -h 1.2.3.4 -u root  指定主机/用户
rem
rem  密码：优先读环境变量 SSH_PASS；没设置就交互输入，直接回车用 ssh-run.js 内置默认值
rem        set SSH_PASS=xxx && tools\deploy.bat
rem
rem  部署步骤（与线上既有约定一致）：
rem    0. 预检   本地包存在 / 本地 md5 / 远端连通 / 磁盘余量 / 当前服务状态
rem    1. 备份   打包当前 /opt/Linuxbushu -> /opt/Linuxbushu.bak.<MMDD-HHMM>.tar.gz
rem    2. 上传   scp 到 /opt/Linuxbushu.tar.gz，md5 双向校验（不一致立即中止）
rem    3. 停服   chmod +x *.sh && ./stop.sh（必须先停再解压，否则 Text file busy）
rem    4. 解压   tar -xzf ... --exclude=server/config.yaml（包里是模板配置）
rem    5. 恢复   把备份的真实 config.yaml 放回 server/
rem    6. 启动   chmod +x server/* && ./start.sh
rem    7. 验证   8080 在监听 + curl 200 + /api 返回 JSON（不是 index.html 回落）
rem
rem  回滚（在服务器上手动执行）：
rem    chmod +x /opt/Linuxbushu/*.sh; /opt/Linuxbushu/stop.sh
rem    rm -rf /opt/Linuxbushu
rem    tar -xzf /opt/Linuxbushu.bak.<时间戳>.tar.gz -C /opt
rem    chmod +x /opt/Linuxbushu/server/*; /opt/Linuxbushu/start.sh
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
set "TMPOUT=%TEMP%\deploy_out_%RANDOM%.txt"

rem ---------- 参数 ----------
:parse
if "%~1"=="" goto parsed
if /i "%~1"=="-f"         ( set "PKG=%~2"      & shift & shift & goto parse )
if /i "%~1"=="--file"     ( set "PKG=%~2"      & shift & shift & goto parse )
if /i "%~1"=="-h"         ( set "SSH_HOST=%~2" & shift & shift & goto parse )
if /i "%~1"=="--host"     ( set "SSH_HOST=%~2" & shift & shift & goto parse )
if /i "%~1"=="-u"         ( set "SSH_USER=%~2" & shift & shift & goto parse )
if /i "%~1"=="--user"     ( set "SSH_USER=%~2" & shift & shift & goto parse )
if /i "%~1"=="--no-restart" ( set "NO_RESTART=1" & shift & goto parse )
if /i "%~1"=="--no-backup"  ( set "NO_BACKUP=1"  & shift & goto parse )
if /i "%~1"=="--dry-run"    ( set "DRY_RUN=1"    & shift & goto parse )
if /i "%~1"=="--help"       goto usage
if /i "%~1"=="-?"           goto usage
echo   XX 未知参数: %~1  （用 --help 看用法）
exit /b 1

:parsed
if not exist "%SSH_RUN%" (
  echo   XX 找不到 %SSH_RUN%
  exit /b 1
)

echo ==^> 部署目标  %SSH_USER%@%SSH_HOST%:%REMOTE_DIR%
if "%DRY_RUN%"=="1" echo   [!] DRY-RUN 模式: 只做预检, 不会改任何东西

rem =========================================================
rem 0. 预检
rem =========================================================
echo ==^> 0/7 预检

if "%SSH_PASS%"=="" (
  set /p "SSH_PASS=SSH 密码 (%SSH_USER%@%SSH_HOST%, 直接回车用内置默认值): "
)

if not exist "%PKG%" (
  echo   XX 部署包不存在: %PKG%
  exit /b 1
)
for %%A in ("%PKG%") do set "LOCAL_SIZE=%%~zA"

set "LOCAL_MD5="
for /f "skip=1 delims=" %%i in ('certutil -hashfile "%PKG%" MD5') do (
  if not defined LOCAL_MD5 set "LOCAL_MD5=%%i"
)
set "LOCAL_MD5=%LOCAL_MD5: =%"
if "!LOCAL_MD5!"=="" (
  echo   XX 算不出本地 md5（certutil 失败）
  exit /b 1
)

set /a LOCAL_MB=%LOCAL_SIZE%/1048576
echo   OK 部署包 %PKG% 约 %LOCAL_MB% MB  md5=%LOCAL_MD5%

call :rsh_read "echo ping" 
if not "%RH_OUT%"=="ping" (
  echo   XX SSH 连不上 %SSH_USER%@%SSH_HOST% （密码错? 端口不通?）
  exit /b 1
)
echo   OK SSH 连通

call :rsh_read "df -P /opt | tail -1 | awk '{print $4}'"
set "DISK_AVAIL=%RH_OUT%"
if "%DISK_AVAIL%"=="" (
  echo   XX 读不到 /opt 磁盘余量
  exit /b 1
)
set /a AVAIL_MB=%DISK_AVAIL%/1024
set /a NEED_MB=%LOCAL_SIZE%/1048576+120
if %AVAIL_MB% LSS %NEED_MB% (
  echo   XX /opt 剩余 %AVAIL_MB%MB, 不够^(需约 %NEED_MB%MB^)
  exit /b 1
)
echo   OK /opt 剩余 %AVAIL_MB%MB ^(需约 %NEED_MB%MB^)

call :rsh_read "test -d '%REMOTE_DIR%' && echo yes || echo no"
if "%RH_OUT%"=="yes" (
  call :rsh_read "cat %REMOTE_DIR%/logs/server.pid 2>/dev/null"
  set "OLD_PID=!RH_OUT!"
  call :rsh_read "ss -lntp 2>/dev/null | grep -c ':8080 ' || true"
  echo   OK 线上已有部署, pid=!OLD_PID!（空=没在跑）, 8080 监听数=!RH_OUT!
) else (
  echo   [!] 线上没有 %REMOTE_DIR% —— 这是全新部署^(没有旧配置可保留^)
)

if "%NO_RESTART%"=="1" echo   [!] --no-restart: 只上传 + 校验, 到此为止
if "%DRY_RUN%"=="1" echo ==^> DRY-RUN 结束, 下面是完整计划:

rem ---------- 1. 备份 ----------
set "BAK_CONFIG="
set "BAK_TGZ="
for /f "tokens=1-4 delims=/-. " %%a in ("%DATE% %TIME%") do set "TS=%%a%%b-%%c%%d"
rem 上面拿不到时退回随机后缀，保证备份名不重（Windows 区域设置差异很大）
if "%TS%"=="" set "TS=%RANDOM%"

call :rsh_read "test -d '%REMOTE_DIR%' && echo yes || echo no"
set "HAS_DIR=%RH_OUT%"

if "%NO_RESTART%"=="0" if "%HAS_DIR%"=="yes" (
  echo ==^> 1/7 备份线上现有部署
  if "%NO_BACKUP%"=="1" (
    echo   [!] --no-backup: 跳过整包备份, 只留 server/config.yaml
    call :rsh "cp -a %REMOTE_DIR%/server/config.yaml %REMOTE_DIR%/server/config.yaml.bak.%TS%"
  ) else (
    rem ★ 必须排除 logs/: 服务正在写 server.log, 不排除 tar 会报 file changed as we read it
    call :rsh "tar -czf '/opt/Linuxbushu.bak.%TS%.tar.gz' --exclude='Linuxbushu/logs' -C /opt Linuxbushu; rc=$?; if [ $rc -gt 1 ]; then exit $rc; fi; ls -la '/opt/Linuxbushu.bak.%TS%.tar.gz'"
    call :rsh "cp -a %REMOTE_DIR%/server/config.yaml %REMOTE_DIR%/server/config.yaml.bak.%TS%"
    set "BAK_TGZ=/opt/Linuxbushu.bak.%TS%.tar.gz"
    echo   OK 备份: !BAK_TGZ! ^(不含 logs/^)
  )
  set "BAK_CONFIG=%REMOTE_DIR%/server/config.yaml.bak.%TS%"
) else (
  echo ==^> 1/7 备份  ^(跳过^)
)

rem ---------- 2. 上传 + 校验 ----------
echo ==^> 2/7 上传部署包
if "%DRY_RUN%"=="1" (
  echo      [dry-run] scp %PKG% -^> %SSH_USER%@%SSH_HOST%:%REMOTE_TGZ%
) else (
  node "%SSH_RUN%" put "%PKG%" "%REMOTE_TGZ%"
  if errorlevel 1 (
    echo   XX 上传失败
    goto fail_clean
  )
  call :rsh_read "md5sum '%REMOTE_TGZ%' | awk '{print $1}'"
  set "REMOTE_MD5=!RH_OUT!"
  call :rsh_read "stat -c %%s '%REMOTE_TGZ%'"
  set "REMOTE_SIZE=!RH_OUT!"
  if not "!REMOTE_MD5!"=="!LOCAL_MD5!" (
    echo   XX md5 不一致! 本地=!LOCAL_MD5! 远端=!REMOTE_MD5! ^(线上未改动^)
    goto fail_clean
  )
  if not "!REMOTE_SIZE!"=="!LOCAL_SIZE!" (
    echo   XX 字节数不一致! 本地=!LOCAL_SIZE! 远端=!REMOTE_SIZE! ^(线上未改动^)
    goto fail_clean
  )
  echo   OK md5 + 字节数双向校验一致 ^(!REMOTE_MD5!^)
)

if "%NO_RESTART%"=="1" (
  echo ==^> 完成  --no-restart 模式: 包已就位, 线上服务未受影响
  echo     要完成部署, 再跑一次^(去掉 --no-restart^): tools\deploy.bat
  goto done
)

rem ---------- 3. 停服 ----------
echo ==^> 3/7 停服
rem ★ 首次部署后 *.sh 常常没有执行位, 不 chmod 会 Permission denied 却以为停成功了
call :rsh "chmod +x %REMOTE_DIR%/*.sh 2>/dev/null; %REMOTE_DIR%/stop.sh; sleep 1; ss -lntp 2>/dev/null | grep ':8080 ' || echo '8080 已释放'"

rem ---------- 4. 解压 ----------
echo ==^> 4/7 解压 ^(排除模板 config.yaml^)
rem ★ 解压会把 tar 里的权限位盖回去(macOS 打包的 *.sh 常是 644), 解压完必须再补 chmod
call :rsh "tar -xzf '%REMOTE_TGZ%' --exclude='Linuxbushu/server/config.yaml' -C /opt && echo 'extract ok'"
if not "!RH_RC!"=="0" (
  echo   XX 解压失败 —— 线上现在没在跑, 用备份回滚: tar -xzf !BAK_TGZ! -C /opt
  goto fail_clean
)
call :rsh "chmod +x %REMOTE_DIR%/*.sh && echo 'chmod sh ok'"
if not "!RH_RC!"=="0" (
  echo   XX chmod *.sh 失败
  goto fail_clean
)
echo   OK 解压完成 + 补回 *.sh 执行位

rem ---------- 5. 恢复配置 ----------
echo ==^> 5/7 恢复真实配置
if not "%BAK_CONFIG%"=="" (
  call :rsh "cp -a '%BAK_CONFIG%' %REMOTE_DIR%/server/config.yaml && echo 'restored from backup'"
  echo   OK 已用备份的真实配置覆盖模板
) else (
  call :rsh_read "test -f '%CONFIG_SEED%' && echo yes || echo no"
  if "!RH_OUT!"=="yes" (
    call :rsh "cp -a '%CONFIG_SEED%' %REMOTE_DIR%/server/config.yaml"
    echo   OK 已用 %CONFIG_SEED% 覆盖模板
  ) else (
    echo   [!] 找不到真实配置! 包里的是模板^(占位密码 + 旧 web_dir^), 服务会起不来
  )
)
call :rsh "chown -R root:root %REMOTE_DIR%"

rem ---------- 6. 启动 ----------
echo ==^> 6/7 启动
call :rsh "chmod +x %REMOTE_DIR%/*.sh %REMOTE_DIR%/server/server %REMOTE_DIR%/server/dbinit %REMOTE_DIR%/server/ezfymigrate 2>/dev/null; cd %REMOTE_DIR% && ./start.sh"
if not "!RH_RC!"=="0" (
  echo   XX 启动脚本失败, 看日志: %REMOTE_DIR%/logs/server.log
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log"
  goto fail_clean
)

rem ---------- 7. 验证 ----------
echo ==^> 7/7 验证
ping -n 7 127.0.0.1 >nul 2>&1
call :rsh_read "ss -lntp 2>/dev/null | grep ':8080 ' >/dev/null && echo LISTEN_OK || echo LISTEN_FAIL"
if not "!RH_OUT!"=="LISTEN_OK" (
  echo   XX 8080 没在监听
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log"
  echo !RH_OUT!
  goto fail_clean
)
echo   OK 8080 已监听

call :rsh_read "curl -s -o /dev/null -w 'HTTP=%%{http_code}' --max-time 8 http://127.0.0.1:8080/"
if not "!RH_OUT!"=="HTTP=200" (
  echo   XX 首页不是 200 ^(!RH_OUT!^)
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log"
  goto fail_clean
)
echo   OK 首页 HTTP 200

rem 版本指纹: 未知路径会回落 index.html 返 200, 所以必须看响应体是不是 JSON
call :rsh_read "curl -s --max-time 8 http://127.0.0.1:8080/api/games/ezfy/view | head -c 60"
echo(!RH_OUT! | findstr /c:"code" >nul
if errorlevel 1 (
  echo   [!] 接口返回的不是 JSON^(可能是 index.html 回落^) —— 确认下版本
  echo(!RH_OUT!
) else (
  echo   OK 接口返回 JSON —— 新版本已生效
)

call :rsh_read "tail -n 8 %REMOTE_DIR%/logs/server.log"
echo(!RH_OUT!

echo.
echo 部署完成  http://%SSH_HOST%:8080
if not "%BAK_TGZ%"=="" (
  echo   备份: %BAK_TGZ%
  echo   回滚: tar -xzf %BAK_TGZ% -C /opt
) else (
  echo   备份: 无
)
goto done

rem =========================================================
rem 子过程
rem =========================================================

rem :rsh "远端命令"  —— 有副作用的命令, DRY_RUN 时只打印
:rsh
if "%DRY_RUN%"=="1" (
  echo      [dry-run] ssh %SSH_USER%@%SSH_HOST%: %~1
  set "RH_RC=0"
  goto :eof
)
node "%SSH_RUN%" exec "%~1"
set "RH_RC=%errorlevel%"
goto :eof

rem :rsh_read "远端命令" —— 只读命令, DRY_RUN 时也真执行^(预检需要真实结果^)
rem 结果放进 RH_OUT^(取第一行^), 退出码放进 RH_RC
:rsh_read
node "%SSH_RUN%" exec "%~1" >"%TMPOUT%" 2>nul
set "RH_RC=%errorlevel%"
set "RH_OUT="
for /f "usebackq delims=" %%i in ("%TMPOUT%") do (
  if not defined RH_OUT set "RH_OUT=%%i"
)
goto :eof

:usage
echo deploy.bat —— 一键推送并部署「家园社区」到线上 Linux 服务器
echo.
echo   tools\deploy.bat                     默认用 ..\Linuxbushu.tar.gz 全量部署
echo   tools\deploy.bat -f D:\pkg.tar.gz    指定部署包
echo   tools\deploy.bat --no-restart        只上传 + 校验，不停服不解压
echo   tools\deploy.bat --dry-run           只打印计划 + 预检，不改任何东西
echo   tools\deploy.bat --no-backup         跳过备份（省空间，不推荐）
echo   tools\deploy.bat -h 1.2.3.4 -u root  指定主机/用户
echo   tools\deploy.bat --help              本帮助
echo.
echo 密码: 优先读环境变量 SSH_PASS；没设置就交互输入，回车用内置默认值
echo   set SSH_PASS=xxx ^&^& tools\deploy.bat
goto done

:fail_clean
if exist "%TMPOUT%" del /q "%TMPOUT%" >nul 2>&1
endlocal
exit /b 1

:done
if exist "%TMPOUT%" del /q "%TMPOUT%" >nul 2>&1
endlocal
exit /b 0
