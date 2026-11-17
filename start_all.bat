@echo off
chcp 65001 >nul
cd /d "%~dp0"
echo ==============================================================================
echo [MedTrust] 跨医院医疗数据可信共享系统 2.0 - 一键并发启动服务
echo ==============================================================================
echo 0. 检查 Hyperledger Fabric 区块链容器集群...
docker ps 2>nul | findstr "peer0.org1.example.com" >nul 2>&1
if %errorlevel% equ 0 goto FABRIC_OK

echo    [TIP] Fabric 容器集群未运行，正在启动区块链网络 [唤醒 Docker Desktop]...
start "MedTrust Blockchain Network" cmd /c call "%~dp0scripts\start_fabric.bat"
echo    [WAIT] 正在等待 Fabric 区块链节点就绪 (localhost:7051)...
set /a FABRIC_WAIT=0
:WAIT_FABRIC_LOOP
timeout /t 2 /nobreak >nul
set /a FABRIC_WAIT+=2
docker ps 2>nul | findstr "peer0.org1.example.com" >nul 2>&1
if %errorlevel% neq 0 (
    if %FABRIC_WAIT% geq 90 goto FABRIC_TIMEOUT
    goto WAIT_FABRIC_LOOP
)
netstat -ano | findstr ":7051 " | findstr "LISTENING" >nul 2>&1
if %errorlevel% neq 0 (
    if %FABRIC_WAIT% geq 90 goto FABRIC_TIMEOUT
    goto WAIT_FABRIC_LOOP
)
timeout /t 2 /nobreak >nul
echo    [OK] Fabric 区块链集群与通信端口 7051 均已就绪 (耗时 %FABRIC_WAIT% 秒).
goto STEP_SERVICES

:FABRIC_TIMEOUT
echo    [WARN] 等待 Fabric 就绪超时，正在继续拉起业务服务 (后台自愈引擎将自动完成连接重试)...
goto STEP_SERVICES

:FABRIC_OK
echo    [OK] Fabric 容器集群正常运行中.

:STEP_SERVICES
echo.
echo 1. 正在拉起 Go 核心业务与安全网关服务 (8080 端口)...
start "MedTrust Go Backend (Port 8080)" cmd /k call "%~dp0scripts\start_backend.bat"

echo 2. 正在拉起 Python FastAPI 受控 AI Agent (8000 端口)...
start "MedTrust Python AI Agent (Port 8000)" cmd /k call "%~dp0scripts\start_ai.bat"

echo 3. 正在拉起 Vue 3 现代化前端 (5173 端口)...
start "MedTrust Vue3 Frontend (Port 5173)" cmd /k call "%~dp0scripts\start_frontend.bat"

echo.
echo ==============================================================================
echo 所有子服务已在独立终端并发启动！
echo 正在为您打开默认浏览器进入登录演示界面...
ping 127.0.0.1 -n 4 >nul
start "" "http://localhost:5173/login"
echo.
echo 系统登录预置账号提示：
echo [1] 医院A (总院区) 李建国医生: doc_li / 123456
echo [2] 医院B (分院区) 王明德医生: doc_wang / 123456
echo [3] 就诊患者 张三:            pat_zhang / 123456
echo [4] 卫健监管专员 陈建华:      supervisor / 123456
echo [5] 平台系统管理员:           admin / 123456
echo ==============================================================================
echo 按任意键关闭本启动主控台（后台服务将继续在各自独立窗口运行）...
pause >nul