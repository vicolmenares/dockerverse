#!/bin/bash

# =============================================================================
# DockerVerse - Sync to Raspberry Pi Script (macOS)
# =============================================================================
# Quick sync of source files without rebuild.
# Useful for rapid iteration during development.
#
# Usage:
#   ./sync-to-raspi.sh              # Sync all
#   ./sync-to-raspi.sh frontend     # Sync frontend only
#   ./sync-to-raspi.sh backend      # Sync backend only
#
# Author: Victor Heredia
# Date: 2026-02-07
# =============================================================================

set -e

# Configuration
RASPI_HOST="pi@192.168.1.145"
RASPI_PATH="/home/pi/dockerverse"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SYNC_TARGET="${1:-all}"

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo -e "${BLUE}▶${NC} Syncing $SYNC_TARGET to Raspberry Pi..."

RSYNC_OPTS="-avz --progress --exclude 'node_modules' --exclude '.git' --exclude '.DS_Store'"

case $SYNC_TARGET in
    frontend)
        rsync $RSYNC_OPTS \
            ./frontend/src/ "$RASPI_HOST:$RASPI_PATH/frontend/src/"
        ;;
    backend)
        rsync $RSYNC_OPTS \
            ./backend/main.go "$RASPI_HOST:$RASPI_PATH/backend/"
        ;;
    all|*)
        rsync $RSYNC_OPTS \
            --exclude 'test-*' \
            --exclude '*.log' \
            --exclude 'test-screenshots' \
            --exclude 'backups' \
            ./ "$RASPI_HOST:$RASPI_PATH/"
        ;;
esac

echo -e "${GREEN}✓${NC} Sync complete!"
echo ""
echo -e "${YELLOW}Note:${NC} To apply changes, rebuild the container:"
echo "  ssh raspi-main 'cd /home/pi/dockerverse && docker-compose -f docker-compose.unified.yml up -d --build'"
