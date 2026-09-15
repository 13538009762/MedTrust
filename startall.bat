@echo off
chcp 65001 >nul
cd /d "%~dp0"
echo ==============================================================================
echo [MedTrust] 跨医院医疗数据可信共享系统 2.0 - 一键并发启动服务
echo ==============================================================================
echo 0. 检查 Hyperledger Fabric 区块链容器集群...
docker ps | findstr "peer0.org1.example.com" >nul 2>&1
if %errorlevel% equ 0 goto FABRIC_OK

echo    [TIP] Fabric 容器集群未运行，正在启动区块链网络 [唤醒 Docker Desktop]...
start "MedTrust Blockchain Network" cmd /c "%~dp0start_blockchain.bat"
goto STEP_SERVICES

:FABRIC_OK
echo    [OK] Fabric 容器集群正常运行中.

:STEP_SERVICES
echo.
echo 1. 正在拉起 Go 核心业务与安全网关服务 (8080 端口)...
start "MedTrust Go Backend (Port 8080)" cmd /k "%~dp0start_backend.bat"

echo 2. 正在拉起 Python FastAPI 受控 AI Agent (8000 端口)...
start "MedTrust Python AI Agent (Port 8000)" cmd /k "%~dp0start_ai.bat"

echo 3. 正在拉起 Vue 3 现代化前端 (5173 端口)...
start "MedTrust Vue3 Frontend (Port 5173)" cmd /k "%~dp0start_frontend.bat"

echo.
echo ==============================================================================
echo 所有子服务已在独立终端并发启动！
echo 正在为您打开默认浏览器进入登录演示界面...
ping 127.0.0.1 -n 4 >nul
start http://localhost:5173/login
echo.
echo 系统登录预置账号提示：
echo • 医院A李建国医生: doc_a / 123456
echo • 医院B王明德医生: doc_b / 123456
echo • 患者张三:        pat_zhang / 123456
echo • 卫健监管专员:    supervisor / 123456
echo • 系统管理员:      admin / 123456
echo ==============================================================================
echo 按任意键关闭本启动主控台（后台服务将继续在各自独立窗口运行）...
pause >nul
