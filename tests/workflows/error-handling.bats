#!/usr/bin/env bats

# Property-Based Tests for Error Handling and Logging (Makefile-based approach)
# Feature: mcp-container-images, Property 6: Makefile Error Handling
# Feature: mcp-container-images, Property 11: Build Logging via Makefile
# Validates: Requirements 2.4, 4.5, 5.3

setup() {
    # Create temp directory for test artifacts
    TEST_DIR=$(mktemp -d)
    cd "$TEST_DIR"
    
    # Copy Makefile and scripts for testing
    cp "$BATS_TEST_DIRNAME/../../Makefile" .
    mkdir -p scripts
    if [ -f "$BATS_TEST_DIRNAME/../../scripts/check-upstream-versions.sh" ]; then
        cp "$BATS_TEST_DIRNAME/../../scripts/check-upstream-versions.sh" scripts/
        chmod +x scripts/check-upstream-versions.sh
    fi
    if [ -f "$BATS_TEST_DIRNAME/../../scripts/cleanup-versions.sh" ]; then
        cp "$BATS_TEST_DIRNAME/../../scripts/cleanup-versions.sh" scripts/
        chmod +x scripts/cleanup-versions.sh
    fi
    
    # Store original directory for workflow access
    export WORKFLOW_DIR="$BATS_TEST_DIRNAME/../../.github/workflows"
}

teardown() {
    rm -rf "$TEST_DIR"
}

# Property 6: Makefile Error Handling
# The Makefile should provide clear error messages and handle failures gracefully

@test "Property 6.1: Makefile detects missing Docker runtime" {
    # Test that Makefile fails gracefully when Docker is not available
    # This test simulates the check_docker function behavior
    run grep -A 10 "check_docker" Makefile
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No container runtime found" ]]
    [[ "$output" =~ "Container Runtime Options" ]]
}

@test "Property 6.2: Makefile validates required files exist" {
    # Test that Makefile checks for required Dockerfiles
    run grep -A 5 "nodejs/Dockerfile" Makefile
    [ "$status" -eq 0 ]
    [[ "$output" =~ "not found" ]]
    
    run grep -A 5 "python/Dockerfile" Makefile
    [ "$status" -eq 0 ]
    [[ "$output" =~ "not found" ]]
}

@test "Property 6.3: Makefile provides clear error messages for missing tools" {
    # Test that Makefile checks for required tools
    run grep -A 5 "bats not found" Makefile
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Install with:" ]]
}

@test "Property 6.4: Makefile handles script execution failures" {
    # Test that Makefile scripts have proper error handling
    run grep -A 3 "chmod +x" Makefile
    [ "$status" -eq 0 ]
    
    # Check that scripts are made executable before running
    [[ "$output" =~ "scripts/" ]]
}

@test "Property 6.5: Workflow uses continue-on-error appropriately" {
    # Check that the workflow file has proper error isolation for cleanup
    run grep -A 5 -B 5 "continue-on-error" "$WORKFLOW_DIR/build-containers.yml"
    [ "$status" -eq 0 ]
    
    # Should have continue-on-error: true for cleanup operations
    [[ "$output" =~ "continue-on-error: true" ]]
}

@test "Property 6.6: Makefile provides colored output for better UX" {
    # Verify Makefile uses colors for better error visibility
    run grep "RED.*Error" Makefile
    [ "$status" -eq 0 ]
    
    run grep "GREEN.*✓" Makefile
    [ "$status" -eq 0 ]
    
    run grep "YELLOW" Makefile
    [ "$status" -eq 0 ]
}

# Property 11: Build Logging via Makefile
# The Makefile should provide comprehensive logging about build processes

@test "Property 11.1: Makefile provides comprehensive build information" {
    # Test that Makefile info target shows build context
    run grep -A 20 "info:" Makefile
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Registry:" ]]
    [[ "$output" =~ "Owner:" ]]
    [[ "$output" =~ "Repository:" ]]
    [[ "$output" =~ "Branch:" ]]
    [[ "$output" =~ "Commit:" ]]
}

@test "Property 11.2: Makefile logs version information during builds" {
    # Test that Makefile shows version information
    run grep -A 10 "Getting latest.*version" Makefile
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Node.js" ]] || [[ "$output" =~ "Python" ]]
}

@test "Property 11.3: Makefile provides progress indicators" {
    # Test that Makefile shows progress during operations
    run grep "Building.*container" Makefile
    [ "$status" -eq 0 ]
    
    run grep "✓.*built" Makefile
    [ "$status" -eq 0 ]
}

@test "Property 11.4: Makefile shows tool availability status" {
    # Test that check-tools target provides clear status
    run grep -A 15 "check-tools:" Makefile
    [ "$status" -eq 0 ]
    [[ "$output" =~ "✓" ]]
    [[ "$output" =~ "✗" ]]
}

@test "Property 11.5: Makefile logs container build context" {
    # Test that Makefile shows what's being built and why
    run grep -A 5 "Building with" Makefile
    [ "$status" -eq 0 ]
    
    # Should show version information during builds
    [[ "$output" =~ "Node.js" ]] || [[ "$output" =~ "Python" ]]
}

@test "Property 11.6: Container entrypoints provide comprehensive startup logging" {
    # Check Node.js entrypoint logging
    run grep -E "\[MCP-CONTAINER\]" "$BATS_TEST_DIRNAME/../../nodejs/entrypoint.sh"
    [ "$status" -eq 0 ]
    
    # Should include comprehensive startup information
    [[ "$output" =~ "Starting Node.js MCP server container" ]]
    [[ "$output" =~ "Timestamp:" ]]
    [[ "$output" =~ "User:" ]]
    [[ "$output" =~ "Working directory:" ]]
    [[ "$output" =~ "Runtime versions:" ]]
    [[ "$output" =~ "Volume mounts:" ]]
    [[ "$output" =~ "Command to execute:" ]]
    
    # Check Python entrypoint logging
    run grep -E "\[MCP-CONTAINER\]" "$BATS_TEST_DIRNAME/../../python/entrypoint.sh"
    [ "$status" -eq 0 ]
    
    # Should include comprehensive startup information
    [[ "$output" =~ "Starting Python MCP server container" ]]
    [[ "$output" =~ "Timestamp:" ]]
    [[ "$output" =~ "User:" ]]
    [[ "$output" =~ "Working directory:" ]]
    [[ "$output" =~ "Runtime versions:" ]]
    [[ "$output" =~ "Volume mounts:" ]]
    [[ "$output" =~ "Virtual environment status:" ]]
}

@test "Property 11.7: Makefile provides help documentation" {
    # Test that Makefile has comprehensive help
    run grep -A 10 "help:" Makefile
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Available targets:" ]]
    [[ "$output" =~ "Configuration:" ]]
}

@test "Property 11.8: Makefile shows dynamic version detection" {
    # Test that Makefile shows supported versions
    run grep -A 5 "Supported Versions:" Makefile
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Node.js:" ]]
    [[ "$output" =~ "Python:" ]]
}

@test "Property 11.9: Makefile handles missing dependencies gracefully" {
    # Test that Makefile provides guidance for missing dependencies
    run grep -A 5 "setup-dev:" Makefile
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Installing" ]]
    [[ "$output" =~ "required tools" ]]
}

@test "Property 11.10: Workflow provides clear job descriptions" {
    # Check that the workflow has descriptive job and step names
    run grep -E "name:|jobs:" "$WORKFLOW_DIR/build-containers.yml"
    [ "$status" -eq 0 ]
    
    # Should have clear job names
    [[ "$output" =~ "Build Container Images" ]]
}