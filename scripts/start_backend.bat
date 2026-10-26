@echo off
chcp 65001 >nul
title MedTrust Go 核心业务与安全网关服务 (Port 8080)
echo ==============================================================================
echo [MedTrust] 正在启动 Go 核心业务与安全网关服务 (端口 8080)...
echo ==============================================================================
cd /d "%~dp0..\backend"
where go >nul 2>nul
if %ERRORLEVEL% EQU 0 (
    echo [MedTrust] 正在编译最新 Go 后端源码...
    go build -o medtrust_backend.exe ./cmd/main.go
)
if exist medtrust_backend.exe (
    medtrust_backend.exe
) else (
    go run ./cmd/main.go
)
pause
