@echo off
chcp 65001 >nul
title MedTrust Go 核心业务与安全网关服务
echo ==============================================================================
echo [MedTrust] 正在启动 Go 核心业务与安全网关服务 (端口 8080)...
echo ==============================================================================
cd /d "%~dp0backend"
if exist medtrust_backend.exe (
    medtrust_backend.exe
) else (
    go run ./cmd/main.go
)
pause
