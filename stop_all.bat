@echo off
chcp 65001 >nul
title MedTrust Stop All Services
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\stop_services.ps1"
echo.
pause
