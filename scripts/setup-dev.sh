#!/bin/bash
# Setup development environment script

set -e

echo -e "\033[0;34mSetting up development environment...\033[0m"
echo -e "\033[0;33mInstalling required tools...\033[0m"

# Detect operating system
OS="$(uname -s)"
case "${OS}" in
    Linux*)     MACHINE=Linux;;
    Darwin*)    MACHINE=Mac;;
    CYGWIN*)    MACHINE=Cygwin;;
    MINGW*)     MACHINE=MinGw;;
    *)          MACHINE="UNKNOWN:${OS}"
esac

echo "Detected OS: ${MACHINE}"

# Warn about sudo requirements
case "${MACHINE}" in
    Mac)
        echo -e "\033[0;33mNote: On macOS, this script may install Homebrew which requires sudo access.\033[0m"
        ;;
    Linux)
        echo -e "\033[0;33mNote: On Linux, this script requires sudo access to install packages via apt-get.\033[0m"
        ;;
esac
echo ""

# Check what's already installed
echo ""
echo -e "\033[0;34mChecking current tool status:\033[0m"
command -v bats >/dev/null 2>&1 && echo -e "  bats: \033[0;32m✓ installed\033[0m" || echo -e "  bats: \033[0;33m✗ not found\033[0m"
command -v jq >/dev/null 2>&1 && echo -e "  jq: \033[0;32m✓ installed\033[0m" || echo -e "  jq: \033[0;33m✗ not found\033[0m"
command -v curl >/dev/null 2>&1 && echo -e "  curl: \033[0;32m✓ installed\033[0m" || echo -e "  curl: \033[0;33m✗ not found\033[0m"
command -v go >/dev/null 2>&1 && echo -e "  go: \033[0;32m✓ installed\033[0m" || echo -e "  go: \033[0;33m✗ not found\033[0m"
command -v checkmake >/dev/null 2>&1 && echo -e "  checkmake: \033[0;32m✓ installed\033[0m" || echo -e "  checkmake: \033[0;33m✗ not found\033[0m"
command -v actionlint >/dev/null 2>&1 && echo -e "  actionlint: \033[0;32m✓ installed\033[0m" || echo -e "  actionlint: \033[0;33m✗ not found\033[0m"
echo ""

# Function to install packages on Linux (Ubuntu/Debian)
install_linux() {
    local package=$1
    echo "Installing $package on Linux..."
    if ! dpkg -l | grep -q "^ii  $package "; then
        sudo apt-get update && sudo apt-get install -y "$package"
    else
        echo "$package is already installed"
    fi
}

# Function to install packages on macOS
install_mac() {
    local package=$1
    echo "Installing $package on macOS..."
    
    # Check if Homebrew is installed
    if ! command -v brew >/dev/null 2>&1; then
        echo -e "\033[0;33mHomebrew not found.\033[0m"
        echo -e "\033[0;33mHomebrew is required to install development tools on macOS.\033[0m"
        echo -e "\033[0;33mThis will require sudo access for installation.\033[0m"
        echo ""
        read -p "Do you want to install Homebrew? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            echo -e "\033[0;33mInstalling Homebrew (this may take a few minutes and will ask for your password)...\033[0m"
            /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
            
            # Add Homebrew to PATH for the current session
            if [[ -f "/opt/homebrew/bin/brew" ]]; then
                # Apple Silicon Mac
                eval "$(/opt/homebrew/bin/brew shellenv)"
                echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> ~/.zprofile
            elif [[ -f "/usr/local/bin/brew" ]]; then
                # Intel Mac
                eval "$(/usr/local/bin/brew shellenv)"
                echo 'eval "$(/usr/local/bin/brew shellenv)"' >> ~/.zprofile
            fi
            echo -e "\033[0;32m✓ Homebrew installed successfully\033[0m"
        else
            echo -e "\033[0;33mSkipping Homebrew installation. You'll need to install $package manually.\033[0m"
            return 1
        fi
    fi
    
    # Check if package is already installed
    if brew list "$package" >/dev/null 2>&1; then
        echo -e "\033[0;32m✓ $package is already installed via Homebrew\033[0m"
    else
        echo "Installing $package via Homebrew..."
        brew install "$package"
        echo -e "\033[0;32m✓ $package installed successfully\033[0m"
    fi
}

# Install bats
if ! command -v bats >/dev/null 2>&1; then
    case "${MACHINE}" in
        Linux)
            install_linux "bats" || echo -e "\033[0;31mFailed to install bats on Linux\033[0m"
            ;;
        Mac)
            install_mac "bats-core" || echo -e "\033[0;31mFailed to install bats-core on macOS\033[0m"
            ;;
        *)
            echo -e "\033[0;31mUnsupported OS: ${MACHINE}\033[0m"
            echo -e "\033[0;33mPlease install bats manually\033[0m"
            ;;
    esac
else
    echo -e "\033[0;32m✓ bats is already installed\033[0m"
fi

# Install jq
if ! command -v jq >/dev/null 2>&1; then
    case "${MACHINE}" in
        Linux)
            install_linux "jq" || echo -e "\033[0;31mFailed to install jq on Linux\033[0m"
            ;;
        Mac)
            install_mac "jq" || echo -e "\033[0;31mFailed to install jq on macOS\033[0m"
            ;;
        *)
            echo -e "\033[0;31mUnsupported OS: ${MACHINE}\033[0m"
            echo -e "\033[0;33mPlease install jq manually\033[0m"
            ;;
    esac
else
    echo -e "\033[0;32m✓ jq is already installed\033[0m"
fi

# Install curl (usually pre-installed on macOS)
if ! command -v curl >/dev/null 2>&1; then
    case "${MACHINE}" in
        Linux)
            install_linux "curl" || echo -e "\033[0;31mFailed to install curl on Linux\033[0m"
            ;;
        Mac)
            echo -e "\033[0;33mcurl is usually pre-installed on macOS\033[0m"
            install_mac "curl" || echo -e "\033[0;31mFailed to install curl on macOS\033[0m"
            ;;
        *)
            echo -e "\033[0;31mUnsupported OS: ${MACHINE}\033[0m"
            echo -e "\033[0;33mPlease install curl manually\033[0m"
            ;;
    esac
else
    echo -e "\033[0;32m✓ curl is already installed\033[0m"
fi

# Install checkmake
if ! command -v checkmake >/dev/null 2>&1; then
    echo "Installing checkmake..."
    if command -v go >/dev/null 2>&1; then
        go install github.com/checkmake/checkmake/cmd/checkmake@latest
        
        # Add Go bin to PATH if not already there
        GOPATH=$(go env GOPATH)
        if [[ ":$PATH:" != *":$GOPATH/bin:"* ]]; then
            echo "Adding Go bin directory to PATH..."
            export PATH="$PATH:$GOPATH/bin"
            
            # Add to shell profile for persistence
            if [[ "$SHELL" == *"zsh"* ]]; then
                echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
            elif [[ "$SHELL" == *"bash"* ]]; then
                echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
            fi
        fi
        
        echo -e "\033[0;32m✓ checkmake installed\033[0m"
    else
        echo -e "\033[0;33mGo not available, skipping checkmake installation\033[0m"
        echo -e "\033[0;33mInstall Go first, then run: go install github.com/checkmake/checkmake/cmd/checkmake@latest\033[0m"
    fi
else
    echo -e "\033[0;32m✓ checkmake is already installed\033[0m"
fi

# Install actionlint
if ! command -v actionlint >/dev/null 2>&1; then
    echo "Installing actionlint..."
    if command -v go >/dev/null 2>&1; then
        go install github.com/rhysd/actionlint/cmd/actionlint@latest
        
        # Add Go bin to PATH if not already there (in case it wasn't added above)
        GOPATH=$(go env GOPATH)
        if [[ ":$PATH:" != *":$GOPATH/bin:"* ]]; then
            echo "Adding Go bin directory to PATH..."
            export PATH="$PATH:$GOPATH/bin"
        fi
        
        echo -e "\033[0;32m✓ actionlint installed\033[0m"
    else
        echo -e "\033[0;33mGo not available, skipping actionlint installation\033[0m"
        echo -e "\033[0;33mInstall Go first, then run: go install github.com/rhysd/actionlint/cmd/actionlint@latest\033[0m"
    fi
else
    echo -e "\033[0;32m✓ actionlint is already installed\033[0m"
fi

# Setup git hooks
echo "Setting up git hooks..."
if [ -f ".githooks/pre-commit" ]; then
    cp .githooks/pre-commit .git/hooks/pre-commit
    chmod +x .git/hooks/pre-commit
    echo -e "\033[0;32m✓ Pre-commit hook installed\033[0m"
else
    echo -e "\033[0;33mWarning: .githooks/pre-commit not found\033[0m"
fi

# Check if Docker/container runtime is available
echo "Checking container runtime..."
if command -v podman >/dev/null 2>&1; then
    echo -e "\033[0;32m✓ Podman found\033[0m"
elif command -v nerdctl >/dev/null 2>&1; then
    echo -e "\033[0;32m✓ nerdctl found\033[0m"
elif command -v finch >/dev/null 2>&1; then
    echo -e "\033[0;32m✓ Finch found\033[0m"
elif command -v docker >/dev/null 2>&1; then
    echo -e "\033[0;32m✓ Docker found\033[0m"
else
    echo -e "\033[0;33mWarning: No container runtime found\033[0m"
    case "${MACHINE}" in
        Mac)
            echo -e "\033[0;33mFor macOS, install Podman: brew install podman\033[0m"
            echo -e "\033[0;33mOr install Docker Desktop: https://www.docker.com/products/docker-desktop\033[0m"
            ;;
        Linux)
            echo -e "\033[0;33mFor Linux, install Podman: sudo apt-get install podman\033[0m"
            echo -e "\033[0;33mOr install Docker: https://docs.docker.com/engine/install/ubuntu/\033[0m"
            ;;
    esac
fi

echo -e "\033[0;32m✓ Development environment setup complete\033[0m"
echo ""
echo -e "\033[0;34mNext steps:\033[0m"
echo "1. Restart your terminal or run 'source ~/.zshrc' (or ~/.bashrc) to update PATH"
echo "2. Run 'make check-tools' to verify all tools are installed"
echo "3. Run 'make test-dev' for fast development tests"
echo "4. Run 'make test' for full test suite"