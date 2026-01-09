#!/bin/bash
# Setup development environment script

set -e

echo -e "\033[0;34mSetting up development environment...\033[0m"
echo -e "\033[0;33mInstalling required tools...\033[0m"

# Install bats
if ! command -v bats >/dev/null 2>&1; then
    echo "Installing bats..."
    sudo apt-get update && sudo apt-get install -y bats
fi

# Install jq
if ! command -v jq >/dev/null 2>&1; then
    echo "Installing jq..."
    sudo apt-get install -y jq
fi

# Install curl
if ! command -v curl >/dev/null 2>&1; then
    echo "Installing curl..."
    sudo apt-get install -y curl
fi

# Install checkmake
if ! command -v checkmake >/dev/null 2>&1; then
    echo "Installing checkmake..."
    if command -v go >/dev/null 2>&1; then
        go install github.com/mrtazz/checkmake/cmd/checkmake@latest
    else
        echo -e "\033[0;33mGo not available, skipping checkmake installation\033[0m"
        echo -e "\033[0;33mInstall manually: go install github.com/mrtazz/checkmake/cmd/checkmake@latest\033[0m"
    fi
fi

echo -e "\033[0;32m✓ Development environment setup complete\033[0m"