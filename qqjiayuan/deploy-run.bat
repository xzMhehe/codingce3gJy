@echo off
chcp 936 >nul
title 一键部署 - 家园社区

rem ============================================================
rem  deploy-run.bat —— 双击即可部署（给不想敲命令行的场景）
rem
rem  位置：qqjiayuan\deploy-run.bat
rem  它只是 tools\deploy.bat 的一层薄封装：
rem    · 固定主机 39.105.151.141，参数可追加覆盖
rem    · 结束时 pause，双击运行也不会闪退看不到结果
rem    · 命令行也能用，例如 deploy-run.bat --no-backup
rem
rem  ★ 密码用 tools\ssh-run.js 的内置默认值，无需输入。
rem  ★ 维护须知见 tools\deploy.bat 的文件头；本文件刻意写得极简，
rem    避免再引入括号/转义之类的解析坑。
rem ============================================================

if not exist "%~dp0tools\deploy.bat" (
  echo   XX 找不到 tools\deploy.bat
  echo      请确认本文件位于 qqjiayuan 目录下
  goto end
)

echo 部署目标 39.105.151.141
echo.

call "%~dp0tools\deploy.bat" -h 39.105.151.141 %*
set "RC=%errorlevel%"

echo.
if "%RC%"=="0" echo 结果: 部署完成
if not "%RC%"=="0" echo 结果: 部署失败, 退出码 %RC%

:end
echo.
echo 按任意键关闭窗口 ...
pause >nul
exit /b
