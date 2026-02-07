#!/bin/bash

# =============================================================================
# DockerVerse - Deploy to Raspberry Pi Script (macOS)
# =============================================================================
# This script syncs the local code to Raspberry Pi and rebuilds the container.
#
# Usage:
#   ./deploy-to-raspi.sh              # Standard deploy
#   ./deploy-to-raspi.sh --no-cache   # Force rebuild without cache
#   ./deploy-to-raspi.sh --quick      # Sync only, no rebuild
#
# Author: Victor Heredia
# Date: 2026-02-07
# =============================================================================

set -e

# Configuration
RASPI_HOST="pi@192.168.1.145"
RASPI_PATH="/home/pi/dockerverse"
COMPOSE_FILE="docker-compose.unified.yml"
CONTAINER_NAME="dockerverse"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Parse arguments
NO_CACHE=""
QUICK_MODE=false

for arg in "$@"; do
    case $arg in
        --no-cache)
            NO_CACHE="--no-cache"
            ;;
        --quick)
            QUICK_MODE=true
            ;;
        --help)
            echo "Usage: $0 [--no-cache] [--quick]"
            echo ""
            echo "Options:"
            echo "  --no-cache   Force rebuild without Docker cache"
            echo "  --quick      Sync files only, no rebuild"
            echo ""
            exit 0
            ;;
    esac
done

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  🐳 DockerVerse Deploy to Raspberry Pi${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# =============================================================================
# Step 1: Check SSH connection
# =============================================================================

echo -e "${YELLOW}▶${NC} Checking SSH connection..."

if ! ssh -o ConnectTimeout=5 -o BatchMode=yes "$RASPI_HOST" "echo ok" &>/dev/null; then
    echo -e "${RED}✗${NC} Cannot connect to $RASPI_HOST"
    echo "  Please check:"
    echo "  • Raspberry Pi is powered on and connected to network"
    echo "  • SSH key is copied (run: ssh-copy-id $RASPI_HOST)"
    echo "  • IP address is correct"
    exit 1
fi

echo -e "${GREEN}✓${NC} SSH connection OK"

# =============================================================================
# Step 2: Sync files
# =============================================================================

echo ""
echo -e "${YELLOW}▶${NC} Syncing files to Raspberry Pi..."

rsync -avz --progress \
    --exclude 'node_modules' \
    --exclude '.git' \
    --exclude 'test-*' \
    --exclude '*.log' \
    --exclude '.DS_Store' \
    --exclude 'test-screenshots' \
    --exclude 'backups' \
    ./ "$RASPI_HOST:$RASPI_PATH/"

echo -e "${GREEN}✓${NC} Files synced"

# If quick mode, stop here
if [ "$QUICK_MODE" = true ]; then
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  ✓ Quick sync complete!${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    exit 0
fi

# =============================================================================
# Step 3: Stop current container
# =============================================================================

echo ""
echo -e "${YELLOW}▶${NC} Stopping current container..."

ssh "$RASPI_HOST" "cd $RASPI_PATH && docker-compose -f $COMPOSE_FILE down 2>/dev/null || true"

echo -e "${GREEN}✓${NC} Container stopped"

# =============================================================================
# Step 4: Build new image
# =============================================================================

echo ""
echo -e "${YELLOW}▶${NC} Building new Docker image..."

if [ -n "$NO_CACHE" ]; then
    echo -e "${BLUE}ℹ${NC} Using --no-cache flag"
fi

ssh "$RASPI_HOST" "cd $RASPI_PATH && docker-compose -f $COMPOSE_FILE build $NO_CACHE"

echo -e "${GREEN}✓${NC} Image built"

# =============================================================================
# Step 5: Start container
# =============================================================================

echo ""
echo -e "${YELLOW}▶${NC} Starting container..."

ssh "$RASPI_HOST" "cd $RASPI_PATH && docker-compose -f $COMPOSE_FILE up -d"

echo -e "${GREEN}✓${NC} Container started"

# =============================================================================
# Step 6: Wait for health check
# =============================================================================

echo ""
echo -e "${YELLOW}▶${NC} Waiting for container to be healthy..."

MAX_WAIT=60
WAIT_COUNT=0

while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
    HEALTH=$(ssh "$RASPI_HOST" "docker inspect --format='{{.State.Health.Status}}' $CONTAINER_NAME 2>/dev/null || echo 'unknown'")
    
    case $HEALTH in
        "healthy")
            echo -e "${GREEN}✓${NC} Container is healthy!"
            break
            ;;
        "unhealthy")
            echo -e "${RED}✗${NC} Container is unhealthy!"
            echo "Checking logs..."
            ssh "$RASPI_HOST" "docker logs --tail 50 $CONTAINER_NAME"
            exit 1
            ;;
        *)
            echo -ne "\r  Waiting... ($WAIT_COUNT/$MAX_WAIT seconds)   "
            sleep 2
            WAIT_COUNT=$((WAIT_COUNT + 2))
            ;;
    esac
done

if [ $WAIT_COUNT -ge $MAX_WAIT ]; then
    echo -e "${YELLOW}⚠${NC} Health check timeout. Container may still be starting."
fi

# =============================================================================
# Step 7: Verify deployment
# =============================================================================

echo ""
echo -e "${YELLOW}▶${NC} Verifying deployment..."

# Get container status
CONTAINER_STATUS=$(ssh "$RASPI_HOST" "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' | grep $CONTAINER_NAME")
echo "  $CONTAINER_STATUS"

# Test API endpoint
echo ""
echo -e "${YELLOW}▶${NC} Testing API endpoint..."

API_RESPONSE=$(ssh "$RASPI_HOST" "curl -s -o /dev/null -w '%{http_code}' http://localhost:3007/api/health 2>/dev/null || echo '000'")

if [ "$API_RESPONSE" = "200" ]; then
    echo -e "${GREEN}✓${NC} API is responding (HTTP 200)"
else
    echo -e "${YELLOW}⚠${NC} API returned HTTP $API_RESPONSE"
fi

# =============================================================================
# Done!
# =============================================================================

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  🎉 Deployment Complete!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "  🌐 Access DockerVerse at:"
echo "     http://192.168.1.145:3007"
echo ""
echo "  📋 Useful commands:"
echo "     ssh raspi-main 'docker logs -f dockerverse'  # View logs"
echo "     ssh raspi-main 'docker exec -it dockerverse sh'  # Shell access"
echo ""
