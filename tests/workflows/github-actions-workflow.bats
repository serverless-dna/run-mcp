#!/usr/bin/env bats

# Property tests for GitHub Actions workflow (Simplified Makefile-based approach)
# **Feature: mcp-container-images, Property 4: Makefile-based CI/CD**
# **Feature: mcp-container-images, Property 5: Local/CI Parity**
# **Feature: mcp-container-images, Property 7: Simple Workflow Orchestration**
# **Validates: Requirements 2.1, 2.2, 2.5, 4.1**

setup() {
    # Create temporary directory for test workspace
    export TEST_WORKSPACE=$(mktemp -d)
    export ORIGINAL_PWD="$PWD"
    
    # Copy workflow file to test workspace
    mkdir -p "$TEST_WORKSPACE/.github/workflows"
    cp ".github/workflows/build-containers.yml" "$TEST_WORKSPACE/.github/workflows/"
    
    # Copy Makefile to test workspace
    cp "Makefile" "$TEST_WORKSPACE/"
    
    # Create minimal repository structure
    mkdir -p "$TEST_WORKSPACE/nodejs" "$TEST_WORKSPACE/python" "$TEST_WORKSPACE/scripts"
    echo "# Node.js container" > "$TEST_WORKSPACE/nodejs/README.md"
    echo "# Python container" > "$TEST_WORKSPACE/python/README.md"
    
    # Copy scripts that exist
    if [ -f "scripts/check-upstream-versions.sh" ]; then
        cp "scripts/check-upstream-versions.sh" "$TEST_WORKSPACE/scripts/"
        chmod +x "$TEST_WORKSPACE/scripts/check-upstream-versions.sh"
    fi
    
    if [ -f "scripts/cleanup-versions.sh" ]; then
        cp "scripts/cleanup-versions.sh" "$TEST_WORKSPACE/scripts/"
        chmod +x "$TEST_WORKSPACE/scripts/cleanup-versions.sh"
    fi
    
    cd "$TEST_WORKSPACE"
    git init
    git config user.email "test@example.com"
    git config user.name "Test User"
    git add .
    git commit -m "Initial commit"
}

teardown() {
    cd "$ORIGINAL_PWD"
    rm -rf "$TEST_WORKSPACE"
}

# Property 4: Makefile-based CI/CD
# The workflow should delegate heavy lifting to Makefile targets for local/CI parity
@test "Property 4: workflow uses Makefile for CI orchestration" {
    # Verify workflow calls make ci
    grep -q "make ci" .github/workflows/build-containers.yml
}

@test "Property 4: workflow uses Makefile for container builds" {
    # Verify workflow calls make containers or make containers-matrix
    grep -q "make containers" .github/workflows/build-containers.yml
}

@test "Property 4: workflow uses Makefile for cleanup" {
    # Verify workflow calls make cleanup-versions
    grep -q "make cleanup-versions" .github/workflows/build-containers.yml
}

@test "Property 4: workflow supports matrix builds via Makefile" {
    # Verify workflow can call make containers-matrix
    grep -q "containers-matrix" .github/workflows/build-containers.yml
}

# Property 5: Local/CI Parity
# The same commands should work locally and in CI
@test "Property 5: workflow triggers on relevant path changes" {
    # Verify workflow triggers on container and script changes
    grep -A 10 "paths:" .github/workflows/build-containers.yml | grep -q "nodejs/"
    grep -A 10 "paths:" .github/workflows/build-containers.yml | grep -q "python/"
    grep -A 10 "paths:" .github/workflows/build-containers.yml | grep -q "scripts/"
}

@test "Property 5: workflow supports manual dispatch" {
    # Verify workflow_dispatch trigger exists
    grep -q "workflow_dispatch:" .github/workflows/build-containers.yml
}

@test "Property 5: workflow has proper permissions for registry" {
    # Verify permissions are set for package publishing
    grep -A 5 "permissions:" .github/workflows/build-containers.yml | grep -q "packages: write"
    grep -A 5 "permissions:" .github/workflows/build-containers.yml | grep -q "contents: read"
}

@test "Property 5: workflow authenticates with GitHub Container Registry" {
    # Verify GHCR registry configuration
    grep -q "REGISTRY: ghcr.io" .github/workflows/build-containers.yml
    
    # Verify login action exists
    grep -A 5 "Log in to Container Registry" .github/workflows/build-containers.yml | grep -q "docker/login-action"
    
    # Verify GITHUB_TOKEN is used
    grep -q "GITHUB_TOKEN" .github/workflows/build-containers.yml
}

# Property 7: Simple Workflow Orchestration
# The workflow should be simple and delegate complexity to Makefile
@test "Property 7: workflow has minimal job structure" {
    # Verify main jobs exist but are simple
    grep -q "ci:" .github/workflows/build-containers.yml
    grep -q "build:" .github/workflows/build-containers.yml
    grep -q "cleanup-old-versions:" .github/workflows/build-containers.yml
}

@test "Property 7: workflow uses Docker Buildx for multi-platform" {
    # Verify Docker Buildx setup for multi-platform builds
    grep -q "docker/setup-buildx-action" .github/workflows/build-containers.yml
    grep -q "linux/amd64,linux/arm64" .github/workflows/build-containers.yml
}

@test "Property 7: workflow includes binary build job" {
    # Verify Go binary build job exists
    grep -q "build-binary:" .github/workflows/build-containers.yml
    grep -q "make build-run-mcp-all" .github/workflows/build-containers.yml
}

@test "Property 7: workflow has proper job dependencies" {
    # Verify job dependencies are logical
    grep -A 3 "needs: ci" .github/workflows/build-containers.yml
    grep -A 3 "needs: \[ci, cleanup-old-versions\]" .github/workflows/build-containers.yml
}

# Additional workflow validation tests
@test "workflow uses correct checkout action version" {
    # Verify modern checkout action
    grep -q "actions/checkout@v4" .github/workflows/build-containers.yml
}

@test "workflow uses correct Go setup action version" {
    # Verify modern Go setup action
    grep -q "actions/setup-go@v5" .github/workflows/build-containers.yml
}

@test "workflow includes artifact upload for binaries" {
    # Verify binary artifacts are uploaded
    grep -q "actions/upload-artifact@v4" .github/workflows/build-containers.yml
    grep -q "run-mcp-binaries" .github/workflows/build-containers.yml
}

@test "workflow supports conditional matrix builds" {
    # Verify conditional logic for matrix builds
    grep -q "build_matrix" .github/workflows/build-containers.yml
    grep -q "force_build" .github/workflows/build-containers.yml
}

@test "workflow includes proper error handling" {
    # Verify continue-on-error is used appropriately
    grep -q "continue-on-error: true" .github/workflows/build-containers.yml
}

@test "workflow runs on Ubuntu latest" {
    # Verify consistent runner environment
    grep -q "runs-on: ubuntu-latest" .github/workflows/build-containers.yml
}

@test "workflow has proper fetch depth for git operations" {
    # Verify fetch-depth is set for git operations
    grep -A 3 "actions/checkout@v4" .github/workflows/build-containers.yml | grep -q "fetch-depth: 0"
}

@test "workflow includes GitHub CLI installation for cleanup" {
    # Verify GitHub CLI is installed for package cleanup
    grep -q "Install GitHub CLI" .github/workflows/build-containers.yml
    grep -q "gh auth login" .github/workflows/build-containers.yml
}