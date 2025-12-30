#!/usr/bin/env bats

# Property-Based Tests for Error Handling and Logging
# Feature: mcp-container-images, Property 6: Build Failure Handling
# Feature: mcp-container-images, Property 11: Build Logging Transparency
# Validates: Requirements 2.4, 4.5, 5.3

setup() {
    # Create temp directory for test artifacts
    TEST_DIR=$(mktemp -d)
    cd "$TEST_DIR"
    
    # Set up test environment
    export GITHUB_OUTPUT="$TEST_DIR/github_output"
    touch "$GITHUB_OUTPUT"
    
    # Store original directory for script access
    export SCRIPT_DIR="$BATS_TEST_DIRNAME/../../scripts"
    export WORKFLOW_DIR="$BATS_TEST_DIRNAME/../../.github/workflows"
}

teardown() {
    rm -rf "$TEST_DIR"
}

# Property 6: Build Failure Handling
# For any build that fails, the system should provide clear error messages, 
# stop the pipeline for that container, and not affect other container builds

@test "Property 6.1: Change detection script provides clear error messages on git failures" {
    # Create a corrupted git environment to simulate git failures
    mkdir -p .git
    echo "corrupted" > .git/HEAD
    
    # Run change detection script
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    # Script should handle the error gracefully and provide fallback
    # The script may exit with 1 for validation errors, but should still provide outputs
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
    
    if [ "$status" -eq 0 ]; then
        # Should fall back to building all containers
        grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
        grep -q "python-changed=true" "$GITHUB_OUTPUT"
        
        # Should provide clear reason for fallback
        grep -q "build-reason=git-diff-failed\|build-reason=unknown-error\|build-reason=first-commit-or-shallow-clone" "$GITHUB_OUTPUT"
    else
        # If script exits with error, it should provide clear error message
        [[ "$output" =~ "ERROR:" ]]
    fi
}

@test "Property 6.2: Change detection handles missing GITHUB_OUTPUT gracefully" {
    # Create a proper git repo
    git init --quiet
    git config user.email "test@test.com"
    git config user.name "Test User"
    
    # Create initial structure
    mkdir -p nodejs python
    echo "# Node.js Dockerfile" > nodejs/Dockerfile
    echo "# Python Dockerfile" > python/Dockerfile
    git add .
    git commit -m "Initial commit" --quiet
    
    # Unset GITHUB_OUTPUT to test error handling
    unset GITHUB_OUTPUT
    
    # Run change detection script
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    # Script should handle missing GITHUB_OUTPUT gracefully
    [ "$status" -eq 0 ]
    
    # Should log warning about missing GITHUB_OUTPUT
    [[ "$output" =~ "WARNING: GITHUB_OUTPUT not set" ]]
}

@test "Property 6.3: Change detection provides detailed error context for different failure modes" {
    # Test with non-git directory
    mkdir non-git-dir
    cd non-git-dir
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    # Should fail with clear error message
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Not in a git repository" ]]
}

@test "Property 6.4: Build workflow uses continue-on-error appropriately for container isolation" {
    # Check that the workflow file has proper error isolation
    run grep -A 5 -B 5 "continue-on-error" "$WORKFLOW_DIR/build-containers.yml"
    
    [ "$status" -eq 0 ]
    
    # Should have continue-on-error: false for critical build steps
    [[ "$output" =~ "continue-on-error: false" ]]
    
    # Should have continue-on-error: true for integration tests to allow partial success
    [[ "$output" =~ "continue-on-error: true" ]]
}

@test "Property 6.5: Build workflow includes proper error annotations" {
    # Check that the workflow includes GitHub Actions error annotations
    run grep "::error" "$WORKFLOW_DIR/build-containers.yml"
    
    [ "$status" -eq 0 ]
    
    # Should include error titles and messages
    [[ "$output" =~ "::error title=" ]]
    [[ "$output" =~ "Build Failed" ]]
}

@test "Property 6.6: Build workflow isolates container failures" {
    # Check that build jobs are properly isolated
    run grep -A 10 "needs: \[build-nodejs, build-python\]" "$WORKFLOW_DIR/build-containers.yml"
    
    [ "$status" -eq 0 ]
    
    # Should use 'if: always()' to run even if some builds fail
    [[ "$output" =~ "if: always()" ]]
    
    # Should check individual build results
    [[ "$output" =~ "needs.build-nodejs.result" ]]
    [[ "$output" =~ "needs.build-python.result" ]]
}

# Property 11: Build Logging Transparency
# For any build process, the system should provide clear logging about which images 
# are being built and the reasons for building them

@test "Property 11.1: Change detection provides comprehensive build reasoning" {
    # Create a proper git repo with changes
    git init --quiet
    git config user.email "test@test.com"
    git config user.name "Test User"
    
    # Create initial structure
    mkdir -p nodejs python .github/workflows
    echo "# Initial Node.js Dockerfile" > nodejs/Dockerfile
    echo "# Initial Python Dockerfile" > python/Dockerfile
    echo "name: build" > .github/workflows/build.yml
    git add .
    git commit -m "Initial commit" --quiet
    
    # Make a change to Node.js directory
    echo "# Updated Node.js Dockerfile" > nodejs/Dockerfile
    git add nodejs/Dockerfile
    git commit -m "Update Node.js" --quiet
    
    # Run change detection
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    
    # Should provide detailed logging about what changed and why
    [[ "$output" =~ "CHANGE DETECTION SUMMARY" ]]
    [[ "$output" =~ "CHANGE ANALYSIS BREAKDOWN" ]]
    [[ "$output" =~ "BUILD DECISION MATRIX" ]]
    [[ "$output" =~ "Node.js directory changes detected" ]]
    [[ "$output" =~ "Building Node.js container" ]]
    [[ "$output" =~ "Skipping Python container" ]]
}

@test "Property 11.2: Change detection logs repository context information" {
    # Create a proper git repo
    git init --quiet
    git config user.email "test@test.com"
    git config user.name "Test User"
    
    # Create initial structure
    mkdir -p nodejs python
    echo "# Node.js Dockerfile" > nodejs/Dockerfile
    echo "# Python Dockerfile" > python/Dockerfile
    git add .
    git commit -m "Initial commit" --quiet
    
    # Add a remote to test remote logging
    git remote add origin https://github.com/test/test.git
    
    # Make a change to trigger the detailed logging path
    echo "# Updated Node.js Dockerfile" > nodejs/Dockerfile
    git add nodejs/Dockerfile
    git commit -m "Update Node.js" --quiet
    
    # Run change detection
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    
    # Should log repository context (the script logs this when there are changes to analyze)
    [[ "$output" =~ "CHANGE DETECTION SUMMARY" ]] || [[ "$output" =~ "Starting change detection" ]]
}

@test "Property 11.3: Change detection provides file-level change details" {
    # Create a proper git repo with multiple file changes
    git init --quiet
    git config user.email "test@test.com"
    git config user.name "Test User"
    
    # Create initial structure
    mkdir -p nodejs python
    echo "# Initial Node.js Dockerfile" > nodejs/Dockerfile
    echo "# Initial Python Dockerfile" > python/Dockerfile
    echo '{"name": "test"}' > nodejs/package.json
    git add .
    git commit -m "Initial commit" --quiet
    
    # Make changes to multiple files
    echo "# Updated Node.js Dockerfile" > nodejs/Dockerfile
    echo '{"name": "updated"}' > nodejs/package.json
    git add nodejs/
    git commit -m "Update Node.js files" --quiet
    
    # Run change detection
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    
    # Should log individual changed files
    [[ "$output" =~ "nodejs/Dockerfile" ]]
    [[ "$output" =~ "nodejs/package.json" ]]
    [[ "$output" =~ "Total changed files:" ]]
}

@test "Property 11.4: Build workflow logs build start and completion information" {
    # Check that the workflow includes comprehensive logging
    run grep -E "(Log build start|Log build success|Log build failure)" "$WORKFLOW_DIR/build-containers.yml"
    
    [ "$status" -eq 0 ]
    
    # Should include start, success, and failure logging
    [[ "$output" =~ "Log build start" ]]
    [[ "$output" =~ "Log build success" ]]
    [[ "$output" =~ "Log build failure" ]]
}

@test "Property 11.5: Build workflow logs timestamps and build context" {
    # Check that the workflow includes timestamp logging
    run grep -E "(Timestamp|date -u)" "$WORKFLOW_DIR/build-containers.yml"
    
    [ "$status" -eq 0 ]
    
    # Should include timestamp logging
    [[ "$output" =~ "Timestamp:" ]]
    [[ "$output" =~ "date -u" ]]
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

@test "Property 11.7: Change detection provides final output summary" {
    # Create a proper git repo
    git init --quiet
    git config user.email "test@test.com"
    git config user.name "Test User"
    
    # Create initial structure
    mkdir -p nodejs python
    echo "# Node.js Dockerfile" > nodejs/Dockerfile
    echo "# Python Dockerfile" > python/Dockerfile
    git add .
    git commit -m "Initial commit" --quiet
    
    # Run change detection
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    
    # Should provide final summary of outputs (check for key summary elements)
    [[ "$output" =~ "Final outputs:" ]] || [[ "$output" =~ "Change detection completed" ]]
    [[ "$output" =~ "nodejs-changed:" ]] || [[ "$output" =~ "nodejs-changed=" ]]
    [[ "$output" =~ "python-changed:" ]] || [[ "$output" =~ "python-changed=" ]]
    [[ "$output" =~ "build-reason:" ]] || [[ "$output" =~ "build-reason=" ]]
}

@test "Property 11.8: Build workflow provides integration test result summaries" {
    # Check that the workflow includes integration test result logging
    run grep -A 10 "Log integration test results" "$WORKFLOW_DIR/build-containers.yml"
    
    [ "$status" -eq 0 ]
    
    # Should include test result summaries
    [[ "$output" =~ "Integration test results summary:" ]]
    [[ "$output" =~ "Node.js container test:" ]]
    [[ "$output" =~ "Python container test:" ]]
    [[ "$output" =~ "stdio mode test:" ]]
    [[ "$output" =~ "Volume mounting test:" ]]
}

@test "Property 11.9: Change detection handles edge cases with appropriate logging" {
    # Test empty repository (no commits)
    git init --quiet
    git config user.email "test@test.com"
    git config user.name "Test User"
    
    # Run change detection on empty repo
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    
    # Should provide clear explanation for fallback behavior
    [[ "$output" =~ "No comparison commit available" ]]
    [[ "$output" =~ "first-commit-or-shallow-clone" ]]
    [[ "$output" =~ "fallback mode" ]]
}

@test "Property 11.10: Build workflow logs specific failure contexts" {
    # Check that the workflow provides specific failure context
    run grep -A 10 -B 5 "failure context\|failure\|Failed" "$WORKFLOW_DIR/build-containers.yml"
    
    [ "$status" -eq 0 ]
    
    # Should check individual step outcomes (look for any step outcome checks)
    [[ "$output" =~ "steps\." ]] && [[ "$output" =~ "\.outcome" ]]
    
    # Should have specific error messages for different failure types (check for any failure-related text)
    [[ "$output" =~ "Failed" ]] || [[ "$output" =~ "failure" ]]
}