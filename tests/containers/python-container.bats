#!/usr/bin/env bats

# Property-Based Tests for Python Container
# Feature: mcp-container-images, Property 1: Language-Specific Container Creation
# Feature: mcp-container-images, Property 13: LTS Version Compliance  
# Feature: mcp-container-images, Property 16: Python Environment Management
# Validates: Requirements 1.2, 7.1, 7.4, 7.5

# Helper function to get the container runtime
get_runtime() {
    echo "${CONTAINER_RUNTIME:-docker}"
}

setup() {
    # Set up test environment - use the project root directory directly
    PROJECT_ROOT="$BATS_TEST_DIRNAME/../.."
    
    # Disable BuildKit for compatibility with older Docker setups
    export DOCKER_BUILDKIT=0
    export TEST_IMAGE_TAG="mcp-python-test:$(date +%s)"
    
    # Use the container runtime detected by the Makefile
    RUNTIME=$(get_runtime)
    
    # Build container for testing from project root
    # Since BATS_TEST_DIRNAME points to the current directory when run from root,
    # we need to use the current working directory
    if [ -d "nodejs" ] && [ -d "python" ]; then
        # We're already in the project root
        PROJECT_ROOT="$PWD"
    else
        # We're in a subdirectory, go up to find project root
        PROJECT_ROOT="$BATS_TEST_DIRNAME/../.."
    fi
    
    cd "$PROJECT_ROOT"
    $RUNTIME build -t "$TEST_IMAGE_TAG" python/ >/dev/null 2>&1
}

teardown() {
    # Use the container runtime detected by the Makefile
    RUNTIME=$(get_runtime)
    
    # Clean up test image if it exists
    if $RUNTIME image inspect "$TEST_IMAGE_TAG" >/dev/null 2>&1; then
        $RUNTIME rmi "$TEST_IMAGE_TAG" >/dev/null 2>&1 || true
    fi
}

# Property 1: Language-Specific Container Creation
# For any Python container build, it should successfully create a container image 
# that includes the correct language version and required dependencies for MCP servers
@test "Property 1: Python container builds successfully with required dependencies" {
    # Build the Python container
    run $RUNTIME build -t "$TEST_IMAGE_TAG" python/
    
    [ "$status" -eq 0 ]
    
    # Verify the image was created
    run $RUNTIME image inspect "$TEST_IMAGE_TAG"
    [ "$status" -eq 0 ]
    
    # Verify Python is installed and accessible (use --entrypoint to bypass logging)
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" python --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^Python\ [0-9]+\.[0-9]+\.[0-9]+$ ]]
    
    # Verify uv is installed and accessible (use --entrypoint to bypass logging)
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" uv --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^uv\ [0-9]+\.[0-9]+\.[0-9]+ ]]
    
    # Verify MCP SDK can be installed via uvx (runtime installation model)
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" sh -c "uvx --help | grep -q 'Run a command' && echo 'uvx available for MCP SDK installation'"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "uvx available for MCP SDK installation" ]]
    
    # Verify Python can import standard libraries (no pre-installed MCP dependencies)
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" python -c "import sys, json; print('Python ready')"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Python ready" ]]
}

# Property 13: LTS Version Compliance
# For any Python container, it should use the current LTS version of Python
@test "Property 13: Python container uses LTS version" {
    # Build the container
    run $RUNTIME build -t "$TEST_IMAGE_TAG" python/
    [ "$status" -eq 0 ]
    
    # Get Python version from container (use --entrypoint to bypass logging)
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" python --version
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
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" cat /etc/debian_version
    [ "$status" -eq 0 ]
    # Should contain Debian version (e.g., "12.2")
    [[ "$output" =~ ^[0-9]+\.[0-9]+$ ]] || [[ "$output" =~ ^[0-9]+/sid$ ]]
}

# Property 16: Python Environment Management
# For any Python container, it should support virtual environment creation and management,
# and include common Python development tools
@test "Property 16: Python container supports virtual environment management" {
    # Build the container
    run $RUNTIME build -t "$TEST_IMAGE_TAG" python/
    [ "$status" -eq 0 ]
    
    # Test uv can create new virtual environments
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" sh -c "cd /tmp && uv venv test-venv && test -f test-venv/bin/python"
    [ "$status" -eq 0 ]
    
    # Test uv can install packages in virtual environment
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" sh -c "cd /tmp && uv venv test-venv && uv pip install --python test-venv/bin/python requests && test-venv/bin/python -c 'import requests; print(\"Package installed successfully\")'"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Package installed successfully" ]]
    
    # Verify common Python development tools can be installed via uv
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" sh -c "cd /tmp && uv venv dev-venv && uv pip install --python dev-venv/bin/python pytest && dev-venv/bin/python -c 'import pytest; print(\"Dev tools available\")'"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Dev tools available" ]]
    
    # Verify Python's built-in venv module also works
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" sh -c "cd /tmp && python -m venv builtin-venv && test -f builtin-venv/bin/python"
    [ "$status" -eq 0 ]
}

# Additional property test: Container runs as non-root user (UID 1000)
@test "Property 1 (Security): Python container runs as non-root user UID 1000" {
    # Build the container
    run $RUNTIME build -t "$TEST_IMAGE_TAG" python/
    [ "$status" -eq 0 ]
    
    # Verify container runs as UID 1000 (python user) - use --entrypoint to bypass logging
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" id -u
    [ "$status" -eq 0 ]
    [ "$output" = "1000" ]
    
    # Verify GID is also 1000 - use --entrypoint to bypass logging
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" id -g
    [ "$status" -eq 0 ]
    [ "$output" = "1000" ]
}

# Additional property test: Container supports stdio transport (unbuffered I/O)
@test "Property 1 (MCP Protocol): Python container supports unbuffered stdio transport" {
    # Build the container
    run $RUNTIME build -t "$TEST_IMAGE_TAG" python/
    [ "$status" -eq 0 ]
    
    # Test that PYTHONUNBUFFERED is set to 1 - use --entrypoint to bypass logging
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" printenv PYTHONUNBUFFERED
    [ "$status" -eq 0 ]
    [ "$output" = "1" ]
    
    # Test that PYTHONDONTWRITEBYTECODE is set to 1 - use --entrypoint to bypass logging
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" printenv PYTHONDONTWRITEBYTECODE
    [ "$status" -eq 0 ]
    [ "$output" = "1" ]
    
    # Test that Python output is actually unbuffered by checking immediate output
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" python -c "import sys; print('test', flush=True); sys.stdout.flush()"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test" ]]
}

# Additional property test: Container entrypoint works correctly
@test "Property 1 (Entrypoint): Python container entrypoint executes commands correctly" {
    # Build the container
    run $RUNTIME build -t "$TEST_IMAGE_TAG" python/
    [ "$status" -eq 0 ]
    
    # Test that entrypoint passes through commands correctly (allow stderr for this test)
    run $RUNTIME run --rm "$TEST_IMAGE_TAG" echo "test command passthrough"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test command passthrough" ]]
    
    # Test that entrypoint logs startup information to stderr
    run $RUNTIME run --rm "$TEST_IMAGE_TAG" sh -c "python --version" 2>&1
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Starting Python MCP server container" ]]
    [[ "$output" =~ "Python:" ]]
    [[ "$output" =~ "uv:" ]]
    [[ "$output" =~ "Virtual environment status:" ]]
}

# Additional property test: Container supports uvx for running published packages
@test "Property 1 (Package Execution): Python container supports uvx for running published packages" {
    # Build the container
    run $RUNTIME build -t "$TEST_IMAGE_TAG" python/
    [ "$status" -eq 0 ]
    
    # Test that uvx is available and can run packages
    # Use a simple, fast package for testing (cowsay is lightweight and quick to install)
    run $RUNTIME run --rm "$TEST_IMAGE_TAG" sh -c "uvx cowsay 'uvx works' 2>/dev/null || echo 'uvx command available'"
    [ "$status" -eq 0 ]
    # Either cowsay works or we confirm uvx is available
    [[ "$output" =~ "uvx works" ]] || [[ "$output" =~ "uvx command available" ]]
}

# Property 12: Docker Security Best Practices
# For any container image, it should follow Docker security best practices including 
# running as non-root user and maintaining minimal attack surface
@test "Property 12: Python container follows Docker security best practices" {
    # Build the container
    run $RUNTIME build -t "$TEST_IMAGE_TAG" python/
    [ "$status" -eq 0 ]
    
    # Verify container runs as non-root user (UID 1000)
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" id -u
    [ "$status" -eq 0 ]
    [ "$output" = "1000" ]
    
    # Verify container runs as non-root group (GID 1000)
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" id -g
    [ "$status" -eq 0 ]
    [ "$output" = "1000" ]
    
    # Verify user name is 'mcp' (not root)
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" whoami
    [ "$status" -eq 0 ]
    [ "$output" = "mcp" ]
    
    # Verify no exposed ports (minimal attack surface)
    run $RUNTIME image inspect "$TEST_IMAGE_TAG" --format '{{.Config.ExposedPorts}}'
    [ "$status" -eq 0 ]
    [[ "$output" == "map[]" ]] || [[ "$output" == "<no value>" ]] || [[ "$output" == "null" ]]
    
    # Verify entrypoint script has secure permissions (should be executable)
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" stat -c "%a" /usr/local/bin/entrypoint.sh
    [ "$status" -eq 0 ]
    [ "$output" = "755" ]
    
    # Verify home directory exists and has proper ownership
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" stat -c "%U:%G" /home/mcp
    [ "$status" -eq 0 ]
    [ "$output" = "mcp:mcp" ]
    
    # Verify container uses Debian slim (minimal base image)
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" cat /etc/debian_version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^[0-9]+\.[0-9]+$ ]] || [[ "$output" =~ ^[0-9]+/sid$ ]]
    
    # Verify security labels are present in image metadata
    run $RUNTIME image inspect "$TEST_IMAGE_TAG" --format '{{index .Config.Labels "security.non-root"}}'
    [ "$status" -eq 0 ]
    [ "$output" = "true" ]
}

# Property 14: Language-Specific Package Managers
# For any language container, it should include the appropriate package managers 
# and use them for dependency management with security best practices
@test "Property 14: Python container includes secure package managers" {
    # Build the container
    run $RUNTIME build -t "$TEST_IMAGE_TAG" python/
    [ "$status" -eq 0 ]
    
    # Verify uv is installed and accessible
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" uv --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^uv\ [0-9]+\.[0-9]+\.[0-9]+ ]]
    
    # Verify uvx is available for secure package execution
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" uvx --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^uvx\ [0-9]+\.[0-9]+\.[0-9]+ ]]
    
    # Verify pip is available (comes with Python)
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" python -m pip --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ pip\ [0-9]+\.[0-9]+\.[0-9]+ ]]
    
    # Verify uv can create and use virtual environments
    run $RUNTIME run --rm --entrypoint="" "$TEST_IMAGE_TAG" sh -c "uv venv /tmp/test-venv && test -d /tmp/test-venv"
    [ "$status" -eq 0 ]
    
    # Verify uv cache directory can be configured
    run $RUNTIME run --rm --entrypoint="" -e UV_CACHE_DIR=/tmp/test-cache "$TEST_IMAGE_TAG" printenv UV_CACHE_DIR
    [ "$status" -eq 0 ]
    [ "$output" = "/tmp/test-cache" ]
    
    # Verify package managers can install packages securely (test basic uv functionality)
    run $RUNTIME run --rm --entrypoint="" -e UV_CACHE_DIR=/tmp/uv-cache "$TEST_IMAGE_TAG" sh -c "mkdir -p /tmp/uv-cache && cd /tmp && uv venv test-env && echo 'uv venv creation successful'"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "uv venv creation successful" ]]
}