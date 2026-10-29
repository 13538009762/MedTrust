@echo off
chcp 65001 >nul
cd /d "%~dp0"
echo ==============================================================================
echo [MedTrust 2.0] 正在优雅关闭 Docker 全栈容器集群...
echo ==============================================================================

echo [1/2] 正在停止业务容器 (Frontend, Backend, AI Agent, MySQL, Redis)...
docker compose down

echo [2/2] 正在停止 Hyperledger Fabric 区块链容器网络...
docker compose -f deploy\fabric\docker-compose-test-net.yaml down

echo ==============================================================================
echo [OK] MedTrust 所有 Docker 容器已安全停止！
echo 提示：MySQL 数据库数据卷与区块链历史资产均已安全持久化在本地 Docker Volume 中，
echo 下次启动将无缝续存。
echo ==============================================================================
timeout /t 3 >nul
