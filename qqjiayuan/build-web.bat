@echo off
REM =====================================================================
REM  QQ JiaYuan - BUILD BOTH FRONTENDS (production)
REM
REM  Produces web\dist and admin-web\dist, which the Go server hosts.
REM  Needs Node + npm on PATH. Missing node_modules triggers npm install.
REM
REM  NOTE: run this on a normal machine. Do NOT expect this to work
REM  inside a restricted/sandboxed shell - the build deletes and
REM  recreates the dist folders, which some sandboxes block.
REM
REM  ASCII-only + CRLF on purpose (see start-dev-win2012.bat).
REM =====================================================================

if /i not "%~1"=="__run__" (
  cmd /k call "%~f0" __run__
  exit /b
)

setlocal
title QQ JiaYuan - Build Frontend
cd /d "%~dp0"

echo ============================================================
echo   QQ JiaYuan  -  BUILD FRONTEND
echo ============================================================
echo.

where node >nul 2>&1
if errorlevel 1 (echo   [ERROR] node not found in PATH. & goto :FAIL)
where npm >nul 2>&1
if errorlevel 1 (echo   [ERROR] npm not found in PATH. & goto :FAIL)
for /f "delims=" %%v in ('node -v') do echo   node %%v
for /f "delims=" %%v in ('npm -v')  do echo   npm  %%v
echo.

set "REG=https://registry.npmmirror.com"

call :BUILD web
if errorlevel 1 goto :FAIL

call :BUILD admin-web
if errorlevel 1 goto :FAIL

echo ============================================================
echo   Done. Output:
echo     %~dp0web\dist
echo     %~dp0admin-web\dist
echo.
echo   Next: copy both dist folders to the server, then start
echo   server-2012r2.exe. Do not forget web\dist\static.
echo ============================================================
goto :DONE

REM ---------------------------------------------------------------------
:BUILD
echo [build] %~1
if not exist "%~1\node_modules" (
  echo   node_modules missing - npm install, first run is slow
  pushd "%~1"
  call npm install --registry=%REG%
  if errorlevel 1 (echo   [ERROR] npm install failed in %~1 & popd & exit /b 1)
  popd
)
pushd "%~1"
call npm run build
if errorlevel 1 (echo   [ERROR] npm run build failed in %~1 & popd & exit /b 1)
popd
echo   ok
echo.
exit /b 0

REM ---------------------------------------------------------------------
:FAIL
echo.
echo ============================================================
echo   BUILD FAILED - see the [ERROR] line above.
echo   This window is kept open on purpose so you can read it.
echo ============================================================
echo.

:DONE
echo   Press any key to close this window.
pause
endlocal
exit /b 0
