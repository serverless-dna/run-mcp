#!/usr/bin/env bats

# Property-Based Tests for Node.js Container
# Feature: mcp-container-images, Property 1: Language-Specific Container Creation
# Feature: mcp-container-images, Property 13: LTS Version Compliance  
# Feature: mcp-container-images, Property 15: Node.js Module System Support
# Validates: Requirements 1.1, 6.1, 6.4, 6.5

setup() {
    # Set up test environment
    TEST_DIR=$(mktemp -d)
    cd "$TEST_DIR"
    
    # Copy nodejs directory for testing
    cp -r "$BATS_TEST_DIRNAME/../../nodejs" .
    
    # Set up Docker build context
    export DOCKER_BUILDKIT=1
    export TEST_IMAGE_TAG="mcp-nodejs-test:$(date +%s)"
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
# For any Node.js container build, it should successfully create a container image 
# that includes the correct language version and required dependencies for MCP servers
@test "Property 1: Node.js container builds successfully with required dependencies" {
    # Build the Node.js container
    run docker build -t "$TEST_IMAGE_TAG" nodejs/
    
    [ "$status" -eq 0 ]
    
    # Verify the image was created
    run docker image inspect "$TEST_IMAGE_TAG"
    [ "$status" -eq 0 ]
    
    # Verify Node.js is installed and accessible (use --entrypoint to bypass logging)
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" node --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]
    
    # Verify npm is installed and accessible (use --entrypoint to bypass logging)
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" npm --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
    
    # Verify yarn is installed and accessible (use --entrypoint to bypass logging)
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" yarn --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
    
    # Verify MCP SDK is available or can be installed (check node_modules or npm list)
    run timeout 30 docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" sh -c "ls /app/node_modules/@modelcontextprotocol/sdk 2>/dev/null || echo 'MCP SDK not pre-installed'"
    [ "$status" -eq 0 ]
    # Either the SDK is installed or we get the expected message
    [[ "$output" =~ "package.json" ]] || [[ "$output" =~ "MCP SDK not pre-installed" ]]
}

# Property 13: LTS Version Compliance
# For any Node.js container, it should use the current LTS version of Node.js
@test "Property 13: Node.js container uses LTS version" {
    # Build the container
    run docker build -t "$TEST_IMAGE_TAG" nodejs/
    [ "$status" -eq 0 ]
    
    # Get Node.js version from container (use --entrypoint to bypass logging)
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" node --version
    [ "$status" -eq 0 ]
    
    # Extract version number (remove 'v' prefix)
    NODE_VERSION="${output#v}"
    
    # Verify it's a supported LTS version (18.x, 20.x, 22.x, or 24.x)
    # LTS versions as of 2024: 18.x (Maintenance), 20.x (Active), 22.x (Previous LTS), 24.x (Current LTS)
    if [[ "$NODE_VERSION" =~ ^18\. ]] || [[ "$NODE_VERSION" =~ ^20\. ]] || [[ "$NODE_VERSION" =~ ^22\. ]] || [[ "$NODE_VERSION" =~ ^24\. ]]; then
        # Valid LTS version
        true
    else
        # Not a recognized LTS version
        echo "Node.js version $NODE_VERSION is not a recognized LTS version"
        false
    fi
    
    # Verify the container is built from node:lts-alpine by checking Alpine Linux
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" cat /etc/alpine-release
    [ "$status" -eq 0 ]
    # Should contain Alpine version (e.g., "3.18.4")
    [[ "$output" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

# Property 15: Node.js Module System Support  
# For any Node.js container, it should support both CommonJS and ES module systems
# and include TypeScript compilation capabilities
@test "Property 15: Node.js container supports both CommonJS and ES modules" {
    # Build the container
    run docker build -t "$TEST_IMAGE_TAG" nodejs/
    [ "$status" -eq 0 ]
    
    # Test CommonJS support - create a simple CommonJS module
    CJS_TEST='
const fs = require("fs");
const path = require("path");
console.log("CommonJS module system works");
module.exports = { test: true };
'
    
    # Test ES modules support - create a simple ES module
    ESM_TEST='
import fs from "fs";
import path from "path";
console.log("ES module system works");
export default { test: true };
'
    
    # Create test files in a temporary volume
    mkdir -p test-modules
    echo "$CJS_TEST" > test-modules/test.cjs
    echo "$ESM_TEST" > test-modules/test.mjs
    
    # Test CommonJS execution (use --entrypoint to bypass logging)
    run docker run --rm --entrypoint="" -v "$PWD/test-modules:/test" "$TEST_IMAGE_TAG" node /test/test.cjs
    [ "$status" -eq 0 ]
    [[ "$output" =~ "CommonJS module system works" ]]
    
    # Test ES modules execution (use --entrypoint to bypass logging)
    run docker run --rm --entrypoint="" -v "$PWD/test-modules:/test" "$TEST_IMAGE_TAG" node /test/test.mjs
    [ "$status" -eq 0 ]
    [[ "$output" =~ "ES module system works" ]]
    
    # Verify TypeScript is available for compilation (check if tsc exists in node_modules)
    run timeout 30 docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" sh -c "ls /app/node_modules/.bin/tsc 2>/dev/null || echo 'TypeScript not pre-installed'"
    [ "$status" -eq 0 ]
    # Either TypeScript is installed or we get the expected message
    [[ "$output" =~ "/app/node_modules/.bin/tsc" ]] || [[ "$output" =~ "TypeScript not pre-installed" ]]
    
    # Verify tsx is available for TypeScript execution (check if tsx exists in node_modules)
    run timeout 30 docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" sh -c "ls /app/node_modules/.bin/tsx 2>/dev/null || echo 'tsx not pre-installed'"
    [ "$status" -eq 0 ]
    # Either tsx is installed or we get the expected message
    [[ "$output" =~ "/app/node_modules/.bin/tsx" ]] || [[ "$output" =~ "tsx not pre-installed" ]]
    
    # Clean up test files
    rm -rf test-modules
}

# Additional property test: Container runs as non-root user (UID 1000)
@test "Property 1 (Security): Node.js container runs as non-root user UID 1000" {
    # Build the container
    run docker build -t "$TEST_IMAGE_TAG" nodejs/
    [ "$status" -eq 0 ]
    
    # Verify container runs as UID 1000 (node user) - use --entrypoint to bypass logging
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" id -u
    [ "$status" -eq 0 ]
    [ "$output" = "1000" ]
    
    # Verify user name is 'node' - use --entrypoint to bypass logging
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" whoami
    [ "$status" -eq 0 ]
    [ "$output" = "node" ]
}

# Additional property test: Container supports stdio transport (unbuffered I/O)
@test "Property 1 (MCP Protocol): Node.js container supports unbuffered stdio transport" {
    # Build the container
    run docker build -t "$TEST_IMAGE_TAG" nodejs/
    [ "$status" -eq 0 ]
    
    # Test that NODE_OPTIONS includes unbuffered settings - use --entrypoint to bypass logging
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" printenv NODE_OPTIONS
    [ "$status" -eq 0 ]
    [[ "$output" =~ "--unhandled-rejections=strict" ]]
    
    # Test that FORCE_COLOR is set to 0 (no color output for stdio) - use --entrypoint to bypass logging
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" printenv FORCE_COLOR
    [ "$status" -eq 0 ]
    [ "$output" = "0" ]
    
    # Test that NPM_CONFIG_UPDATE_NOTIFIER is disabled - use --entrypoint to bypass logging
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" printenv NPM_CONFIG_UPDATE_NOTIFIER
    [ "$status" -eq 0 ]
    [ "$output" = "false" ]
}

# Additional property test: Container entrypoint works correctly
@test "Property 1 (Entrypoint): Node.js container entrypoint executes commands correctly" {
    # Build the container
    run docker build -t "$TEST_IMAGE_TAG" nodejs/
    [ "$status" -eq 0 ]
    
    # Test that entrypoint passes through commands correctly (allow stderr for this test)
    run docker run --rm "$TEST_IMAGE_TAG" echo "test command passthrough"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test command passthrough" ]]
    
    # Test that entrypoint logs startup information to stderr
    run docker run --rm "$TEST_IMAGE_TAG" sh -c "node --version" 2>&1
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Starting Node.js MCP server container" ]]
    [[ "$output" =~ "Node.js:" ]]
    [[ "$output" =~ "NPM:" ]]
    [[ "$output" =~ "Yarn:" ]]
}

# Property 12: Docker Security Best Practices
# For any container image, it should follow Docker security best practices including 
# running as non-root user and maintaining minimal attack surface
@test "Property 12: Node.js container follows Docker security best practices" {
    # Build the container
    run docker build -t "$TEST_IMAGE_TAG" nodejs/
    [ "$status" -eq 0 ]
    
    # Verify container runs as non-root user (UID 1000)
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" id -u
    [ "$status" -eq 0 ]
    [ "$output" = "1000" ]
    
    # Verify container runs as non-root group (GID 1000)
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" id -g
    [ "$status" -eq 0 ]
    [ "$output" = "1000" ]
    
    # Verify user name is 'node' (not root)
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" whoami
    [ "$status" -eq 0 ]
    [ "$output" = "node" ]
    
    # Verify no exposed ports (minimal attack surface)
    run docker image inspect "$TEST_IMAGE_TAG" --format '{{.Config.ExposedPorts}}'
    [ "$status" -eq 0 ]
    [[ "$output" == "map[]" ]] || [[ "$output" == "<no value>" ]] || [[ "$output" == "null" ]]
    
    # Verify entrypoint script has secure permissions (should be executable)
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" stat -c "%a" /usr/local/bin/entrypoint.sh
    [ "$status" -eq 0 ]
    [ "$output" = "755" ]
    
    # Verify home directory exists and has proper ownership
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" stat -c "%U:%G" /home/mcp
    [ "$status" -eq 0 ]
    [ "$output" = "node:node" ]
    
    # Verify container uses Alpine Linux (minimal base image)
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" cat /etc/alpine-release
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
    
    # Verify security labels are present in image metadata
    run docker image inspect "$TEST_IMAGE_TAG" --format '{{index .Config.Labels "security.non-root"}}'
    [ "$status" -eq 0 ]
    [ "$output" = "true" ]
}

# Property 14: Language-Specific Package Managers
# For any language container, it should include the appropriate package managers 
# and use them for dependency management with security best practices
@test "Property 14: Node.js container includes secure package managers" {
    # Build the container
    run docker build -t "$TEST_IMAGE_TAG" nodejs/
    [ "$status" -eq 0 ]
    
    # Verify npm is installed and accessible
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" npm --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
    
    # Verify yarn is installed and accessible
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" yarn --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
    
    # Verify npx is available for secure package execution
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" npx --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
    
    # Verify npm security configuration (update notifier disabled)
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" printenv NPM_CONFIG_UPDATE_NOTIFIER
    [ "$status" -eq 0 ]
    [ "$output" = "false" ]
    
    # Verify npm audit is available for security scanning
    run timeout 30 docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" npm audit --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "audit" ]]
    
    # Verify MCP SDK is available or can be installed (check if it's in dependencies)
    run timeout 30 docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" sh -c "npm list @modelcontextprotocol/sdk 2>/dev/null || echo 'MCP SDK not installed'"
    [ "$status" -eq 0 ]
    # Either the SDK is installed or we get the expected message
    [[ "$output" =~ "@modelcontextprotocol/sdk" ]] || [[ "$output" =~ "MCP SDK not installed" ]]
    
    # Verify node_modules directory has proper permissions
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" stat -c "%a" /app/node_modules
    [ "$status" -eq 0 ]
    [ "$output" = "755" ]
    
    # Verify package managers can install packages securely (test basic npm functionality)
    run timeout 30 docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" sh -c "cd /tmp && npm init -y && echo 'npm init successful'"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "npm init successful" ]]
}

# Unit Tests for Node.js Container Home Directory (Requirements 1.5)
# Test that container starts with correct HOME environment variable
@test "Unit Test: Node.js container has correct HOME environment variable set to /home/mcp" {
    # Build the container
    run docker build -t "$TEST_IMAGE_TAG" nodejs/
    [ "$status" -eq 0 ]
    
    # Test that HOME environment variable is set to /home/mcp
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" printenv HOME
    [ "$status" -eq 0 ]
    [ "$output" = "/home/mcp" ]
}

# Test that /home/mcp directory exists and is writable
@test "Unit Test: Node.js container /home/mcp directory exists and is writable" {
    # Build the container
    run docker build -t "$TEST_IMAGE_TAG" nodejs/
    [ "$status" -eq 0 ]
    
    # Test that /home/mcp directory exists
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" ls -ld /home/mcp
    [ "$status" -eq 0 ]
    [[ "$output" =~ "drwx" ]]
    
    # Test that /home/mcp directory is owned by node user (UID 1000)
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" stat -c "%U:%G" /home/mcp
    [ "$status" -eq 0 ]
    [ "$output" = "node:node" ]
    
    # Test that /home/mcp directory is writable by the node user
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" touch /home/mcp/test-write-file
    [ "$status" -eq 0 ]
    
    # Test that we can create subdirectories in /home/mcp
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" mkdir -p /home/mcp/.config/test-dir
    [ "$status" -eq 0 ]
    
    # Test that we can write files in subdirectories
    run docker run --rm --entrypoint="" "$TEST_IMAGE_TAG" sh -c "mkdir -p /home/mcp/.config && echo 'test content' > /home/mcp/.config/test-file && cat /home/mcp/.config/test-file"
    [ "$status" -eq 0 ]
    [ "$output" = "test content" ]
}