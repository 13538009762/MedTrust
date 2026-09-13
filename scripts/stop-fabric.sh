#!/bin/bash
echo "====================================================================="
echo "          MedTrust Hyperledger Fabric 2.5 本地网络关闭"
echo "====================================================================="

docker compose -f deploy/fabric/docker-compose-test-net.yaml down -v --remove-orphans
docker ps -a --filter "name=dev-peer" -q | xargs -r docker rm -f >/dev/null 2>&1 || true
rm -f deploy/fabric/channel-config/medchannel.block deploy/fabric/medchannel.block

echo "====================================================================="
echo "Fabric Network Stopped Successfully."
echo "====================================================================="
