@echo off
chcp 65001 >nul
cd /d "%~dp0"
echo ==============================================================================
echo [MedTrust 2.0] 全栈 Docker 一键交付与运行向导 (0 本地语言依赖)
echo ==============================================================================

:: 1. 检查并唤醒 Docker Engine
echo [1/4] 检查 Docker 引擎运行状态...
docker info >nul 2>&1
if %errorlevel% equ 0 goto DOCKER_READY

echo    [INFO] 探测到 Docker 守护进程未启动，正在尝试唤醒 Docker Desktop...
set "DOCKER_EXE="
if exist "C:\Program Files\Docker\Docker\Docker Desktop.exe" set "DOCKER_EXE=C:\Program Files\Docker\Docker\Docker Desktop.exe"
if not defined DOCKER_EXE if exist "%ProgramFiles%\Docker\Docker\Docker Desktop.exe" set "DOCKER_EXE=%ProgramFiles%\Docker\Docker\Docker Desktop.exe"
if not defined DOCKER_EXE if exist "%LOCALAPPDATA%\Programs\Docker\Docker Desktop.exe" set "DOCKER_EXE=%LOCALAPPDATA%\Programs\Docker\Docker Desktop.exe"

if defined DOCKER_EXE (
    start "" "%DOCKER_EXE%"
    echo    [WAIT] 正在等待 Docker Engine 引擎初始化就绪（约 15~30 秒）...
    set /a RETRY_COUNT=0
    :WAIT_DOCKER_LOOP
    timeout /t 3 /nobreak >nul
    set /a RETRY_COUNT+=3
    docker info >nul 2>&1
    if %errorlevel% equ 0 goto DOCKER_READY
    if %RETRY_COUNT% geq 90 goto DOCKER_TIMEOUT
    echo    [WAIT] Docker 引擎启动中... (已等待 %RETRY_COUNT% 秒)
    goto WAIT_DOCKER_LOOP
)

:DOCKER_TIMEOUT
echo ==============================================================================
echo [ERROR] 无法自动启动 Docker，请先手动启动 Docker Desktop，并在运行后再双击此脚本！
echo ==============================================================================
pause
exit /b 1

:DOCKER_READY
echo    [OK] Docker 引擎运行正常.

:: 2. 检查本地开发端口占用情况 (防范与 start_all.bat 冲突)
netstat -ano | findstr ":8080 " | findstr "LISTENING" >nul 2>&1
if %errorlevel% equ 0 (
    docker ps | findstr "medtrust-backend" >nul 2>&1
    if %errorlevel% neq 0 (
        echo.
        echo    [WARN] 探测到宿主机端口 8080 已被本地进程占用！
        echo    如果您刚才运行了本地模式 (start_all.bat)，请先双击 stop_all.bat 释放端口。
        echo.
    )
)

:: 3. 检查并拉起 Hyperledger Fabric 2.5 区块链网络
echo.
echo [2/4] 检查 Hyperledger Fabric 区块链容器集群与智能合约...
docker ps | findstr "peer0.org1.example.com" >nul 2>&1
if %errorlevel% equ 0 (
    echo    [OK] Fabric 区块链集群已在运行.
) else (
    echo    [INIT] 正在启动 Fabric 容器集群 (Orderer, Org1 Peer, Org2 Peer, CLI)...
    docker compose -f deploy\fabric\docker-compose-test-net.yaml up -d
    timeout /t 3 /nobreak >nul
    echo    [INIT] 正在加入 medchannel 统一医疗通道...
    docker exec cli /bin/bash /opt/gopath/src/github.com/hyperledger/fabric/peer/scripts/network.sh channel
    echo    [INIT] 正在部署 medical 链上智能合约...
    docker exec cli /bin/bash /opt/gopath/src/github.com/hyperledger/fabric/peer/scripts/network.sh deploy
    echo    [OK] 区块链智能合约部署完成.
)

:: 4. 构建并启动 MedTrust 全栈容器集群 (MySQL, Redis, Backend, AI Agent, Frontend)
echo.
echo [3/4] 启动 MedTrust 全栈业务服务 (首次启动将自动执行多阶段轻量构建)...
docker compose up -d --build
if %errorlevel% neq 0 (
    echo [ERROR] 容器拉起失败，请检查 Docker 报错日志.
    pause
    exit /b 1
)

:: 5. 提示就绪并打开前端
echo.
echo [4/4] 正在等待业务网关与前端就绪...
timeout /t 5 /nobreak >nul

echo ==============================================================================
echo [SUCCESS] MedTrust 2.0 全栈服务已全部在 Docker 容器集群中成功运行！
echo ==============================================================================
echo 服务端点拓扑：
echo   - 前端 Web 界面:     http://localhost:5173
echo   - Go 核心业务网关:   http://localhost:8080 (API: /api/v1)
echo   - Python AI Agent:   http://localhost:8000
echo   - MySQL 数据库:      localhost:3307 (容器内 3306，预置全量数据)
echo   - Redis 缓存服务:    localhost:6379
echo   - Fabric Peer1:      localhost:7051 (Org1MSP)
echo   - Fabric Peer2:      localhost:9051 (Org2MSP)
echo ------------------------------------------------------------------------------
echo 预置系统演示账号：
echo   [1] 医院A (总院区) 李建国医生: doc_li / 123456  (或 doc_a / 123456)
echo   [2] 医院B (分院区) 王明德医生: doc_wang / 123456 (或 doc_b / 123456)
echo   [3] 就诊患者 张三:            pat_zhang / 123456
echo   [4] 卫健监管专员:             supervisor / 123456
echo   [5] 平台系统管理员:           admin / 123456
echo ==============================================================================
echo 正在为您打开默认浏览器进入登录系统...
start http://localhost:5173/login
echo.
echo 如需停止 Docker 全栈集群，请直接双击运行 stop_docker.bat！
pause
