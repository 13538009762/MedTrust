#!/bin/bash
set -e

echo "====================================================================="
echo "          MedTrust Hyperledger Fabric 2.5 本地网络启动向导"
echo "====================================================================="

# 1. 检查 Docker 运行环境
echo "[1/5] 检查 Docker 运行环境..."
if ! docker info >/dev/null 2>&1; then
    echo "====================================================================="
    echo "[ERROR] 探测到 Docker 守护进程未启动！"
    echo "请启动 Docker Desktop，并等待 Docker Engine 启动就绪后再运行此脚本。"
    echo "====================================================================="
    exit 1
fi
echo "[OK] Docker Engine 正常运行中."

# 2. 检查组织证书
echo "[2/5] 检查组织证书 (Org1=Hospital A, Org2=Hospital B, Orderer)..."
if [ ! -d "deploy/fabric/crypto-config/ordererOrganizations" ]; then
    echo "正在使用 Fabric Tools 官方镜像生成组织证书与 MSP 配置..."
    docker run --rm -v "$(pwd)/deploy/fabric:/fabric" hyperledger/fabric-tools:2.5 bash -c "cd /fabric && cryptogen generate --config=crypto-config.yaml --output=crypto-config"
    mkdir -p backend/fabric_crypto
    cp -r deploy/fabric/crypto-config/* backend/fabric_crypto/
fi
echo "[OK] 组织证书与 MSP 配置就绪."

# 3. 启动 Fabric 2.5 容器集群
echo "[3/5] 启动 Fabric 容器 (Orderer, Peer0.Org1, Peer0.Org2, CLI)..."
docker compose -f deploy/fabric/docker-compose-test-net.yaml up -d

# 4. 创建通道 medchannel
echo "[4/5] 创建统一通道 medchannel 并加入组织节点..."
sleep 3
docker exec cli /bin/bash /opt/gopath/src/github.com/hyperledger/fabric/peer/scripts/network.sh channel

# 5. 部署智能合约 medical
echo "[5/5] 打包、审批并提交 medical 医疗智能合约..."
docker exec cli /bin/bash /opt/gopath/src/github.com/hyperledger/fabric/peer/scripts/network.sh deploy

echo "====================================================================="
echo "Fabric Network Started Successfully"
echo "====================================================================="
