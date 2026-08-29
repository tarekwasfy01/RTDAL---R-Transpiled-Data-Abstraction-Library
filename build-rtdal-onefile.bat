@echo off
setlocal
cd /d "%~dp0"

set "GOEXE=go"
if exist "C:\Program Files\Go\bin\go.exe" (
  set "GOEXE=C:\Program Files\Go\bin\go.exe"
  set "PATH=C:\Program Files\Go\bin;%PATH%"
)
"%GOEXE%" version >nul 2>nul || (
  echo ERROR: Go was not found.
  exit /b 1
)

if not exist .cache\go-build mkdir .cache\go-build
if not exist .cache\go-mod mkdir .cache\go-mod
if not exist dist mkdir dist

set "GOCACHE=%CD%\.cache\go-build"
set "GOMODCACHE=%CD%\.cache\go-mod"
set "CGO_ENABLED=0"

echo [1/3] Testing RTDAL CLI and generators...
"%GOEXE%" test ./cmd/rtdal ./cmd/rtdal-transpile ./internal/command ./internal/cran ./internal/transpile || exit /b 1

echo [2/3] Building Pure-Go Windows onefile...
"%GOEXE%" build -trimpath -ldflags "-s -w -X main.version=0.1.0" -o dist\RTDAL.exe ./cmd/rtdal || exit /b 1

echo [3/3] Verifying executable...
dist\RTDAL.exe --version || exit /b 1
dist\RTDAL.exe help >nul || exit /b 1
dist\RTDAL.exe --license >nul || exit /b 1

echo DONE: %CD%\dist\RTDAL.exe
endlocal
