@echo off
chcp 936 >nul
setlocal enabledelayedexpansion
title 家园社区 - 部署到线上

rem ============================================================================
rem  deploy.bat —— 一键推送并部署「家园社区」到线上 Linux 服务器（Windows 版）
rem
rem  与 tools/deploy.sh 等价，逻辑与步骤完全一致，给 Win11 / Win2012 用。
rem  依赖：node（tools/ssh-run.js 会调用 ssh/scp）
rem
rem  用法（在本目录或项目根目录双击/命令行执行都行）：
rem    tools\deploy.bat                     默认用 ..\Linuxbushu.tar.gz 全量部署
rem    tools\deploy.bat -f D:\pkg.tar.gz    指定部署包
rem    tools\deploy.bat --no-restart        只上传 + 校验，不停服不解压
rem    tools\deploy.bat --dry-run           只打印计划 + 预检，不改任何东西
rem    tools\deploy.bat --no-backup         跳过备份（省空间，不推荐）
rem    tools\deploy.bat -h 1.2.3.4 -u root  指定主机/用户
rem    tools\deploy.bat -p 密码             指定 SSH 密码
rem
rem  密码优先级：-p 参数 > 环境变量 SSH_PASS > 交互输入 > ssh-run.js 内置默认值
rem
rem  各终端执行方式（★ 必须在项目根目录 qqjiayuan 下，且用 .bat 不能用 .sh，见下）：
rem    CMD:         tools\deploy.bat -p 密码 -h 39.105.151.141 --no-backup
rem    PowerShell:  .\tools\deploy.bat -p 密码 -h 39.105.151.141 --no-backup
rem    Git Bash:    SSH_PASS='密码' ./tools/deploy.bat -h 39.105.151.141 --no-backup
rem
rem  ★★ `SSH_PASS='密码' 命令` 是 **bash 专属语法**，在 PowerShell 里会直接报
rem     「无法将"SSH_PASS=..."项识别为 cmdlet、函数、脚本文件或可运行程序的名称」。
rem     PowerShell 两条路：① 用 -p 参数；② `$env:SSH_PASS='密码'; .\tools\deploy.bat ...`
rem     （分号分隔，环境变量必须在前面单独设，不能写成 `SSH_PASS=x 命令` 那种前缀）。
rem
rem  ★★ 为什么 Windows 上要用 .bat 而不是 .sh（2026-10-07 查明）：
rem    仓库里 deploy.sh 与 deploy.bat 在 index 里都是 LF，但本机 core.autocrlf=true，
rem    checkout 到工作区后两个文件都变成 CRLF。.bat 必须 CRLF（正确），
rem    而 .sh 一旦是 CRLF，在 Git Bash 下会报 `$'\r': command not found` 直接跑不了。
rem    所以 Windows 侧统一走 deploy.bat；deploy.sh 只给 Linux/macOS 用。
rem
rem  部署步骤（与线上既有约定一致）：
rem    0. 预检   本地包存在 / 本地 md5 / 远端连通 / 磁盘余量 / 当前服务状态
rem    1. 备份   打包当前 /opt/Linuxbushu -> /opt/Linuxbushu.bak.<随机后缀>.tar.gz
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
rem
rem  ★★ 维护须知（2026-09-28 重写，踩过的两个坑，改之前先读）：
rem  【坑1】if / else 嵌套块里的**每一行都不能用裸小括号**：
rem         `(跳过)` `XX 解压失败(线上没在跑)` 这种写法里的 `)` 会被 cmd 当成
rem         「块结束」，把整个 if 块提前截断 → 后面每一行的 if/else 全部错位 →
rem         cmd 报 "was unexpected at this time." 然后**整个脚本什么都不做就退出**。
rem         块内必须写成 ^( ^) 转义。本脚本原来就是死在这里：深度累加后结尾还剩 3 个
rem         未闭合的 `(`，连后面的 :rsh / :rsh_read / :usage 子过程都被困在块里。
rem  【坑2】同一 if/else 块里**不能对同一个变量赋值并立刻用 !var! 取值**：
rem         `for ... do ( if not defined RH_OUT set "RH_OUT=%%i" )` —— 整个 for 块在
rem         执行前就被一次性展开，`if not defined RH_OUT` 永远看到的是块外旧值，
rem         于是循环里每一次都会赋值，RH_OUT 最终等于**最后一行输出**而不是第一行。
rem         rsh_read 要「取第一行」就绝不能放进 for 块，本脚本改用
rem         `set /p RH_OUT=<文件` 直接读首行（无需延迟展开，也没有块展开问题）。
rem  【坑3】`quit` 这类词不是 cmd 命令/标签，会被当成外部程序去找并报错 → 统一用 goto :eof。
rem  【坑4】★★ 输出里禁止写「感叹号方括号」那种形式：
rem         本脚本的 echo 输出会被调用方 PS 控制台解析，而 PowerShell 会把
rem         「方括号加感叹号」当成通配符字符集，吃掉方括号，
rem         右侧紧邻的字符被当成命令名去执行，报出一串莫名其妙的东西：
rem           '.js' is not recognized as an internal or external command
rem           'eploy.bat' is not recognized as an internal or external command
rem         看起来像路径 bug，其实跟路径无关。统一改用 WARN: 这类纯文字标记。
rem  ★★ 2026-10-07 补充：**连 rem 注释里也不要写半角方括号和半角圆括号** ——
rem     ① cmd 解析 if 块时会把块内注释一起纳入括号配对，注释里的裸圆括号会把块提前截断
rem        （本次就踩了：备份段一行注释末尾的圆括号导致整个脚本块错位）；
rem     ② 万一有人 cat / type 本文件再粘到终端，方括号又会被 PS 当通配符。
rem     本文件的 if 块内注释一律只用全角括号；块外注释举例里的半角括号仅供阅读，
rem     别把本文件 cat / type 之后整段粘到终端执行。
rem  【坑5】★★ if( ) 块内**不能**用 %VAR% 读「块内 call 子过程刚设好的变量」：
rem         cmd 在执行 if 块之前会把整个块一次性展开，%RH_RC% 在这一刻就被替换成
rem         **展开前的旧值** → `if not "%RH_RC%"=="0" goto fail` 永远拿到旧值、形同虚设。
rem         块内必须写 `!RH_RC!`（本脚本已 setlocal enabledelayedexpansion）。
rem         ★ 备份段(1/7)里的失败检查就在 if 块内，那里用的是 !RH_RC!，别改成 %RH_RC%；
rem           块外（如停服/启动/验证段）则统一用 %RH_RC%，两种都对但别混着看晕。
rem  【坑6】★ 本文件必须是 CRLF 行尾，且**只能用 ASCII 直引号**：
rem         改完脚本请确认 `grep -c $'\r' deploy.bat` 等于总行数，否则 cmd 解析标签会出错。
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
set "TMPERR=%TEMP%\deploy_err_%RANDOM%.txt"

rem ---------- 参数 ----------
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
echo   XX 未知参数: %~1  （用 --help 看用法）
exit /b 1

:parsed
if not exist "%SSH_RUN%" (
  echo   XX 找不到 %SSH_RUN%
  exit /b 1
)
rem ★ node 不在 PATH 时 :rsh_read 会静默失败（错误被重定向吞掉），
rem   表现出来就是「SSH 连不上」，极难排查 —— 这里提前拦一道。
node -v >nul 2>&1
if errorlevel 1 (
  echo   XX 找不到 node 命令, 请安装 Node.js 并把 node.exe 所在目录加入 PATH
  exit /b 1
)

echo ==^> 部署目标  %SSH_USER%@%SSH_HOST%:%REMOTE_DIR%
if "%DRY_RUN%"=="1" echo   WARN: DRY-RUN 模式: 只做预检, 不会改任何东西

rem =========================================================
rem 0. 预检
rem =========================================================
echo ==^> 0/7 预检

rem ★ 用 if not defined 而不是 if "%SSH_PASS%"==""：后者在展开阶段会把密码里的
rem   ^& ^| ^< 等字符当成命令分隔符/重定向 → 语法错乱；if defined 只看变量是否存在、
rem   完全不展开值，是最安全的判空写法。
rem ★ 这里的括号必须 ^( ^) 转义：它在 if( ) 块内，裸括号会把块提前截断（见文件头「坑1」）
rem ★★ 刻意不用 if 块：块内必须把圆括号转义成 ^( ^)，而 set /p 的提示文本是字面量、
rem   不解析转义 → 会把 ^ 原样显示给用户。改成 if + goto 拍平后括号就是普通字符。
if defined SSH_PASS goto pass_ready
set /p "SSH_PASS=SSH 密码 (%SSH_USER%@%SSH_HOST%, 直接回车用内置默认值): "
:pass_ready

if not exist "%PKG%" (
  echo   XX 部署包不存在: %PKG%
  exit /b 1
)
for %%A in ("%PKG%") do set "LOCAL_SIZE=%%~zA"

set "LOCAL_MD5="
for /f "skip=1 delims=" %%i in ('certutil -hashfile "%PKG%" MD5 2^>nul') do (
  if not defined LOCAL_MD5 set "LOCAL_MD5=%%i"
)
set "LOCAL_MD5=%LOCAL_MD5: =%"
if "%LOCAL_MD5%"=="" (
  echo   XX 算不出本地 md5（certutil 失败）
  exit /b 1
)

set /a LOCAL_MB=%LOCAL_SIZE%/1048576
echo   OK 部署包 %PKG% 约 %LOCAL_MB% MB  md5=%LOCAL_MD5%

call :rsh_read "echo ping"
if "%RH_OUT%"=="ping" goto ssh_ok
echo   XX SSH 连不上 %SSH_USER%@%SSH_HOST%
echo       ssh-run.js 原始输出如下, 请据此判断原因:
if exist "%TMPERR%" type "%TMPERR%"
if exist "%TMPOUT%" type "%TMPOUT%"
exit /b 1
:ssh_ok
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
set "HAS_DIR=%RH_OUT%"
if "%HAS_DIR%"=="yes" (
  call :rsh_read "cat %REMOTE_DIR%/logs/server.pid 2>/dev/null"
  set "OLD_PID=!RH_OUT!"
  call :rsh_read "ss -lntp 2>/dev/null | grep -c ':8080 ' || true"
  echo   OK 线上已有部署, pid=!OLD_PID!^(空=没在跑^), 8080 监听数=!RH_OUT!
) else (
  echo   WARN: 线上没有 %REMOTE_DIR% —— 这是全新部署^(没有旧配置可保留^)
)

if "%NO_RESTART%"=="1" echo   WARN: --no-restart: 只上传 + 校验, 到此为止
if "%DRY_RUN%"=="1" echo ==^> DRY-RUN 结束, 下面是完整计划:

rem ---------- 1. 备份 ----------
set "BAK_CONFIG="
set "BAK_TGZ="
call :mktimestamp

if "%NO_RESTART%"=="0" if "%HAS_DIR%"=="yes" (
  echo ==^> 1/7 备份线上现有部署
  if "%NO_BACKUP%"=="1" (
    echo   WARN: --no-backup: 跳过整包备份, 只留 server/config.yaml
    call :rsh "cp -a %REMOTE_DIR%/server/config.yaml %REMOTE_DIR%/server/config.yaml.bak.%TS%"
    if not "!RH_RC!"=="0" (
      echo   XX 备份 config.yaml 失败
      goto fail_clean
    )
  ) else (
    rem ★ 必须排除 logs/: 服务正在写 server.log, 不排除 tar 会报 file changed as we read it
    rem ★ 非空校验: tar 因告警返回 1 时也会留下文件, 空包不能当备份成功, 对齐 deploy.sh
    call :rsh "tar -czf '/opt/Linuxbushu.bak.%TS%.tar.gz' --exclude='Linuxbushu/logs' -C /opt Linuxbushu; rc=$?; if [ $rc -gt 1 ]; then exit $rc; fi; [ -s '/opt/Linuxbushu.bak.%TS%.tar.gz' ] && ls -la '/opt/Linuxbushu.bak.%TS%.tar.gz'"
    if not "!RH_RC!"=="0" (
      echo   XX 备份失败, 已中止^(线上未改动^)
      goto fail_clean
    )
    call :rsh "cp -a %REMOTE_DIR%/server/config.yaml %REMOTE_DIR%/server/config.yaml.bak.%TS%"
    if not "!RH_RC!"=="0" (
      echo   XX 备份 config.yaml 失败
      goto fail_clean
    )
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
  echo      dry-run: scp %PKG% -^> %SSH_USER%@%SSH_HOST%:%REMOTE_TGZ%
) else (
  node "%SSH_RUN%" put "%PKG%" "%REMOTE_TGZ%"
  set "PUT_RC=!errorlevel!"
  chcp 936 >nul
  if not "!PUT_RC!"=="0" (
    echo   XX 上传失败
    goto fail_clean
  )
  call :rsh_read "md5sum '%REMOTE_TGZ%' | awk '{print $1}'"
  set "REMOTE_MD5=!RH_OUT!"
rem ★ 不能用 stat -c %s：call 二次展开会把 %s 吃成 s（实测）。用 wc -c 输出纯数字。
call :rsh_read "wc -c < '%REMOTE_TGZ%'"
  set "REMOTE_SIZE=!RH_OUT!"
  if not "!REMOTE_MD5!"=="%LOCAL_MD5%" (
    echo   XX md5 不一致！ 本地=%LOCAL_MD5% 远端=!REMOTE_MD5! ^(线上未改动^)
    goto fail_clean
  )
  if not "!REMOTE_SIZE!"=="%LOCAL_SIZE%" (
    echo   XX 字节数不一致！ 本地=%LOCAL_SIZE% 远端=!REMOTE_SIZE! ^(线上未改动^)
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
if not "%RH_RC%"=="0" echo   WARN: stop.sh 返回非 0^(可能本来就没在跑^), 继续

rem ---------- 4. 解压 ----------
echo ==^> 4/7 解压 ^(排除模板 config.yaml^)
rem ★ 解压会把 tar 里的权限位盖回去(macOS 打包的 *.sh 常是 644), 解压完必须再补 chmod
call :rsh "tar -xzf '%REMOTE_TGZ%' --exclude='Linuxbushu/server/config.yaml' -C /opt && echo 'extract ok'"
if not "%RH_RC%"=="0" (
  echo   XX 解压失败 —— 线上现在没在跑, 用备份回滚: tar -xzf %BAK_TGZ% -C /opt
  goto fail_clean
)
call :rsh "chmod +x %REMOTE_DIR%/*.sh && echo 'chmod sh ok'"
if not "%RH_RC%"=="0" (
  echo   XX chmod *.sh 失败
  goto fail_clean
)
echo   OK 解压完成 + 补回 *.sh 执行位

rem ---------- 5. 恢复配置 ----------
echo ==^> 5/7 恢复真实配置
if not "%BAK_CONFIG%"=="" goto restore_from_backup
call :rsh_read "test -f '%CONFIG_SEED%' && echo yes || echo no"
if "%RH_OUT%"=="yes" goto restore_from_seed
echo   WARN: 找不到真实配置！ 包里的是模板^(占位密码 + 旧 web_dir^), 服务会起不来
goto after_restore

:restore_from_backup
call :rsh "cp -a '%BAK_CONFIG%' %REMOTE_DIR%/server/config.yaml && echo 'restored from backup'"
echo   OK 已用备份的真实配置覆盖模板
goto after_restore

:restore_from_seed
call :rsh "cp -a '%CONFIG_SEED%' %REMOTE_DIR%/server/config.yaml"
echo   OK 已用 %CONFIG_SEED% 覆盖模板

:after_restore
call :rsh "chown -R root:root %REMOTE_DIR%"
if not "%RH_RC%"=="0" echo   WARN: chown 失败^(macOS 打包的 uid 501^), 不影响启动

rem ---------- 6. 启动 ----------
echo ==^> 6/7 启动
call :rsh "chmod +x %REMOTE_DIR%/*.sh %REMOTE_DIR%/server/server %REMOTE_DIR%/server/dbinit %REMOTE_DIR%/server/ezfymigrate 2>/dev/null; cd %REMOTE_DIR% && ./start.sh"
if not "%RH_RC%"=="0" (
  echo   XX 启动脚本失败, 看日志: %REMOTE_DIR%/logs/server.log
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log"
  echo !RH_OUT!
  goto fail_clean
)

rem ---------- 7. 验证 ----------
echo ==^> 7/7 验证
ping -n 7 127.0.0.1 >nul 2>&1
call :rsh_read "ss -lntp 2>/dev/null | grep ':8080 ' >/dev/null && echo LISTEN_OK || echo LISTEN_FAIL"
if not "%RH_OUT%"=="LISTEN_OK" (
  echo   XX 8080 没在监听
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log"
  echo !RH_OUT!
  goto fail_clean
)
echo   OK 8080 已监听

rem ★ 不能用 curl -w '%{http_code}'：call 会对参数做二次展开，% 会被吃掉。
rem   改用 -sI 取状态行（形如 HTTP/1.1 200 OK），再 findstr 匹配 " 200 "。
call :rsh_read "curl -sI --max-time 8 http://127.0.0.1:8080/ | head -1"
echo !RH_OUT! | findstr /c:" 200 " >nul
if errorlevel 1 (
  echo   XX 首页不是 200 ^(!RH_OUT!^)
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log"
  echo !RH_OUT!
  goto fail_clean
)
echo   OK 首页 HTTP 200

rem 版本指纹: 未知路径会回落 index.html 返 200, 所以必须看响应体是不是 JSON
call :rsh_read "curl -s --max-time 8 http://127.0.0.1:8080/api/games/ezfy/view | head -c 60"
echo %RH_OUT% | findstr /c:"code" >nul
if errorlevel 1 goto verify_not_json
echo   OK 接口返回 JSON —— 新版本已生效
goto after_verify

:verify_not_json
echo   WARN: 接口返回的不是 JSON^(可能是 index.html 回落^) —— 确认下版本
echo %RH_OUT%

:after_verify
call :rsh_read "tail -n 8 %REMOTE_DIR%/logs/server.log"
echo %RH_OUT%

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

rem :mktimestamp —— 生成备份用的时间戳后缀，结果放 TS。
rem
rem ★★ 背景（都踩过，改之前先读）：
rem   ① `for /f "tokens=1-4 delims=/-. " %%a in ("%DATE% %TIME%")`
rem      —— 中文 Windows 的 %DATE% 是「2026/09/28 周一」，多出中文星期，
rem      分词结果随区域设置漂移，可能拿到 "周一" 这种垃圾。
rem   ② `wmic OS get LocalDateTime`
rem      —— **Win11 24H2 起 wmic 已被微软移除**（实测本机 build 26200 上不存在），
rem      用了只会静默失败。
rem   ③ 各种「剔除非数字字符」的字符串体操
rem      —— cmd 的延迟展开 + for 变量替换混用陷阱太多，写对了也很脆。
rem
rem ★ 最终方案：**只要一个「本次运行唯一」的后缀就够了，不追求好看的日期格式**。
rem   备份文件名的作用只是「不和之前的备份撞车」，用 %RANDOM% 完全够用，
rem   而且 100% 可靠、零依赖、不挑系统语言和版本。
rem   （%TIME% 的百分秒段在个别区域设置下可能带非数字字符，一并避开不用。）
rem   想让人看出时间也没关系 —— 备份完成后脚本会 echo 出备份包的完整路径和大小。
:mktimestamp
set "TS=%RANDOM%%RANDOM%"
goto :eof

rem :rsh "远端命令"  —— 有副作用的命令, DRY_RUN 时只打印
:rsh
if "%DRY_RUN%"=="1" (
  echo      dry-run: ssh %SSH_USER%@%SSH_HOST%: %~1
  set "RH_RC=0"
  goto :eof
)
node "%SSH_RUN%" exec "%~1"
set "RH_RC=%errorlevel%"
rem ★ node 在 Windows 上启动时会把控制台代码页强制改成 UTF-8，
rem   之后脚本 echo 的中文就会乱码 —— 每次调完 node 都要切回 936。
chcp 936 >nul
goto :eof

rem :rsh_read "远端命令" —— 只读命令, DRY_RUN 时也真执行^(预检需要真实结果^)
rem ★ 结果只取**第一行**放进 RH_OUT，退出码放进 RH_RC。
rem   这里必须用 set /p（见文件头「坑2」），不能用 for /f 循环包 if。
:rsh_read
if exist "%TMPOUT%" del /q "%TMPOUT%" >nul 2>&1
if exist "%TMPERR%" del /q "%TMPERR%" >nul 2>&1
node "%SSH_RUN%" exec "%~1" >"%TMPOUT%" 2>"%TMPERR%"
set "RH_RC=%errorlevel%"
chcp 936 >nul
set "RH_OUT="
if exist "%TMPOUT%" set /p "RH_OUT="<"%TMPOUT%"
goto :eof

rem :parse_pass —— 处理 -p/--pass <密码>。
rem   ★ 刻意用「标签跳跃」而不是 `if ... ( set "SSH_PASS=%~2" & shift )`：
rem     密码里可能含 `)` 或 `(`，块内的裸括号会把 if 块提前截断（见文件头「坑1」），
rem     而 `set "SSH_PASS=..."` 在标签处执行时整体在引号内，含括号也安全。
:parse_pass
set "SSH_PASS=%~2"
shift
shift
goto parse

:usage
echo deploy.bat —— 一键推送并部署「家园社区」到线上 Linux 服务器
echo.
echo 脚本位置: qqjiayuan\tools\deploy.bat
echo   本脚本按自身所在位置定位部署包, 从任意目录执行都行
echo   注意 git 仓库根是 codingce3gJy, 脚本在它下面的 qqjiayuan\tools\, 别在仓库根直接敲 .\tools\deploy.bat
echo.
echo   tools\deploy.bat                     默认用 ..\Linuxbushu.tar.gz 全量部署
echo   tools\deploy.bat -f D:\pkg.tar.gz    指定部署包
echo   tools\deploy.bat -h 1.2.3.4 -u root  指定主机/用户
echo   tools\deploy.bat -p 密码             指定 SSH 密码
echo   tools\deploy.bat --no-restart        只上传 + 校验，不停服不解压
echo   tools\deploy.bat --no-backup         跳过备份（省空间，不推荐）
echo   tools\deploy.bat --dry-run           只打印计划 + 预检，不改任何东西
echo   tools\deploy.bat --help              本帮助
echo.
echo 密码优先级: -p 参数 ^> 环境变量 SSH_PASS ^> 交互输入 ^> ssh-run.js 内置默认值
echo.
echo 各终端用法^(在项目根目录 qqjiayuan 下执行^):
echo   CMD:         tools\deploy.bat -p 密码 -h 39.105.151.141 --no-backup
echo   PowerShell:  .\tools\deploy.bat -p 密码 -h 39.105.151.141 --no-backup
echo   PowerShell:  $env:SSH_PASS='密码'; .\tools\deploy.bat -h 39.105.151.141 --no-backup
echo   Git Bash:    SSH_PASS='密码' ./tools/deploy.bat -h 39.105.151.141 --no-backup
echo.
echo 注意: SSH_PASS='密码' 前缀只在 Git Bash 里有效; PowerShell 会报 CommandNotFoundException
echo       CMD / PowerShell 请用 -p 参数, 或先执行 $env:SSH_PASS='密码'
echo.
echo 提示: 密码含 ^& ^| ^^ 等特殊字符时用双引号包起来, 例如 -p "a^&b"
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
