#!/usr/bin/env bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=============================================================================="
echo "[MedTrust 2.0] Stopping all Docker containers..."
echo "=============================================================================="

docker compose down
docker compose -f deploy/fabric/docker-compose-test-net.yaml down

echo "[OK] All MedTrust Docker containers stopped cleanly."
