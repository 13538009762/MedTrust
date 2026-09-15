@echo off
chcp 65001 >nul
cd /d "%~dp0.."
echo =====================================================================
echo           MedTrust Hyperledger Fabric 2.5 本地网络启动向导
echo =====================================================================

:: 1. 检查并自动拉起 Docker 守护进程
echo [1/5] 检查 Docker 运行环境...
docker info >nul 2>&1
if %errorlevel% equ 0 goto DOCKER_IS_READY

echo ---------------------------------------------------------------------
echo [INFO] 探测到 Docker 守护进程未启动，正在尝试自动拉起 Docker Desktop...
set "DOCKER_EXE="
if exist "C:\Program Files\Docker\Docker\Docker Desktop.exe" set "DOCKER_EXE=C:\Program Files\Docker\Docker\Docker Desktop.exe"
if not defined DOCKER_EXE if exist "%ProgramFiles%\Docker\Docker\Docker Desktop.exe" set "DOCKER_EXE=%ProgramFiles%\Docker\Docker\Docker Desktop.exe"
if not defined DOCKER_EXE if exist "%LOCALAPPDATA%\Programs\Docker\Docker Desktop.exe" set "DOCKER_EXE=%LOCALAPPDATA%\Programs\Docker\Docker Desktop.exe"

if not defined DOCKER_EXE (
    echo =====================================================================
    echo [ERROR] 未能在系统默认路径找到 Docker Desktop.exe！
    echo 请手动启动 Docker Desktop，并等待系统托盘处 Docker 运行就绪后再运行此脚本。
    echo =====================================================================
    pause
    exit /b 1
)

echo [INFO] 找到 Docker 路径，正在启动: "%DOCKER_EXE%"
start "" "%DOCKER_EXE%"
echo [WAIT] 正在等待 Docker Engine 引擎初始化就绪（通常需要 15~35 秒）...
set /a RETRY_COUNT=0

:WAIT_DOCKER_LOOP
timeout /t 3 /nobreak >nul
set /a RETRY_COUNT+=3
docker info >nul 2>&1
if %errorlevel% equ 0 goto DOCKER_IS_READY
if %RETRY_COUNT% geq 90 goto DOCKER_TIMEOUT
echo [WAIT] Docker 引擎正在启动中，请稍候... (已等待 %RETRY_COUNT% 秒)
goto WAIT_DOCKER_LOOP

:DOCKER_TIMEOUT
echo =====================================================================
echo [ERROR] 等待 Docker Engine 启动超时（已等待 90 秒）！
echo 请检查 Docker Desktop 是否启动正常或正在等待用户授权。
echo =====================================================================
pause
exit /b 1

:DOCKER_IS_READY
echo [OK] Docker Engine 正常运行中.

:: 2. 检查并生成证书与 MSP 资产
echo [2/5] 检查组织证书 (Org1=Hospital A, Org2=Hospital B, Orderer)...
if not exist "deploy\fabric\crypto-config\ordererOrganizations" (
    echo 正在使用 Fabric Tools 官方镜像生成组织证书与 MSP 配置...
    docker run --rm -v "%cd%\deploy\fabric:/fabric" hyperledger/fabric-tools:2.5 bash -c "cd /fabric && cryptogen generate --config=crypto-config.yaml --output=crypto-config"
    if not exist "backend\fabric_crypto" mkdir "backend\fabric_crypto"
    xcopy /E /I /Y "deploy\fabric\crypto-config\*" "backend\fabric_crypto\" >nul
)
echo [OK] 组织证书与 MSP 配置就绪.

:: 3. 启动 Fabric 2.5 容器集群
echo [3/5] 启动 Fabric 容器 (Orderer, Peer0.Org1, Peer0.Org2, CLI)...
docker compose -f deploy\fabric\docker-compose-test-net.yaml up -d
if %errorlevel% neq 0 (
    echo [ERROR] 启动 Fabric 容器失败，请检查 Docker 端口占用与资源限制.
    pause
    exit /b 1
)

:: 4. 创建通道 medchannel 并将 Peer 节点加入
echo [4/5] 检查统一通道 medchannel...
timeout /t 3 >nul
docker exec cli /bin/bash /opt/gopath/src/github.com/hyperledger/fabric/peer/scripts/network.sh channel

:: 5. 部署智能合约 medical
echo [5/5] 打包、审批并提交 medical 医疗智能合约...
docker exec cli /bin/bash /opt/gopath/src/github.com/hyperledger/fabric/peer/scripts/network.sh deploy

echo =====================================================================
echo Fabric Network Started Successfully
echo =====================================================================
echo 节点拓扑：
echo   - Orderer:    orderer.example.com:7050
echo   - Hospital A: peer0.org1.example.com:7051 (Org1MSP)
echo   - Hospital B: peer0.org2.example.com:9051 (Org2MSP)
echo   - 通道:        medchannel
echo   - 智能合约:    medical (Go Contract-API 1.2)
echo =====================================================================
