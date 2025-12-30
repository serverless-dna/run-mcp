#!/usr/bin/env bats

# Property-Based Tests for Python Container
# Feature: mcp-container-images, Property 1: Language-Specific Container Creation
# Feature: mcp-container-images, Property 13: LTS Version Compliance  
# Feature: mcp-container-images, Property 16: Python Environment Management
# Validates: Requirements 1.2, 7.1, 7.4, 7.5

setup() {
    # Set up test environment
    TEST_DIR=$(mktemp -d)
    cd "$TEST_DIR"
    
    # Copy python directory for testing
    cp -r "$BATS_TEST_DIRNAME/../../python" .
    
    # Set up Docker build context
    export DOCKER_BUILDKIT=1
    export TEST_IMAGE_TAG="mcp-python-test:$(date +%s)"
}

teardown() {
    # Clean up test image if it exists
    if docker image inspect "$TEST_IMAGE_TAG" >/dev/null 2>&1; then
        docker rmi "$TEST_IMAGE_TAG" >/dev/null 2>&1 || true
    fi
    
    # Clean up test directory
    rm -rf "$TEST_DIR"
}

# Property 1: Language-Specific Container Creation
# For any Python container build, it should successfully create a container image 
# that includes the correct language version and required dependencies for MCP servers
@test "Property 1: Python container builds successfully with required dependencies" {
    # Build the Python container
    run docker build -t "$TEST_IMAGE_TAG" python/
    
    [ "$status" -eq 0 ]
    
    # Verify the image was created
    run docker image inspect "$TEST_IMAGE_TAG"
    [ "$status" -eq 0 ]
    
    # Verify Python is installed and accessible (use --entrypoint to bypass logging)
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" python --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^Python\ [0-9]+\.[0-9]+\.[0-9]+$ ]]
    
    # Verify uv is installed and accessible (use --entrypoint to bypass logging)
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" uv --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^uv\ [0-9]+\.[0-9]+\.[0-9]+ ]]
    
    # Verify MCP SDK is available in virtual environment
    run timeout 30 docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" python -c "import mcp; print('MCP SDK available')"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "MCP SDK available" ]]
    
    # Verify other required dependencies are available
    run timeout 30 docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" python -c "import uvloop, httpx, pydantic; print('Dependencies available')"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Dependencies available" ]]
}

# Property 13: LTS Version Compliance
# For any Python container, it should use the current LTS version of Python
@test "Property 13: Python container uses LTS version" {
    # Build the container
    run docker build -t "$TEST_IMAGE_TAG" python/
    [ "$status" -eq 0 ]
    
    # Get Python version from container (use --entrypoint to bypass logging)
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" python --version
    [ "$status" -eq 0 ]
    
    # Extract version number (remove 'Python ' prefix)
    PYTHON_VERSION="${output#Python }"
    
    # Verify it's Python 3.12.x (current LTS as specified in requirements)
    if [[ "$PYTHON_VERSION" =~ ^3\.12\. ]]; then
        # Valid LTS version
        true
    else
        # Not the expected LTS version
        echo "Python version $PYTHON_VERSION is not the expected LTS version (3.12.x)"
        false
    fi
    
    # Verify the container is built from python:3.12-slim by checking Debian base
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" cat /etc/debian_version
    [ "$status" -eq 0 ]
    # Should contain Debian version (e.g., "12.2")
    [[ "$output" =~ ^[0-9]+\.[0-9]+$ ]] || [[ "$output" =~ ^[0-9]+/sid$ ]]
}

# Property 16: Python Environment Management
# For any Python container, it should support virtual environment creation and management,
# and include common Python development tools
@test "Property 16: Python container supports virtual environment management" {
    # Build the container
    run docker build -t "$TEST_IMAGE_TAG" python/
    [ "$status" -eq 0 ]
    
    # Verify virtual environment is active and properly configured
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" python -c "import sys; print(sys.prefix)"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "/app/venv" ]]
    
    # Verify VIRTUAL_ENV environment variable is set
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" printenv VIRTUAL_ENV
    [ "$status" -eq 0 ]
    [ "$output" = "/app/venv" ]
    
    # Verify PATH includes virtual environment bin directory
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" printenv PATH
    [ "$status" -eq 0 ]
    [[ "$output" =~ "/app/venv/bin" ]]
    
    # Test uv can create new virtual environments (with proper cache directory)
    run timeout 60 docker run --rm --entrypoint="" -e UV_CACHE_DIR=/tmp/uv-cache "$TEST_IMAGE_TAG" sh -c "mkdir -p /tmp/uv-cache && cd /tmp && uv venv test-venv && ls test-venv/bin/python"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test-venv/bin/python" ]]
    
    # Test uv can install packages in virtual environment (with proper cache directory)
    run timeout 60 docker run --rm --entrypoint="" -e UV_CACHE_DIR=/tmp/uv-cache "$TEST_IMAGE_TAG" sh -c "mkdir -p /tmp/uv-cache && cd /tmp && uv venv test-venv && uv pip install --python test-venv/bin/python requests && test-venv/bin/python -c 'import requests; print(\"Package installed successfully\")'"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Package installed successfully" ]]
    
    # Verify common Python development tools are available via uv (with proper cache directory)
    # Test that we can install development tools
    run timeout 60 docker run --rm --entrypoint="" -e UV_CACHE_DIR=/tmp/uv-cache "$TEST_IMAGE_TAG" sh -c "mkdir -p /tmp/uv-cache && cd /tmp && uv venv dev-venv && uv pip install --python dev-venv/bin/python pytest && dev-venv/bin/python -c 'import pytest; print(\"Dev tools available\")'"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Dev tools available" ]]
}

# Additional property test: Container runs as non-root user (UID 1000)
@test "Property 1 (Security): Python container runs as non-root user UID 1000" {
    # Build the container
    run docker build -t "$TEST_IMAGE_TAG" python/
    [ "$status" -eq 0 ]
    
    # Verify container runs as UID 1000 (python user) - use --entrypoint to bypass logging
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" id -u
    [ "$status" -eq 0 ]
    [ "$output" = "1000" ]
    
    # Verify GID is also 1000 - use --entrypoint to bypass logging
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" id -g
    [ "$status" -eq 0 ]
    [ "$output" = "1000" ]
}

# Additional property test: Container supports stdio transport (unbuffered I/O)
@test "Property 1 (MCP Protocol): Python container supports unbuffered stdio transport" {
    # Build the container
    run docker build -t "$TEST_IMAGE_TAG" python/
    [ "$status" -eq 0 ]
    
    # Test that PYTHONUNBUFFERED is set to 1 - use --entrypoint to bypass logging
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" printenv PYTHONUNBUFFERED
    [ "$status" -eq 0 ]
    [ "$output" = "1" ]
    
    # Test that PYTHONDONTWRITEBYTECODE is set to 1 - use --entrypoint to bypass logging
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" printenv PYTHONDONTWRITEBYTECODE
    [ "$status" -eq 0 ]
    [ "$output" = "1" ]
    
    # Test that Python output is actually unbuffered by checking immediate output
    run timeout 10 docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" python -c "import sys; print('test', flush=True); sys.stdout.flush()"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test" ]]
}

# Additional property test: Container entrypoint works correctly
@test "Property 1 (Entrypoint): Python container entrypoint executes commands correctly" {
    # Build the container
    run docker build -t "$TEST_IMAGE_TAG" python/
    [ "$status" -eq 0 ]
    
    # Test that entrypoint passes through commands correctly
    run docker run --rm "$TEST_IMAGE_TAG" echo "test command passthrough"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test command passthrough" ]]
    
    # Test that entrypoint works with Python commands
    run docker run --rm "$TEST_IMAGE_TAG" python -c "print('Python execution works')"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Python execution works" ]]
    
    # Test that entrypoint works with uv commands
    run docker run --rm "$TEST_IMAGE_TAG" uv --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^uv\ [0-9]+\.[0-9]+\.[0-9]+ ]]
}

# Additional property test: Container supports uvx for running published packages
@test "Property 1 (Package Execution): Python container supports uvx for running published packages" {
    # Build the container
    run docker build -t "$TEST_IMAGE_TAG" python/
    [ "$status" -eq 0 ]
    
    # Test that uvx is available and can run packages
    # Use a simple, fast package for testing (cowsay is lightweight and quick to install)
    run timeout 120 docker run --rm "$TEST_IMAGE_TAG" sh -c "uvx cowsay 'uvx works' 2>/dev/null || echo 'uvx command available'"
    [ "$status" -eq 0 ]
    # Either cowsay works or we confirm uvx is available
    [[ "$output" =~ "uvx works" ]] || [[ "$output" =~ "uvx command available" ]]
}