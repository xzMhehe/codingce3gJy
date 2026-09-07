@echo off
REM ============================================
REM  家园社区 - 开发模式一键启动(热更新)
REM  后端 air 热重载 + 用户端/管理端 HMR
REM  端口: 后端8080 用户端8000 管理端8001
REM ============================================
cd /d %~dp0

echo [1/3] 启动后端 (air 热重载) ...
start "家园-后端(air)" cmd /k "cd /d %~dp0server && air"

echo [2/3] 启动用户端 (HMR :8000) ...
start "家园-用户端" cmd /k "cd /d %~dp0web && npm run serve"

echo [3/3] 启动管理端 (HMR :8001) ...
start "家园-管理端" cmd /k "cd /d %~dp0admin-web && npm run serve"

echo.
echo 全部已启动:
echo   后端  http://127.0.0.1:8080      (air 监听 .go 自动重启)
echo   用户端 http://localhost:8000     (Vue HMR 热更新)
echo   管理端 http://localhost:8001/admin-ui/
echo.
echo 关闭窗口即停止对应服务。改 Go/Vue 代码自动生效,无需手动编译。
pause
