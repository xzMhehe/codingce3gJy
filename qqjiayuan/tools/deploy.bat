@echo off
chcp 936 >nul
setlocal enabledelayedexpansion
title ��԰���� - ��������

rem ============================================================================
rem  deploy.bat ���� һ�����Ͳ����𡸼�԰������������ Linux ��������Windows �棩
rem
rem  �� tools/deploy.sh �ȼۣ��߼��벽����ȫһ�£��� Win11 / Win2012 �á�
rem  ������node��tools/ssh-run.js ����� ssh/scp��
rem
rem  �÷����ڱ�Ŀ¼����Ŀ��Ŀ¼˫��/������ִ�ж��У���
rem    tools\deploy.bat                     Ĭ���� ..\Linuxbushu.tar.gz ȫ������
rem    tools\deploy.bat -f D:\pkg.tar.gz    ָ�������
rem    tools\deploy.bat --no-restart        ֻ�ϴ� + У�飬��ͣ������ѹ
rem    tools\deploy.bat --dry-run           ֻ��ӡ�ƻ� + Ԥ�죬�����κζ���
rem    tools\deploy.bat --no-backup         �������ݣ�ʡ�ռ䣬���Ƽ���
rem    tools\deploy.bat -h 1.2.3.4 -u root  ָ������/�û�
rem    tools\deploy.bat -p ����             ָ�� SSH ����
rem
rem  �������ȼ���-p ���� > �������� SSH_PASS > �������� > ssh-run.js ����Ĭ��ֵ
rem
rem  ���ն�ִ�з�ʽ���� ��������Ŀ��Ŀ¼ qqjiayuan �£����� .bat ������ .sh�����£���
rem    CMD:         tools\deploy.bat -p ���� -h 39.105.151.141 --no-backup
rem    PowerShell:  .\tools\deploy.bat -p ���� -h 39.105.151.141 --no-backup
rem    Git Bash:    SSH_PASS='����' ./tools/deploy.bat -h 39.105.151.141 --no-backup
rem
rem  ��� `SSH_PASS='����' ����` �� **bash ר���﷨**���� PowerShell ���ֱ�ӱ�
rem     ���޷���"SSH_PASS=..."��ʶ��Ϊ cmdlet���������ű��ļ�������г�������ơ���
rem     PowerShell ����·���� �� -p �������� `$env:SSH_PASS='����'; .\tools\deploy.bat ...`
rem     ���ֺŷָ�����������������ǰ�浥���裬����д�� `SSH_PASS=x ����` ����ǰ׺����
rem
rem  ��� Ϊʲô Windows ��Ҫ�� .bat ������ .sh��2026-10-07 ��������
rem    �ֿ��� deploy.sh �� deploy.bat �� index �ﶼ�� LF�������� core.autocrlf=true��
rem    checkout ���������������ļ������ CRLF��.bat ���� CRLF����ȷ����
rem    �� .sh һ���� CRLF���� Git Bash �»ᱨ `$'\r': command not found` ֱ���ܲ��ˡ�
rem    ���� Windows ��ͳһ�� deploy.bat��deploy.sh ֻ�� Linux/macOS �á�
rem
rem  �����裨�����ϼ���Լ��һ�£���
rem    0. Ԥ��   ���ذ����� / ���� md5 / Զ����ͨ / �������� / ��ǰ����״̬
rem    1. ����   �����ǰ /opt/Linuxbushu -> /opt/Linuxbushu.bak.<�����׺>.tar.gz
rem    2. �ϴ�   scp �� /opt/Linuxbushu.tar.gz��md5 ˫��У�飨��һ��������ֹ��
rem    3. ͣ��   chmod +x *.sh && ./stop.sh��������ͣ�ٽ�ѹ������ Text file busy��
rem    4. ��ѹ   tar -xzf ... --exclude=server/config.yaml��������ģ�����ã�
rem    5. �ָ�   �ѱ��ݵ���ʵ config.yaml �Ż� server/
rem    6. ����   chmod +x server/* && ./start.sh
rem    7. ��֤   8080 �ڼ��� + curl 200 + /api ���� JSON������ index.html ���䣩
rem
rem  �ع����ڷ��������ֶ�ִ�У���
rem    chmod +x /opt/Linuxbushu/*.sh; /opt/Linuxbushu/stop.sh
rem    rm -rf /opt/Linuxbushu
rem    tar -xzf /opt/Linuxbushu.bak.<ʱ���>.tar.gz -C /opt
rem    chmod +x /opt/Linuxbushu/server/*; /opt/Linuxbushu/start.sh
rem
rem  ��� ά����֪��2026-09-28 ��д���ȹ��������ӣ���֮ǰ�ȶ�����
rem  ����1��if / else Ƕ�׿����**ÿһ�ж���������С����**��
rem         `(����)` `XX ��ѹʧ��(����û����)` ����д����� `)` �ᱻ cmd ����
rem         ����������������� if ����ǰ�ض� �� ����ÿһ�е� if/else ȫ����λ ��
rem         cmd �� "was unexpected at this time." Ȼ��**�����ű�ʲô���������˳�**��
rem         ���ڱ���д�� ^( ^) ת�塣���ű�ԭ�����������������ۼӺ��β��ʣ 3 ��
rem         δ�պϵ� `(`��������� :rsh / :rsh_read / :usage �ӹ��̶������ڿ��
rem  ����2��ͬһ if/else ����**���ܶ�ͬһ��������ֵ�������� !var! ȡֵ**��
rem         `for ... do ( if not defined RH_OUT set "RH_OUT=%%i" )` ���� ���� for ����
rem         ִ��ǰ�ͱ�һ����չ����`if not defined RH_OUT` ��Զ�������ǿ����ֵ��
rem         ����ѭ����ÿһ�ζ��ḳֵ��RH_OUT ���յ���**���һ�����**�����ǵ�һ�С�
rem         rsh_read Ҫ��ȡ��һ�С��;����ܷŽ� for �飬���ű�����
rem         `set /p RH_OUT=<�ļ�` ֱ�Ӷ����У������ӳ�չ����Ҳû�п�չ�����⣩��
rem  ����3��`quit` ����ʲ��� cmd ����/��ǩ���ᱻ�����ⲿ����ȥ�Ҳ����� �� ͳһ�� goto :eof��
rem  ����4����� ������ֹд����̾�ŷ����š�������ʽ��
rem         ���ű��� echo ����ᱻ���÷� PS ����̨�������� PowerShell ���
rem         �������żӸ�̾�š�����ͨ����ַ������Ե������ţ�
rem         �Ҳ���ڵ��ַ�������������ȥִ�У�����һ��Ī������Ķ�����
rem           '.js' is not recognized as an internal or external command
rem           'eploy.bat' is not recognized as an internal or external command
rem         ��������·�� bug����ʵ��·���޹ء�ͳһ���� WARN: ���ി���ֱ�ǡ�
rem  ��� 2026-10-07 ���䣺**�� rem ע����Ҳ��Ҫд��Ƿ����źͰ��Բ����** ����
rem     �� cmd ���� if ��ʱ��ѿ���ע��һ������������ԣ�ע�������Բ���Ż�ѿ���ǰ�ض�
rem        �����ξͲ��ˣ����ݶ�һ��ע��ĩβ��Բ���ŵ��������ű����λ����
rem     �� ��һ���� cat / type ���ļ���ճ���նˣ��������ֻᱻ PS ��ͨ�����
rem     ���ļ��� if ����ע��һ��ֻ��ȫ�����ţ�����ע�;�����İ�����Ž����Ķ���
rem     ��ѱ��ļ� cat / type ֮������ճ���ն�ִ�С�
rem  ����5����� if( ) ����**����**�� %VAR% �������� call �ӹ��̸���õı�������
rem         cmd ��ִ�� if ��֮ǰ���������һ����չ����%RH_RC% ����һ�̾ͱ��滻��
rem         **չ��ǰ�ľ�ֵ** �� `if not "%RH_RC%"=="0" goto fail` ��Զ�õ���ֵ����ͬ���衣
rem         ���ڱ���д `!RH_RC!`�����ű��� setlocal enabledelayedexpansion����
rem         �� ���ݶ�(1/7)���ʧ�ܼ����� if ���ڣ������õ��� !RH_RC!����ĳ� %RH_RC%��
rem           ���⣨��ͣ��/����/��֤�Σ���ͳһ�� %RH_RC%�����ֶ��Ե�����ſ��Ρ�
rem  ����6���� ���ļ������� CRLF ��β����**ֻ���� ASCII ֱ����**��
rem         ����ű���ȷ�� `grep -c $'\r' deploy.bat` ���������������� cmd ������ǩ�������
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

rem ---------- ���� ----------
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
echo   XX δ֪����: %~1  ���� --help ���÷���
exit /b 1

:parsed
if not exist "%SSH_RUN%" (
  echo   XX �Ҳ��� %SSH_RUN%
  exit /b 1
)
rem �� node ���� PATH ʱ :rsh_read �ᾲĬʧ�ܣ������ض����̵�����
rem   ���ֳ������ǡ�SSH �����ϡ��������Ų� ���� ������ǰ��һ����
node -v >nul 2>&1
if errorlevel 1 (
  echo   XX �Ҳ��� node ����, �밲װ Node.js ���� node.exe ����Ŀ¼���� PATH
  exit /b 1
)

echo ==^> ����Ŀ��  %SSH_USER%@%SSH_HOST%:%REMOTE_DIR%
if "%DRY_RUN%"=="1" echo   WARN: DRY-RUN ģʽ: ֻ��Ԥ��, ������κζ���

rem =========================================================
rem 0. Ԥ��
rem =========================================================
echo ==^> 0/7 Ԥ��

rem �� �� if not defined ������ if "%SSH_PASS%"==""��������չ���׶λ���������
rem   ^& ^| ^< ���ַ���������ָ���/�ض��� �� �﷨���ң�if defined ֻ�������Ƿ���ڡ�
rem   ��ȫ��չ��ֵ�����ȫ���п�д����
rem �� ��������ű��� ^( ^) ת�壺���� if( ) ���ڣ������Ż�ѿ���ǰ�ضϣ����ļ�ͷ����1����
rem ��� ���ⲻ�� if �飺���ڱ����Բ����ת��� ^( ^)���� set /p ����ʾ�ı�����������
rem   ������ת�� �� ��� ^ ԭ����ʾ���û����ĳ� if + goto ��ƽ�����ž�����ͨ�ַ���
if defined SSH_PASS goto pass_ready
set /p "SSH_PASS=SSH ���� (%SSH_USER%@%SSH_HOST%, ֱ�ӻس�������Ĭ��ֵ): "
:pass_ready

if not exist "%PKG%" (
  echo   XX �����������: %PKG%
  exit /b 1
)
for %%A in ("%PKG%") do set "LOCAL_SIZE=%%~zA"

set "LOCAL_MD5="
for /f "skip=1 delims=" %%i in ('certutil -hashfile "%PKG%" MD5 2^>nul') do (
  if not defined LOCAL_MD5 set "LOCAL_MD5=%%i"
)
set "LOCAL_MD5=%LOCAL_MD5: =%"
if "%LOCAL_MD5%"=="" (
  echo   XX �㲻������ md5��certutil ʧ�ܣ�
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
echo   OK ����� %PKG% Լ %LOCAL_MB% MB  md5=%LOCAL_MD5%

call :rsh_read "echo ping"
if "%RH_OUT%"=="ping" goto ssh_ok
echo   XX SSH ������ %SSH_USER%@%SSH_HOST%
echo       ssh-run.js ԭʼ�������, ��ݴ��ж�ԭ��:
if exist "%TMPERR%" type "%TMPERR%"
if exist "%TMPOUT%" type "%TMPOUT%"
exit /b 1
:ssh_ok
echo   OK SSH ��ͨ

call :rsh_read "df -P /opt | tail -1 | awk '{print $4}'"
set "DISK_AVAIL=%RH_OUT%"
if "%DISK_AVAIL%"=="" (
  echo   XX ������ /opt ��������
  exit /b 1
)
set /a AVAIL_MB=%DISK_AVAIL%/1024
set /a NEED_MB=%LOCAL_SIZE%/1048576+120
if %AVAIL_MB% LSS %NEED_MB% (
  echo   XX /opt ʣ�� %AVAIL_MB%MB, ����^(��Լ %NEED_MB%MB^)
  exit /b 1
)
echo   OK /opt ʣ�� %AVAIL_MB%MB ^(��Լ %NEED_MB%MB^)

call :rsh_read "test -d '%REMOTE_DIR%' && echo yes || echo no"
set "HAS_DIR=%RH_OUT%"
if "%HAS_DIR%"=="yes" (
  call :rsh_read "cat %REMOTE_DIR%/logs/server.pid 2>/dev/null"
  set "OLD_PID=!RH_OUT!"
  call :rsh_read "ss -lntp 2>/dev/null | grep -c ':8080 ' || true"
  echo   OK �������в���, pid=!OLD_PID!^(��=û����^), 8080 ������=!RH_OUT!
) else (
  echo   WARN: ����û�� %REMOTE_DIR% ���� ����ȫ�²���^(û�о����ÿɱ���^)
)

if "%NO_RESTART%"=="1" echo   WARN: --no-restart: ֻ�ϴ� + У��, ����Ϊֹ
if "%DRY_RUN%"=="1" echo ==^> DRY-RUN ����, �����������ƻ�:

rem ---------- 1. ���� ----------
set "BAK_CONFIG="
set "BAK_TGZ="
call :mktimestamp

if "%NO_RESTART%"=="0" if "%HAS_DIR%"=="yes" (
  echo ==^> 1/7 �����������в���
  if "%NO_BACKUP%"=="1" (
    echo   WARN: --no-backup: ������������, ֻ�� server/config.yaml
    call :rsh "cp -a %REMOTE_DIR%/server/config.yaml %REMOTE_DIR%/server/config.yaml.bak.%TS%"
    if not "!RH_RC!"=="0" (
      echo   XX ���� config.yaml ʧ��
      goto fail_clean
    )
  ) else (
    rem �� �����ų� logs/: ��������д server.log, ���ų� tar �ᱨ file changed as we read it
    rem �� �ǿ�У��: tar ��澯���� 1 ʱҲ�������ļ�, �հ����ܵ����ݳɹ�, ���� deploy.sh
    call :rsh "tar -czf '/opt/Linuxbushu.bak.%TS%.tar.gz' --exclude='Linuxbushu/logs' -C /opt Linuxbushu; rc=$?; if [ $rc -gt 1 ]; then exit $rc; fi; [ -s '/opt/Linuxbushu.bak.%TS%.tar.gz' ] && ls -la '/opt/Linuxbushu.bak.%TS%.tar.gz'"
    if not "!RH_RC!"=="0" (
      echo   XX ����ʧ��, ����ֹ^(����δ�Ķ�^)
      goto fail_clean
    )
    call :rsh "cp -a %REMOTE_DIR%/server/config.yaml %REMOTE_DIR%/server/config.yaml.bak.%TS%"
    if not "!RH_RC!"=="0" (
      echo   XX ���� config.yaml ʧ��
      goto fail_clean
    )
    set "BAK_TGZ=/opt/Linuxbushu.bak.%TS%.tar.gz"
    echo   OK ����: !BAK_TGZ! ^(���� logs/^)
  )
  set "BAK_CONFIG=%REMOTE_DIR%/server/config.yaml.bak.%TS%"
) else (
  echo ==^> 1/7 ����  ^(����^)
)

rem ---------- 2. �ϴ� + У�� ----------
echo ==^> 2/7 �ϴ������
if "%DRY_RUN%"=="1" (
  echo      dry-run: scp %PKG% -^> %SSH_USER%@%SSH_HOST%:%REMOTE_TGZ%
) else (
  node "%SSH_RUN%" put "%PKG%" "%REMOTE_TGZ%"
  set "PUT_RC=!errorlevel!"
  chcp 936 >nul
  if not "!PUT_RC!"=="0" (
    echo   XX �ϴ�ʧ��
    goto fail_clean
  )
  call :rsh_read "md5sum '%REMOTE_TGZ%' | awk '{print $1}'"
  set "REMOTE_MD5=!RH_OUT!"
rem �� ������ stat -c %s��call ����չ����� %s �Գ� s��ʵ�⣩���� wc -c ��������֡�
call :rsh_read "wc -c < '%REMOTE_TGZ%'"
  set "REMOTE_SIZE=!RH_OUT!"
  if not "!REMOTE_MD5!"=="%LOCAL_MD5%" (
    echo   XX md5 ��һ�£� ����=%LOCAL_MD5% Զ��=!REMOTE_MD5! ^(����δ�Ķ�^)
    goto fail_clean
  )
  if not "!REMOTE_SIZE!"=="%LOCAL_SIZE%" (
    echo   XX �ֽ�����һ�£� ����=%LOCAL_SIZE% Զ��=!REMOTE_SIZE! ^(����δ�Ķ�^)
    goto fail_clean
  )
  echo   OK md5 + �ֽ���˫��У��һ�� ^(!REMOTE_MD5!^)
)

if "%NO_RESTART%"=="1" (
  echo ==^> ���  --no-restart ģʽ: ���Ѿ�λ, ���Ϸ���δ��Ӱ��
  echo     Ҫ��ɲ���, ����һ��^(ȥ�� --no-restart^): tools\deploy.bat
  goto done
)

rem ---------- 3. ͣ�� ----------
echo ==^> 3/7 ͣ��
rem �� �״β���� *.sh ����û��ִ��λ, �� chmod �� Permission denied ȴ��Ϊͣ�ɹ���
call :rsh "chmod +x %REMOTE_DIR%/*.sh 2>/dev/null; %REMOTE_DIR%/stop.sh; sleep 1; ss -lntp 2>/dev/null | grep ':8080 ' || echo '8080 ���ͷ�'"
if not "%RH_RC%"=="0" echo   WARN: stop.sh ���ط� 0^(���ܱ�����û����^), ����

rem ---------- 4. ��ѹ ----------
echo ==^> 4/7 ��ѹ ^(�ų�ģ�� config.yaml^)
rem �� ��ѹ��� tar ���Ȩ��λ�ǻ�ȥ(macOS ����� *.sh ���� 644), ��ѹ������ٲ� chmod
call :rsh "tar -xzf '%REMOTE_TGZ%' --exclude='Linuxbushu/server/config.yaml' -C /opt && echo 'extract ok'"
if not "%RH_RC%"=="0" (
  echo   XX ��ѹʧ�� ���� ��������û����, �ñ��ݻع�: tar -xzf %BAK_TGZ% -C /opt
  goto fail_clean
)
call :rsh "chmod +x %REMOTE_DIR%/*.sh && echo 'chmod sh ok'"
if not "%RH_RC%"=="0" (
  echo   XX chmod *.sh ʧ��
  goto fail_clean
)
echo   OK ��ѹ��� + ���� *.sh ִ��λ

rem ---------- 5. �ָ����� ----------
echo ==^> 5/7 �ָ���ʵ����
if not "%BAK_CONFIG%"=="" goto restore_from_backup
call :rsh_read "test -f '%CONFIG_SEED%' && echo yes || echo no"
if "%RH_OUT%"=="yes" goto restore_from_seed
echo   WARN: �Ҳ�����ʵ���ã� �������ģ��^(ռλ���� + �� web_dir^), ���������
goto after_restore

:restore_from_backup
call :rsh "cp -a '%BAK_CONFIG%' %REMOTE_DIR%/server/config.yaml && echo 'restored from backup'"
echo   OK ���ñ��ݵ���ʵ���ø���ģ��
goto after_restore

:restore_from_seed
call :rsh "cp -a '%CONFIG_SEED%' %REMOTE_DIR%/server/config.yaml"
echo   OK ���� %CONFIG_SEED% ����ģ��

:after_restore
call :rsh "chown -R root:root %REMOTE_DIR%"
if not "%RH_RC%"=="0" echo   WARN: chown ʧ��^(macOS ����� uid 501^), ��Ӱ������

rem ---------- 6. ���� ----------
echo ==^> 6/7 ����
call :rsh "chmod +x %REMOTE_DIR%/*.sh %REMOTE_DIR%/server/server %REMOTE_DIR%/server/dbinit %REMOTE_DIR%/server/ezfymigrate 2>/dev/null; cd %REMOTE_DIR% && ./start.sh"
if not "%RH_RC%"=="0" (
  echo   XX �����ű�ʧ��, ����־: %REMOTE_DIR%/logs/server.log
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log | iconv -f UTF-8 -t GBK -c 2>/dev/null"
  if exist "%TMPOUT%" type "%TMPOUT%"
  goto fail_clean
)

rem ---------- 7. ��֤ ----------
echo ==^> 7/7 ��֤
ping -n 7 127.0.0.1 >nul 2>&1
call :rsh_read "ss -lntp 2>/dev/null | grep ':8080 ' >/dev/null && echo LISTEN_OK || echo LISTEN_FAIL"
if not "%RH_OUT%"=="LISTEN_OK" (
  echo   XX 8080 û�ڼ���
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log | iconv -f UTF-8 -t GBK -c 2>/dev/null"
  if exist "%TMPOUT%" type "%TMPOUT%"
  goto fail_clean
)
echo   OK 8080 �Ѽ���

rem �� ������ curl -w '%{http_code}'��call ��Բ���������չ����% �ᱻ�Ե���
rem   ���� -sI ȡ״̬�У����� HTTP/1.1 200 OK������ findstr ƥ�� " 200 "��
call :rsh_read "curl -sI --max-time 8 http://127.0.0.1:8080/ | head -1"
echo !RH_OUT! | findstr /c:" 200 " >nul
if errorlevel 1 (
  echo   XX ��ҳ���� 200 ^(!RH_OUT!^)
  call :rsh_read "tail -n 25 %REMOTE_DIR%/logs/server.log | iconv -f UTF-8 -t GBK -c 2>/dev/null"
  if exist "%TMPOUT%" type "%TMPOUT%"
  goto fail_clean
)
echo   OK ��ҳ HTTP 200

rem �汾ָ��: δ֪·������� index.html �� 200, ���Ա��뿴��Ӧ���ǲ��� JSON
call :rsh_read "curl -s --max-time 8 http://127.0.0.1:8080/api/games/ezfy/view | head -c 400 | iconv -f UTF-8 -t GBK -c 2>/dev/null"
findstr /c:"code" "%TMPOUT%" >nul
if errorlevel 1 goto verify_not_json
echo   OK �ӿڷ��� JSON ���� �°汾����Ч
goto after_verify

:verify_not_json
echo   WARN: �ӿڷ��صĲ��� JSON^(������ index.html ����^) ���� ȷ���°汾
echo %RH_OUT%

:after_verify
call :rsh_read "tail -n 8 %REMOTE_DIR%/logs/server.log | iconv -f UTF-8 -t GBK -c 2>/dev/null"
if exist "%TMPOUT%" type "%TMPOUT%"

echo.
echo �������  http://%SSH_HOST%:8080
if not "%BAK_TGZ%"=="" (
  echo   ����: %BAK_TGZ%
  echo   �ع�: tar -xzf %BAK_TGZ% -C /opt
) else (
  echo   ����: ��
)
goto done

rem =========================================================
rem �ӹ���
rem =========================================================

rem :mktimestamp ���� ���ɱ����õ�ʱ�����׺������� TS��
rem
rem ��� ���������ȹ�����֮ǰ�ȶ�����
rem   �� `for /f "tokens=1-4 delims=/-. " %%a in ("%DATE% %TIME%")`
rem      ���� ���� Windows �� %DATE% �ǡ�2026/09/28 ��һ��������������ڣ�
rem      �ִʽ������������Ư�ƣ������õ� "��һ" ����������
rem   �� `wmic OS get LocalDateTime`
rem      ���� **Win11 24H2 �� wmic �ѱ�΢���Ƴ�**��ʵ�Ȿ�� build 26200 �ϲ����ڣ���
rem      ����ֻ�ᾲĬʧ�ܡ�
rem   �� ���֡��޳��������ַ������ַ������
rem      ���� cmd ���ӳ�չ�� + for �����滻��������̫�࣬д����Ҳ�ܴࡣ
rem
rem �� ���շ�����**ֻҪһ������������Ψһ���ĺ�׺�͹��ˣ���׷��ÿ������ڸ�ʽ**��
rem   �����ļ���������ֻ�ǡ�����֮ǰ�ı���ײ�������� %RANDOM% ��ȫ���ã�
rem   ���� 100% �ɿ���������������ϵͳ���ԺͰ汾��
rem   ��%TIME% �İٷ�����ڸ������������¿��ܴ��������ַ���һ���ܿ����á���
rem   �����˿���ʱ��Ҳû��ϵ ���� ������ɺ�ű��� echo �����ݰ�������·���ʹ�С��
:mktimestamp
set "TS=%RANDOM%%RANDOM%"
goto :eof

rem :rsh "Զ������"  ���� �и����õ�����, DRY_RUN ʱֻ��ӡ
:rsh
if "%DRY_RUN%"=="1" (
  echo      dry-run: ssh %SSH_USER%@%SSH_HOST%: "%~1"
  set "RH_RC=0"
  goto :eof
)
node "%SSH_RUN%" exec "%~1"
set "RH_RC=%errorlevel%"
rem �� node �� Windows ������ʱ��ѿ���̨����ҳǿ�Ƹĳ� UTF-8��
rem   ֮��ű� echo �����ľͻ����� ���� ÿ�ε��� node ��Ҫ�л� 936��
chcp 936 >nul
goto :eof

rem :rsh_read "Զ������" ���� ֻ������, DRY_RUN ʱҲ��ִ��^(Ԥ����Ҫ��ʵ���^)
rem �� ���ֻȡ**��һ��**�Ž� RH_OUT���˳���Ž� RH_RC��
rem   ��������� set /p�����ļ�ͷ����2������������ for /f ѭ���� if��
:rsh_read
if exist "%TMPOUT%" del /q "%TMPOUT%" >nul 2>&1
if exist "%TMPERR%" del /q "%TMPERR%" >nul 2>&1
node "%SSH_RUN%" exec "%~1" >"%TMPOUT%" 2>"%TMPERR%"
set "RH_RC=%errorlevel%"
chcp 936 >nul
set "RH_OUT="
if exist "%TMPOUT%" set /p "RH_OUT="<"%TMPOUT%"
goto :eof

rem :parse_pass ���� ���� -p/--pass <����>��
rem   �� �����á���ǩ��Ծ�������� `if ... ( set "SSH_PASS=%~2" & shift )`��
rem     ��������ܺ� `)` �� `(`�����ڵ������Ż�� if ����ǰ�ضϣ����ļ�ͷ����1������
rem     �� `set "SSH_PASS=..."` �ڱ�ǩ��ִ��ʱ�����������ڣ�������Ҳ��ȫ��
:parse_pass
set "SSH_PASS=%~2"
shift
shift
goto parse

:usage
echo deploy.bat ���� һ�����Ͳ����𡸼�԰������������ Linux ������
echo.
echo �ű�λ��: qqjiayuan\tools\deploy.bat
echo   ���ű�����������λ�ö�λ�����, ������Ŀ¼ִ�ж���
echo   ע�� git �ֿ���� codingce3gJy, �ű���������� qqjiayuan\tools\, ���ڲֿ��ֱ���� .\tools\deploy.bat
echo.
echo   tools\deploy.bat                     Ĭ���� ..\Linuxbushu.tar.gz ȫ������
echo   tools\deploy.bat -f D:\pkg.tar.gz    ָ�������
echo   tools\deploy.bat -h 1.2.3.4 -u root  ָ������/�û�
echo   tools\deploy.bat -p ����             ָ�� SSH ����
echo   tools\deploy.bat --no-restart        ֻ�ϴ� + У�飬��ͣ������ѹ
echo   tools\deploy.bat --no-backup         �������ݣ�ʡ�ռ䣬���Ƽ���
echo   tools\deploy.bat --dry-run           ֻ��ӡ�ƻ� + Ԥ�죬�����κζ���
echo   tools\deploy.bat --help              ������
echo.
echo �������ȼ�: -p ���� ^> �������� SSH_PASS ^> �������� ^> ssh-run.js ����Ĭ��ֵ
echo.
echo ���ն��÷�^(����Ŀ��Ŀ¼ qqjiayuan ��ִ��^):
echo   CMD:         tools\deploy.bat -p ���� -h 39.105.151.141 --no-backup
echo   PowerShell:  .\tools\deploy.bat -p ���� -h 39.105.151.141 --no-backup
echo   PowerShell:  $env:SSH_PASS='����'; .\tools\deploy.bat -h 39.105.151.141 --no-backup
echo   Git Bash:    SSH_PASS='����' ./tools/deploy.bat -h 39.105.151.141 --no-backup
echo.
echo ע��: SSH_PASS='����' ǰ׺ֻ�� Git Bash ����Ч; PowerShell �ᱨ CommandNotFoundException
echo       CMD / PowerShell ���� -p ����, ����ִ�� $env:SSH_PASS='����'
echo.
echo ��ʾ: ���뺬 ^& ^| ^^ �������ַ�ʱ��˫���Ű�����, ���� -p "a^&b"
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
