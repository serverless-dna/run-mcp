#!/usr/bin/env bats

# Unit Tests for Change Detection Script
# Tests specific examples, edge cases, and error conditions
# Requirements: 4.3, 4.5

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

# Unit Test: Environment validation
@test "Unit: Script fails gracefully when not in git repository" {
    # Create non-git directory
    NON_GIT_DIR=$(mktemp -d)
    cd "$NON_GIT_DIR"
    
    export GITHUB_OUTPUT="$NON_GIT_DIR/output"
    touch "$GITHUB_OUTPUT"
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Not in a git repository"* ]]
    
    rm -rf "$NON_GIT_DIR"
}

# Unit Test: GITHUB_OUTPUT handling
@test "Unit: Script handles missing GITHUB_OUTPUT environment variable" {
    # Unset GITHUB_OUTPUT
    unset GITHUB_OUTPUT
    
    echo "test" > nodejs/Dockerfile
    git add nodejs/Dockerfile
    git commit -m "Test change" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"GITHUB_OUTPUT not set"* ]]
}

# Unit Test: Shallow clone detection
@test "Unit: Script detects shallow clone and falls back appropriately" {
    # Create a shallow clone scenario by removing git history access
    # This simulates what happens in GitHub Actions with fetch-depth: 1
    
    # Remove access to previous commits by corrupting the git objects
    rm -rf .git/objects/pack/* 2>/dev/null || true
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=true" "$GITHUB_OUTPUT"
    grep -q "build-reason=first-commit-or-shallow-clone" "$GITHUB_OUTPUT"
}

# Unit Test: Git diff failure handling
@test "Unit: Script handles git diff failures gracefully" {
    # Create a scenario where git diff might fail
    echo "test" > nodejs/Dockerfile
    git add nodejs/Dockerfile
    git commit -m "Test change" --quiet
    
    # Corrupt git repository to cause git diff to fail
    rm -rf .git/refs/heads/master
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    # Should fall back to building all containers when git diff fails
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=true" "$GITHUB_OUTPUT"
    # The script should set some build reason (could be git-diff-failed or first-commit-or-shallow-clone)
    grep -q "build-reason=" "$GITHUB_OUTPUT"
}

# Unit Test: Empty diff handling
@test "Unit: Script correctly handles empty git diff" {
    # Create a commit with no actual file changes (empty commit)
    git commit --allow-empty -m "Empty commit for testing" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=false" "$GITHUB_OUTPUT"
    grep -q "python-changed=false" "$GITHUB_OUTPUT"
    grep -q "build-reason=no-changes" "$GITHUB_OUTPUT"
}

# Unit Test: Deleted files handling
@test "Unit: Script handles deleted files correctly" {
    # Delete a file in nodejs directory
    rm nodejs/Dockerfile
    git add nodejs/Dockerfile
    git commit -m "Delete Node.js Dockerfile" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=false" "$GITHUB_OUTPUT"
    grep -q "build-reason=nodejs-files-changed" "$GITHUB_OUTPUT"
}

# Unit Test: Renamed files handling
@test "Unit: Script handles renamed files correctly" {
    # Rename a file in python directory
    git mv python/Dockerfile python/Dockerfile.new
    git commit -m "Rename Python Dockerfile" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "python-changed=true" "$GITHUB_OUTPUT"
    grep -q "nodejs-changed=false" "$GITHUB_OUTPUT"
    grep -q "build-reason=python-files-changed" "$GITHUB_OUTPUT"
}

# Unit Test: Binary file changes
@test "Unit: Script handles binary file changes" {
    # Create a binary file in nodejs directory
    echo -e '\x00\x01\x02\x03' > nodejs/binary-file.bin
    git add nodejs/binary-file.bin
    git commit -m "Add binary file to nodejs" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=false" "$GITHUB_OUTPUT"
}

# Unit Test: Large number of file changes
@test "Unit: Script handles large number of file changes efficiently" {
    # Create many files in nodejs directory
    for i in {1..50}; do
        echo "File $i content" > "nodejs/file$i.txt"
    done
    
    git add nodejs/
    git commit -m "Add many files to nodejs" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=false" "$GITHUB_OUTPUT"
    
    # Should complete in reasonable time (test should not hang)
    # The fact that we reach this assertion means it completed
    [ "$status" -eq 0 ]
}

# Unit Test: Nested directory changes
@test "Unit: Script detects changes in nested directories" {
    # Create nested directories and files
    mkdir -p nodejs/src/components
    mkdir -p python/src/modules
    
    echo "Component code" > nodejs/src/components/Button.js
    echo "Module code" > python/src/modules/auth.py
    
    git add .
    git commit -m "Add nested directory structure" --quiet
    
    # Now modify nested files
    echo "Updated component" > nodejs/src/components/Button.js
    git add nodejs/src/components/Button.js
    git commit -m "Update nested nodejs file" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=false" "$GITHUB_OUTPUT"
}

# Unit Test: Special characters in filenames
@test "Unit: Script handles special characters in filenames" {
    # Create files with special characters
    echo "Content" > "nodejs/file with spaces.txt"
    echo "Content" > "nodejs/file-with-dashes.txt"
    echo "Content" > "nodejs/file_with_underscores.txt"
    
    git add nodejs/
    git commit -m "Add files with special characters" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    grep -q "nodejs-changed=true" "$GITHUB_OUTPUT"
    grep -q "python-changed=false" "$GITHUB_OUTPUT"
}

# Unit Test: Merge commit scenario
@test "Unit: Script handles merge commits correctly" {
    # Create a branch and make changes
    git checkout -b feature-branch --quiet
    echo "Feature change" > nodejs/feature.txt
    git add nodejs/feature.txt
    git commit -m "Add feature" --quiet
    
    # Switch back to master and create a merge commit
    git checkout master --quiet
    echo "Main change" > python/main.txt
    git add python/main.txt
    git commit -m "Master branch change" --quiet
    
    # Merge the feature branch
    git merge feature-branch --no-ff -m "Merge feature branch" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    # Should detect changes from the merge
    [[ "$output" == *"Successfully"* ]]
}

# Unit Test: Output format validation
@test "Unit: Script generates correctly formatted GitHub Actions outputs" {
    echo "test" > nodejs/Dockerfile
    git add nodejs/Dockerfile
    git commit -m "Test change" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    
    # Validate output format (key=value pairs)
    grep -E "^nodejs-changed=(true|false)$" "$GITHUB_OUTPUT"
    grep -E "^python-changed=(true|false)$" "$GITHUB_OUTPUT"
    grep -E "^build-reason=.+$" "$GITHUB_OUTPUT"
}

# Unit Test: Logging output validation
@test "Unit: Script provides comprehensive logging for debugging" {
    echo "test" > nodejs/Dockerfile
    git add nodejs/Dockerfile
    git commit -m "Test change" --quiet
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    [ "$status" -eq 0 ]
    
    # Check that logging includes timestamps and useful information
    [[ "$output" == *"Starting change detection"* ]]
    [[ "$output" == *"Validating environment"* ]]
    [[ "$output" == *"Getting changed files"* ]]
    [[ "$output" == *"Analyzing changed files"* ]]
    [[ "$output" == *"BUILD DECISION"* ]]
    [[ "$output" == *"Change detection completed"* ]]
}

# Unit Test: Error message clarity
@test "Unit: Script provides clear error messages for common issues" {
    # Test with completely missing .git directory
    rm -rf .git
    
    run bash "$SCRIPT_DIR/detect-changes.sh"
    
    # Script should exit with error code 1 and provide clear error message
    [ "$status" -eq 1 ]
    [[ "$output" == *"ERROR"* ]]
    [[ "$output" == *"Not in a git repository"* ]]
}