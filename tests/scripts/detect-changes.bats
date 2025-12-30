#!/usr/bin/env bats

# Property-Based Test for Change Detection
# Feature: mcp-container-images, Property 9: Git-Based Change Detection
# Validates: Requirements 4.3

setup() {
    # Create temp git repo for each test
    TEST_DIR=$(mktemp -d)
    cd "$TEST_DIR"
    git init --quiet
    git config user.email "test@test.com"
    git config user.name "Test User"
    
    # Create initial repository structure
    mkdir -p nodejs python .github/workflows scripts
    echo "# Initial Node.js Dockerfile" > nodejs/Dockerfile
    echo "# Initial Python Dockerfile" > python/Dockerfile
    echo "name: build" > .github/workflows/build.yml
    echo "#!/bin/bash" > scripts/build.sh
    echo "# Initial README" > README.md
    
    git add .
    git commit -m "Initial commit" --quiet
    
    # Set up GitHub Actions output file
    export GITHUB_OUTPUT="$TEST_DIR/github_output"
    touch "$GITHUB_OUTPUT"
    
    # Store original directory for script access
    export SCRIPT_DIR="$BATS_TEST_DIRNAME/../../scripts"
}

teardown() {
    rm -rf "$TEST_DIR"
}

# Property Test: Node.js directory changes should trigger only Node.js builds
@test "Property 9.1: Node.js changes trigger only Node.js container build" {
    # Generate random content change in nodejs directory
    echo "# Modified Node.js Dockerfile $(date)" > nodejs/Dockerfile
    git add nodejs/Dockerfile
    git commit -m "Update Node.js container" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=false" "$GITHUB_OUTPUT"
    grep -q "build-reason=nodejs-files-changed" "$GITHUB_OUTPUT"
}

# Property Test: Python directory changes should trigger only Python builds
@test "Property 9.2: Python changes trigger only Python container build" {
    # Generate random content change in python directory
    echo "# Modified Python Dockerfile $(date)" > python/Dockerfile
    git add python/Dockerfile
    git commit -m "Update Python container" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "python-changed=true" "$GITHUB_OUTPUT"
    grep -q "nodejs-changed=false" "$GITHUB_OUTPUT"
    grep -q "build-reason=python-files-changed" "$GITHUB_OUTPUT"
}

# Property Test: Both directory changes should trigger both builds
@test "Property 9.3: Both container changes trigger both builds" {
    # Generate random content changes in both directories
    echo "# Modified Node.js Dockerfile $(date)" > nodejs/Dockerfile
    echo "# Modified Python Dockerfile $(date)" > python/Dockerfile
    git add nodejs/Dockerfile python/Dockerfile
    git commit -m "Update both containers" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=true" "$GITHUB_OUTPUT"
    grep -q "build-reason=nodejs-files-changed,python-files-changed" "$GITHUB_OUTPUT"
}

# Property Test: Common file changes should trigger all builds
@test "Property 9.4: Workflow changes trigger all container builds" {
    # Modify workflow file (common file)
    echo "name: updated-build" > .github/workflows/build.yml
    git add .github/workflows/build.yml
    git commit -m "Update workflow" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=true" "$GITHUB_OUTPUT"
    grep -q "build-reason=common-files-changed" "$GITHUB_OUTPUT"
}

# Property Test: README changes should trigger all builds
@test "Property 9.5: README changes trigger all container builds" {
    # Modify README (common file)
    echo "# Updated README $(date)" > README.md
    git add README.md
    git commit -m "Update README" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=true" "$GITHUB_OUTPUT"
    grep -q "build-reason=common-files-changed" "$GITHUB_OUTPUT"
}

# Property Test: Scripts changes should trigger all builds
@test "Property 9.6: Scripts changes trigger all container builds" {
    # Modify script file (common file)
    echo "#!/bin/bash" > scripts/build.sh
    echo "echo 'Updated build script'" >> scripts/build.sh
    git add scripts/build.sh
    git commit -m "Update build script" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=true" "$GITHUB_OUTPUT"
    grep -q "build-reason=common-files-changed" "$GITHUB_OUTPUT"
}

# Property Test: No relevant changes should skip all builds
@test "Property 9.7: No relevant changes skip all builds" {
    # Make empty commit (no file changes)
    git commit --allow-empty -m "Empty commit" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=false" "$GITHUB_OUTPUT"
    grep -q "python-changed=false" "$GITHUB_OUTPUT"
    grep -q "build-reason=no-changes" "$GITHUB_OUTPUT"
}

# Property Test: Irrelevant file changes should skip all builds
@test "Property 9.8: Irrelevant file changes skip all builds" {
    # Create and modify a file that shouldn't trigger builds
    echo "Some documentation" > DOCS.md
    git add DOCS.md
    git commit -m "Add documentation" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=false" "$GITHUB_OUTPUT"
    grep -q "python-changed=false" "$GITHUB_OUTPUT"
    grep -q "build-reason=no-relevant-changes" "$GITHUB_OUTPUT"
}

# Property Test: First commit scenario (shallow clone simulation)
@test "Property 9.9: First commit triggers all builds (shallow clone fallback)" {
    # Create a new repo with only one commit (simulates shallow clone)
    NEW_TEST_DIR=$(mktemp -d)
    cd "$NEW_TEST_DIR"
    git init --quiet
    git config user.email "test@test.com"
    git config user.name "Test User"
    
    # Create structure and commit
    mkdir -p nodejs python
    echo "# Node.js Dockerfile" > nodejs/Dockerfile
    echo "# Python Dockerfile" > python/Dockerfile
    git add .
    git commit -m "Initial commit" --quiet
    
    export GITHUB_OUTPUT="$NEW_TEST_DIR/github_output"
    touch "$GITHUB_OUTPUT"
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=true" "$GITHUB_OUTPUT"
    grep -q "build-reason=first-commit-or-shallow-clone" "$GITHUB_OUTPUT"
    
    # Cleanup
    rm -rf "$NEW_TEST_DIR"
}

# Property Test: Multiple file changes in same directory
@test "Property 9.10: Multiple Node.js files changed trigger single Node.js build" {
    # Change multiple files in nodejs directory
    echo "# Updated Dockerfile" > nodejs/Dockerfile
    echo '{"name": "test"}' > nodejs/package.json
    echo "#!/bin/sh" > nodejs/entrypoint.sh
    
    git add nodejs/
    git commit -m "Update multiple Node.js files" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=false" "$GITHUB_OUTPUT"
    grep -q "build-reason=nodejs-files-changed" "$GITHUB_OUTPUT"
}

# Property Test: Versions.json changes trigger all builds
@test "Property 9.11: versions.json changes trigger all builds" {
    # Create and modify versions.json (common file)
    echo '{"nodejs": {"22": "22.11.0"}, "python": {"3.12": "3.12.8"}}' > versions.json
    git add versions.json
    git commit -m "Update versions" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=true" "$GITHUB_OUTPUT"
    grep -q "build-reason=common-files-changed" "$GITHUB_OUTPUT"
}

# Property Test: CONTRIBUTING.md changes trigger all builds
@test "Property 9.12: CONTRIBUTING.md changes trigger all builds" {
    # Create and modify CONTRIBUTING.md (common file)
    echo "# Contributing Guidelines" > CONTRIBUTING.md
    git add CONTRIBUTING.md
    git commit -m "Update contributing guidelines" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=true" "$GITHUB_OUTPUT"
    grep -q "build-reason=common-files-changed" "$GITHUB_OUTPUT"
}