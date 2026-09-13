@echo off
chcp 65001 >nul
title MedTrust Python 受控 AI Agent 服务
echo ==============================================================================
echo [MedTrust] 正在启动 Python FastAPI 受控 AI Agent 独立服务 (端口 8000)...
echo ==============================================================================
cd /d "%~dp0ai_agent"
python -m uvicorn main:app --host 127.0.0.1 --port 8000
pause
