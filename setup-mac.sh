#!/bin/bash

# =============================================================================
# DockerVerse - macOS Development Environment Setup
# =============================================================================
# This script sets up all required tools and dependencies for DockerVerse
# development on macOS.
#
# Usage:
#   chmod +x setup-mac.sh
#   ./setup-mac.sh
#
# Author: Victor Heredia
# Date: 2026-02-07
# Version: 1.0.0
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# =============================================================================
# Helper Functions
# =============================================================================

print_header() {
    echo ""
    echo -e "${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${PURPLE}  $1${NC}"
    echo -e "${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

print_step() {
    echo -e "${BLUE}▶${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${CYAN}ℹ${NC} $1"
}

# Compare semantic versions: returns 0 if $1 >= $2
version_gte() {
    [ "$(printf '%s\n' "$2" "$1" | sort -V | head -n1)" = "$2" ]
}

# Extract version number from string
extract_version() {
    echo "$1" | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1
}

# Check if command exists
command_exists() {
    command -v "$1" &> /dev/null
}

# =============================================================================
# Tool Definitions
# =============================================================================

# Format: "name|min_version|brew_package|version_command|description"
TOOLS=(
    "brew|4.0.0|MANUAL|brew --version|Homebrew Package Manager"
    "git|2.40.0|git|git --version|Version Control"
    "node|20.0.0|node@20|node --version|Node.js Runtime"
    "npm|10.0.0|WITH_NODE|npm --version|Node Package Manager"
    "go|1.22.0|go|go version|Go Programming Language"
    "docker|24.0.0|docker|docker --version|Docker Container Engine"
    "gh|2.40.0|gh|gh --version|GitHub CLI"
    "code|1.80.0|visual-studio-code|code --version|Visual Studio Code"
    "jq|1.6|jq|jq --version|JSON Processor"
)

# VS Code Extensions
VSCODE_EXTENSIONS=(
    "svelte.svelte-vscode"
    "golang.go"
    "bradlc.vscode-tailwindcss"
    "ms-azuretools.vscode-docker"
    "GitHub.copilot"
    "GitHub.copilot-chat"
    "esbenp.prettier-vscode"
    "dbaeumer.vscode-eslint"
)

# =============================================================================
# Main Logic
# =============================================================================

print_header "🐳 DockerVerse - macOS Development Setup"

echo -e "${CYAN}This script will install and configure all tools needed for${NC}"
echo -e "${CYAN}DockerVerse development on your Mac.${NC}"
echo ""
echo -e "${YELLOW}The following will be checked/installed:${NC}"
echo "  • Homebrew (package manager)"
echo "  • Git (version control)"
echo "  • Node.js 20 LTS (frontend runtime)"
echo "  • Go 1.22+ (backend language)"
echo "  • Docker Desktop (containerization)"
echo "  • GitHub CLI (repository management)"
echo "  • VS Code + Extensions"
echo "  • SSH configuration for Raspberry Pis"
echo ""

read -p "Continue? (y/n) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

# =============================================================================
# Step 1: Check/Install Homebrew
# =============================================================================

print_header "Step 1: Homebrew Package Manager"

if command_exists brew; then
    BREW_VERSION=$(extract_version "$(brew --version)")
    print_success "Homebrew is installed (v$BREW_VERSION)"
    
    print_step "Updating Homebrew..."
    brew update
    print_success "Homebrew updated"
else
    print_warning "Homebrew not found. Installing..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    
    # Add to PATH for Apple Silicon
    if [[ $(uname -m) == 'arm64' ]]; then
        echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> ~/.zprofile
        eval "$(/opt/homebrew/bin/brew shellenv)"
    fi
    
    print_success "Homebrew installed"
fi

# =============================================================================
# Step 2: Check/Install Required Tools
# =============================================================================

print_header "Step 2: Required Development Tools"

INSTALL_REQUIRED=()
UPGRADE_SUGGESTED=()

for tool_def in "${TOOLS[@]}"; do
    IFS='|' read -r name min_ver brew_pkg ver_cmd description <<< "$tool_def"
    
    print_step "Checking $description ($name)..."
    
    if [[ "$brew_pkg" == "MANUAL" ]]; then
        continue  # Skip, already handled
    fi
    
    if command_exists "$name"; then
        CURRENT_VER=$(extract_version "$($ver_cmd 2>&1)")
        
        if version_gte "$CURRENT_VER" "$min_ver"; then
            print_success "$name v$CURRENT_VER (>= $min_ver required) ✓"
        else
            print_warning "$name v$CURRENT_VER found, but v$min_ver+ required"
            UPGRADE_SUGGESTED+=("$name|$brew_pkg|$CURRENT_VER|$min_ver")
        fi
    else
        print_warning "$name not found"
        if [[ "$brew_pkg" != "WITH_NODE" ]]; then
            INSTALL_REQUIRED+=("$name|$brew_pkg|$min_ver|$description")
        fi
    fi
done

# =============================================================================
# Step 3: Install Missing Tools
# =============================================================================

if [ ${#INSTALL_REQUIRED[@]} -gt 0 ]; then
    print_header "Step 3: Installing Missing Tools"
    
    for tool_info in "${INSTALL_REQUIRED[@]}"; do
        IFS='|' read -r name brew_pkg min_ver description <<< "$tool_info"
        
        print_step "Installing $description..."
        
        case "$brew_pkg" in
            "visual-studio-code")
                brew install --cask visual-studio-code
                ;;
            "docker")
                brew install --cask docker
                print_warning "Please open Docker Desktop to complete installation"
                ;;
            "node@20")
                brew install node@20
                # Link node@20
                brew link --overwrite node@20 2>/dev/null || true
                ;;
            *)
                brew install "$brew_pkg"
                ;;
        esac
        
        print_success "$name installed"
    done
else
    print_header "Step 3: Installing Missing Tools"
    print_success "All required tools are already installed!"
fi

# =============================================================================
# Step 4: Handle Version Upgrades
# =============================================================================

if [ ${#UPGRADE_SUGGESTED[@]} -gt 0 ]; then
    print_header "Step 4: Version Upgrades"
    
    echo -e "${YELLOW}The following tools have older versions:${NC}"
    for upgrade_info in "${UPGRADE_SUGGESTED[@]}"; do
        IFS='|' read -r name brew_pkg current_ver min_ver <<< "$upgrade_info"
        echo "  • $name: v$current_ver → v$min_ver+ recommended"
    done
    echo ""
    
    read -p "Upgrade these tools? (y/n) " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        for upgrade_info in "${UPGRADE_SUGGESTED[@]}"; do
            IFS='|' read -r name brew_pkg current_ver min_ver <<< "$upgrade_info"
            
            print_step "Upgrading $name..."
            
            case "$brew_pkg" in
                "visual-studio-code")
                    brew upgrade --cask visual-studio-code || true
                    ;;
                "docker")
                    brew upgrade --cask docker || true
                    ;;
                *)
                    brew upgrade "$brew_pkg" || true
                    ;;
            esac
        done
        print_success "Upgrades complete"
    else
        print_warning "Skipping upgrades. Some features may not work correctly."
    fi
else
    print_header "Step 4: Version Upgrades"
    print_success "All tools are at required versions!"
fi

# =============================================================================
# Step 5: VS Code Extensions
# =============================================================================

print_header "Step 5: VS Code Extensions"

if command_exists code; then
    print_step "Installing VS Code extensions..."
    
    for ext in "${VSCODE_EXTENSIONS[@]}"; do
        if code --list-extensions | grep -q "^$ext$"; then
            print_success "$ext (already installed)"
        else
            print_step "Installing $ext..."
            code --install-extension "$ext" --force 2>/dev/null || true
            print_success "$ext installed"
        fi
    done
else
    print_warning "VS Code CLI not found. Please install extensions manually."
fi

# =============================================================================
# Step 6: SSH Configuration for Raspberry Pis
# =============================================================================

print_header "Step 6: SSH Configuration"

SSH_CONFIG_FILE="$HOME/.ssh/config"

# Create .ssh directory if not exists
mkdir -p "$HOME/.ssh"
chmod 700 "$HOME/.ssh"

# Check if SSH key exists
if [ ! -f "$HOME/.ssh/id_rsa" ] && [ ! -f "$HOME/.ssh/id_ed25519" ]; then
    print_step "Generating SSH key..."
    read -p "Enter email for SSH key: " ssh_email
    ssh-keygen -t ed25519 -C "$ssh_email" -f "$HOME/.ssh/id_ed25519" -N ""
    print_success "SSH key generated at ~/.ssh/id_ed25519"
fi

# Add Raspberry Pi configurations
RASPI_CONFIG="
# DockerVerse Raspberry Pi Hosts
Host raspi-main
    HostName 192.168.1.145
    User pi
    IdentityFile ~/.ssh/id_ed25519
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null

Host raspi-secondary
    HostName 192.168.1.146
    User pi
    IdentityFile ~/.ssh/id_ed25519
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
"

if [ -f "$SSH_CONFIG_FILE" ]; then
    if grep -q "raspi-main" "$SSH_CONFIG_FILE"; then
        print_success "Raspberry Pi SSH config already exists"
    else
        echo "$RASPI_CONFIG" >> "$SSH_CONFIG_FILE"
        print_success "Added Raspberry Pi hosts to SSH config"
    fi
else
    echo "$RASPI_CONFIG" > "$SSH_CONFIG_FILE"
    chmod 600 "$SSH_CONFIG_FILE"
    print_success "Created SSH config with Raspberry Pi hosts"
fi

print_info "To copy your SSH key to Raspberry Pi, run:"
echo "    ssh-copy-id -i ~/.ssh/id_ed25519 pi@192.168.1.145"

# =============================================================================
# Step 7: Clone Repository (if not already cloned)
# =============================================================================

print_header "Step 7: Project Setup"

DOCKERVERSE_DIR="$HOME/Projects/dockerverse"

if [ -d "$DOCKERVERSE_DIR" ]; then
    print_success "DockerVerse directory exists at $DOCKERVERSE_DIR"
else
    echo ""
    print_info "DockerVerse project not found locally."
    read -p "Enter GitHub repository URL (or press Enter to skip): " repo_url
    
    if [ -n "$repo_url" ]; then
        mkdir -p "$HOME/Projects"
        git clone "$repo_url" "$DOCKERVERSE_DIR"
        print_success "Repository cloned to $DOCKERVERSE_DIR"
    else
        print_warning "Skipping repository clone. Clone manually when ready."
    fi
fi

# =============================================================================
# Step 8: Install Node Dependencies
# =============================================================================

print_header "Step 8: Node.js Dependencies"

if [ -d "$DOCKERVERSE_DIR/frontend" ]; then
    print_step "Installing frontend dependencies..."
    cd "$DOCKERVERSE_DIR/frontend"
    npm install
    print_success "Frontend dependencies installed"
else
    print_warning "Frontend directory not found. Run 'npm install' in frontend/ later."
fi

# =============================================================================
# Step 9: Verify Docker
# =============================================================================

print_header "Step 9: Docker Verification"

if command_exists docker; then
    if docker info &> /dev/null; then
        print_success "Docker daemon is running"
        
        DOCKER_VER=$(docker --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')
        print_info "Docker version: $DOCKER_VER"
        
        # Check if docker-compose is available
        if command_exists docker-compose || docker compose version &> /dev/null; then
            print_success "Docker Compose is available"
        else
            print_warning "Docker Compose not found. It should come with Docker Desktop."
        fi
    else
        print_warning "Docker is installed but not running."
        print_info "Please open Docker Desktop to start the daemon."
    fi
else
    print_warning "Docker not found. Please install Docker Desktop."
fi

# =============================================================================
# Step 10: Final Summary
# =============================================================================

print_header "🎉 Setup Complete!"

echo -e "${GREEN}Your macOS development environment is ready!${NC}"
echo ""
echo -e "${CYAN}Quick Reference:${NC}"
echo ""
echo "  📁 Project Location:"
echo "     $DOCKERVERSE_DIR"
echo ""
echo "  🔧 Development Commands:"
echo "     cd $DOCKERVERSE_DIR/frontend"
echo "     npm run dev           # Start frontend dev server"
echo ""
echo "  🚀 Deploy Commands:"
echo "     ./deploy-to-raspi.sh  # Deploy to Raspberry Pi"
echo ""
echo "  🔌 SSH to Raspberry Pi:"
echo "     ssh raspi-main        # Connect to main Raspi"
echo "     ssh raspi-secondary   # Connect to secondary Raspi"
echo ""
echo "  📖 Documentation:"
echo "     See DEVELOPMENT_CONTINUATION_GUIDE.md for full details"
echo ""

# Create a quick reference file
cat > "$HOME/.dockerverse-quickref" << 'EOF'
# DockerVerse Quick Reference
# ===========================

# SSH to Raspberry Pis
alias raspi1='ssh raspi-main'
alias raspi2='ssh raspi-secondary'

# Deploy to production
alias dv-deploy='cd ~/Projects/dockerverse && ./deploy-to-raspi.sh'

# Start frontend dev
alias dv-dev='cd ~/Projects/dockerverse/frontend && npm run dev'

# Check container status
alias dv-status='ssh raspi-main "docker ps | grep dockerverse"'

# View container logs
alias dv-logs='ssh raspi-main "docker logs -f dockerverse"'
EOF

print_info "Quick reference aliases saved to ~/.dockerverse-quickref"
print_info "Add to your shell: source ~/.dockerverse-quickref"

echo ""
echo -e "${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  Happy coding! 🐳${NC}"
echo -e "${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
