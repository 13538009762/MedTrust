@echo off
chcp 65001 >nul
echo =====================================================================
echo           MedTrust Hyperledger Fabric 2.5 本地网络关闭
echo =====================================================================

docker compose -f deploy\fabric\docker-compose-test-net.yaml down -v --remove-orphans
for /f "tokens=*" %%i in ('docker ps -a --filter "name=dev-peer" -q') do docker rm -f %%i >nul 2>&1
if exist "deploy\fabric\channel-config\medchannel.block" del "deploy\fabric\channel-config\medchannel.block" >nul 2>&1
if exist "deploy\fabric\medchannel.block" del "deploy\fabric\medchannel.block" >nul 2>&1

echo =====================================================================
echo Fabric Network Stopped Successfully.
echo =====================================================================
