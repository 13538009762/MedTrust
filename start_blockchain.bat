@echo off
chcp 65001 >nul
cd /d "%~dp0"
echo ==============================================================================
echo [MedTrust] 一键拉起 Docker 与 Hyperledger Fabric 区块链联盟网络
echo ==============================================================================
call "%~dp0scripts\start-fabric.bat"
echo.
echo 按任意键退出本窗口...
pause >nul
