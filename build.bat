@echo off
setlocal enabledelayedexpansion

rem Derive version from env or git; fallback to dev
if "%VERSION%"=="" (
  for /f "delims=" %%a in ('git describe --tags --always --dirty 2^>nul') do set "VERSION=%%a"
)
if "%VERSION%"=="" set "VERSION=dev"
set "LDFLAGS=-s -w -X main.version=%VERSION%"

set GOOS=windows
set GOARCH=amd64
go build -ldflags "%LDFLAGS%" -o adbee-windows-amd64.exe .

set GOOS=linux
set GOARCH=amd64
go build -ldflags "%LDFLAGS%" -o adbee-linux-amd64 .

set GOOS=darwin
set GOARCH=amd64
go build -ldflags "%LDFLAGS%" -o adbee-darwin-amd64 .

set GOOS=darwin
set GOARCH=arm64
go build -ldflags "%LDFLAGS%" -o adbee-darwin-arm64 .

echo Build complete (version: %VERSION%). Binaries created:
dir adbee-*

endlocal
