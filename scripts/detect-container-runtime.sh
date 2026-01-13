#!/bin/bash

# Container runtime detection script
# Uses the same logic as the Makefile to detect available container runtime

detect_runtime() {
    if command -v podman >/dev/null 2>&1; then
        echo "podman"
    elif command -v nerdctl >/dev/null 2>&1; then
        echo "nerdctl"
    elif command -v finch >/dev/null 2>&1; then
        echo "finch"
    elif command -v docker.exe >/dev/null 2>&1; then
        echo "docker.exe"
    elif command -v docker >/dev/null 2>&1; then
        echo "docker"
    else
        echo ""
    fi
}

# Output the detected runtime
RUNTIME=$(detect_runtime)

if [ -z "$RUNTIME" ]; then
    echo "Error: No container runtime found" >&2
    echo "Container Runtime Options:" >&2
    echo "1. Docker: Install Docker Desktop or Docker Engine" >&2
    echo "2. Podman: Install Podman (docker-compatible)" >&2
    echo "3. nerdctl: Install nerdctl with containerd" >&2
    echo "4. Finch: Install AWS Finch (macOS/Linux)" >&2
    echo "5. WSL2: Use docker.exe via Docker Desktop" >&2
    exit 1
fi

echo "$RUNTIME"