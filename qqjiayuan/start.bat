@echo off
chcp 65001 >nul
title 家园社区
cd /d "%~dp0server"

echo ==========================================
echo   正在启动 家园社区...
echo   首次运行会自动建表并写入种子数据
echo   启动后访问 http://127.0.0.1:8080
echo   管理员：10000 / admin123
echo ==========================================
echo.

if not exist ".web_dist_marker" (
    if not exist "..\web\dist\index.html" (
        echo [提示] 尚未构建前端，正在执行 npm install ^& npm run build ...
        cd /d "%~dp0web"
        call npm install --registry=https://registry.npmmirror.com
        call npm run build
        cd /d "%~dp0server"
    )
    echo ok > web_dist_marker
)

go build -o server.exe . || (echo 后端编译失败 & pause & exit /b 1)
server.exe
pause
