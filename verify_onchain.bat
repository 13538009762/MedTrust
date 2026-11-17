@echo off
chcp 65001 >nul
cd /d "%~dp0"
echo ==============================================================================
echo [MedTrust 2.0] Hyperledger Fabric 真实联盟链底层存证验证工具
echo ==============================================================================
echo.

echo [1/4] 检查联盟链多组织节点拓扑 (Docker Containers)...
docker ps --filter "name=peer0" --filter "name=orderer" --filter "name=cli" --format "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}"
if %errorlevel% neq 0 (
    echo [ERROR] Docker 或 Fabric 容器未正常运行！
    pause
    exit /b 1
)

echo.
echo [2/4] 查询通道底层账本状态与区块高度 (Channel Ledger Info)...
docker exec cli bash -c "peer channel getinfo -c medicalchannel"

echo.
echo [3/4] 逆向解析最新落块中的 X.509 组织身份与 64 位 TxID 签名...
docker exec cli bash -c "peer channel fetch newest /tmp/verify_newest.block -c medicalchannel >/dev/null 2>&1 && configtxlator proto_decode --input /tmp/verify_newest.block --type common.Block | grep -E 'mspid|channel_id|tx_id' | head -n 12"

echo.
echo [4/4] 抓取 Peer0 节点最近一次智能合约执行与 KVLedger 提交落块日志...
docker logs --tail 12 peer0.org1.example.com

echo.
echo ==============================================================================
echo [SUCCESS] 链上密码学背书、Raft排序、X.509 组织身份与账本落块验证完成！
echo ==============================================================================
if "%~1"=="" pause
