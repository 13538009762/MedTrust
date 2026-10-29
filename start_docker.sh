#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=============================================================================="
echo "[MedTrust 2.0] Docker Full-Stack Startup Wizard (Zero local dev environment)"
echo "=============================================================================="

# 1. Check Docker
if ! docker info >/dev/null 2>&1; then
    echo "[ERROR] Docker is not running. Please start Docker Engine first."
    exit 1
fi
echo "[1/4] Docker daemon is running."

# 2. Check and start Fabric Network
echo "[2/4] Checking Hyperledger Fabric cluster..."
if ! docker ps | grep "peer0.org1.example.com" >/dev/null 2>&1; then
    echo "      Starting Fabric containers..."
    docker compose -f deploy/fabric/docker-compose-test-net.yaml up -d
    sleep 3
    echo "      Joining channel medchannel..."
    docker exec cli /bin/bash /opt/gopath/src/github.com/hyperledger/fabric/peer/scripts/network.sh channel
    echo "      Deploying chaincode medical..."
    docker exec cli /bin/bash /opt/gopath/src/github.com/hyperledger/fabric/peer/scripts/network.sh deploy
else
    echo "      Fabric cluster is already running."
fi

# 3. Start Business Containers
echo "[3/4] Building and starting MedTrust services (MySQL, Redis, Backend, AI, Frontend)..."
docker compose up -d --build

# 4. Ready
echo "[4/4] Services started successfully!"
echo "=============================================================================="
echo "Topology endpoints:"
echo "  - Frontend Web:     http://localhost:5173"
echo "  - Go Backend API:   http://localhost:8080"
echo "  - Python AI Agent:  http://localhost:8000"
echo "  - MySQL Database:   localhost:3307"
echo "  - Redis Cache:      localhost:6379"
echo "=============================================================================="
