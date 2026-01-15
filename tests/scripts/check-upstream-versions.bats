#!/usr/bin/env bats

# Property-Based Test for Upstream Version Detection
# Feature: mcp-container-images, Property 18: Upstream Version Detection
# Validates: Requirements 12.1, 12.2, 12.3, 12.4, 12.6

setup() {
    # Create temp directory for each test
    TEST_DIR=$(mktemp -d)
    cd "$TEST_DIR"
    
    # Set up GitHub Actions output file
    export GITHUB_OUTPUT="$TEST_DIR/github_output"
    touch "$GITHUB_OUTPUT"
    
    # Store original directory for script access
    export SCRIPT_DIR="$BATS_TEST_DIRNAME/../../scripts"
    export FIXTURES_DIR="$BATS_TEST_DIRNAME/../fixtures"
    
    # Mock curl and jq commands for testing
    export PATH="$TEST_DIR/mocks:$PATH"
    mkdir -p "$TEST_DIR/mocks"
    
    # Create mock jq that uses real jq
    cat > "$TEST_DIR/mocks/jq" << 'EOF'
#!/bin/bash
exec /usr/bin/jq "$@"
EOF
    chmod +x "$TEST_DIR/mocks/jq"
    
    # Mock git commands for consistent test environment
    cat > "$TEST_DIR/mocks/git" << 'EOF'
#!/bin/bash
case "$*" in
    "config --get remote.origin.url")
        echo "https://github.com/test-owner/test-repo.git"
        ;;
    "rev-parse --show-toplevel")
        echo "/tmp/test-repo"
        ;;
    *)
        echo "Mock git: Unknown command $*" >&2
        exit 1
        ;;
esac
EOF
    chmod +x "$TEST_DIR/mocks/git"
}

teardown() {
    rm -rf "$TEST_DIR"
}

# Helper function to create mock curl responses for Node.js and Python APIs
create_mock_curl() {
    local nodejs_response="$1"
    local python_response="$2"
    
    cat > "$TEST_DIR/mocks/curl" << EOF
#!/bin/bash
if [[ "\$*" == *"nodejs.org"* ]]; then
    echo '$nodejs_response'
elif [[ "\$*" == *"endoflife.date"* ]]; then
    echo '$python_response'
elif [[ "\$*" == *"api.github.com"* ]]; then
    # Mock GitHub API response (empty - no published packages)
    echo '[]'
else
    echo "Mock curl: Unknown URL \$*" >&2
    exit 1
fi
EOF
    chmod +x "$TEST_DIR/mocks/curl"
}

# Helper function to run script and extract JSON output
run_script_and_get_json() {
    # Run the script and capture stdout (JSON) separately from stderr (logs)
    bash "$SCRIPT_DIR/check-upstream-versions.sh" 2>/dev/null
}

# Property Test: Current versions should not trigger updates when already built
@test "Property 18.1: Current versions detected as no updates needed" {
    # Mock GitHub API to return existing published versions (simulating already built)
    cat > "$TEST_DIR/mocks/curl" << 'EOF'
#!/bin/bash
if [[ "$*" == *"nodejs.org"* ]]; then
    echo '[{"version": "v22.11.0", "lts": "Hydrogen"}, {"version": "v20.18.1", "lts": "Iron"}]'
elif [[ "$*" == *"endoflife.date"* ]]; then
    echo '[{"cycle": "3.12", "latest": "3.12.8", "releaseDate": "2023-10-02"}, {"cycle": "3.11", "latest": "3.11.11", "releaseDate": "2022-10-24"}]'
elif [[ "$*" == *"api.github.com"* && "$*" == *"nodejs"* ]]; then
    echo '[{"metadata": {"container": {"tags": ["node22.11.0", "node20.18.1"]}}}]'
elif [[ "$*" == *"api.github.com"* && "$*" == *"python"* ]]; then
    echo '[{"metadata": {"container": {"tags": ["python3.12.8", "python3.11.11"]}}}]'
else
    echo "Mock curl: Unknown URL $*" >&2
    exit 1
fi
EOF
    chmod +x "$TEST_DIR/mocks/curl"
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    [ "$status" -eq 0 ]
    
    # Get JSON output
    local json_output
    json_output=$(run_script_and_get_json)
    
    # Check that no updates are detected
    local updates_detected
    updates_detected=$(echo "$json_output" | jq -r '.summary.updates_detected')
    [[ "$updates_detected" == "false" ]]
    
    # Check GitHub Actions outputs
    grep -q "updates-detected=false" "$GITHUB_OUTPUT"
}

# Property Test: Outdated versions should trigger updates
@test "Property 18.2: Outdated versions detected as updates needed" {
    # Mock API responses with newer versions available
    create_mock_curl \
        '[{"version": "v22.12.0", "lts": "Hydrogen"}, {"version": "v20.19.0", "lts": "Iron"}]' \
        '[{"cycle": "3.12", "latest": "3.12.9", "releaseDate": "2023-10-02"}, {"cycle": "3.11", "latest": "3.11.12", "releaseDate": "2022-10-24"}]'
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    [ "$status" -eq 0 ]
    
    # Get JSON output
    local json_output
    json_output=$(run_script_and_get_json)
    
    # Check that updates are detected
    local updates_detected
    updates_detected=$(echo "$json_output" | jq -r '.summary.updates_detected')
    [[ "$updates_detected" == "true" ]]
    
    # Check that versions to build are populated
    local nodejs_updates python_updates
    nodejs_updates=$(echo "$json_output" | jq -r '.nodejs.versions_to_build | length')
    python_updates=$(echo "$json_output" | jq -r '.python.versions_to_build | length')
    
    [[ "$nodejs_updates" -gt 0 ]]
    [[ "$python_updates" -gt 0 ]]
    
    # Check GitHub Actions outputs
    grep -q "updates-detected=true" "$GITHUB_OUTPUT"
}

# Property Test: API failure handling for Node.js
@test "Property 18.3: Node.js API failure handled gracefully" {
    # Create mock curl that fails for Node.js API
    cat > "$TEST_DIR/mocks/curl" << 'EOF'
#!/bin/bash
if [[ "$*" == *"nodejs.org"* ]]; then
    echo "API Error" >&2
    exit 1
elif [[ "$*" == *"endoflife.date"* ]]; then
    echo '[{"cycle": "3.12", "latest": "3.12.8", "releaseDate": "2023-10-02"}, {"cycle": "3.11", "latest": "3.11.11", "releaseDate": "2022-10-24"}]'
elif [[ "$*" == *"api.github.com"* ]]; then
    echo '[]'
else
    echo "Mock curl: Unknown URL $*" >&2
    exit 1
fi
EOF
    chmod +x "$TEST_DIR/mocks/curl"
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    [ "$status" -eq 0 ]
    
    # Get JSON output
    local json_output
    json_output=$(run_script_and_get_json)
    
    # Should still have Python versions but no Node.js versions
    local python_supported nodejs_supported
    python_supported=$(echo "$json_output" | jq -r '.python.supported_versions | length')
    nodejs_supported=$(echo "$json_output" | jq -r '.nodejs.supported_versions | length')
    
    [[ "$python_supported" -gt 0 ]]
    [[ "$nodejs_supported" -eq 0 ]]
}

# Property Test: API failure handling for Python
@test "Property 18.4: Python API failure handled gracefully" {
    # Create mock curl that fails for Python API
    cat > "$TEST_DIR/mocks/curl" << 'EOF'
#!/bin/bash
if [[ "$*" == *"nodejs.org"* ]]; then
    echo '[{"version": "v22.11.0", "lts": "Hydrogen"}, {"version": "v20.18.1", "lts": "Iron"}]'
elif [[ "$*" == *"endoflife.date"* ]]; then
    echo "API Error" >&2
    exit 1
elif [[ "$*" == *"api.github.com"* ]]; then
    echo '[]'
else
    echo "Mock curl: Unknown URL $*" >&2
    exit 1
fi
EOF
    chmod +x "$TEST_DIR/mocks/curl"
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    [ "$status" -eq 0 ]
    
    # Get JSON output
    local json_output
    json_output=$(run_script_and_get_json)
    
    # Should still have Node.js versions but no Python versions
    local nodejs_supported python_supported
    nodejs_supported=$(echo "$json_output" | jq -r '.nodejs.supported_versions | length')
    python_supported=$(echo "$json_output" | jq -r '.python.supported_versions | length')
    
    [[ "$nodejs_supported" -gt 0 ]]
    [[ "$python_supported" -eq 0 ]]
}

# Property Test: Invalid JSON response handling
@test "Property 18.5: Invalid JSON responses handled gracefully" {
    # Mock API responses with invalid JSON
    create_mock_curl \
        'invalid json response' \
        'also invalid json'
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    [ "$status" -eq 0 ]
    
    # Get JSON output - should still produce valid JSON even with API failures
    local json_output
    json_output=$(run_script_and_get_json)
    
    # Should be valid JSON
    echo "$json_output" | jq . >/dev/null
    
    # Should have empty supported versions due to API failures
    local nodejs_supported python_supported
    nodejs_supported=$(echo "$json_output" | jq -r '.nodejs.supported_versions | length')
    python_supported=$(echo "$json_output" | jq -r '.python.supported_versions | length')
    
    [[ "$nodejs_supported" -eq 0 ]]
    [[ "$python_supported" -eq 0 ]]
}

# Property Test: Script execution without external dependencies
@test "Property 18.6: Script handles missing external APIs gracefully" {
    # Create mock curl that always fails
    cat > "$TEST_DIR/mocks/curl" << 'EOF'
#!/bin/bash
echo "Network error" >&2
exit 1
EOF
    chmod +x "$TEST_DIR/mocks/curl"
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    [ "$status" -eq 0 ]
    
    # Should produce valid JSON output
    local json_output
    json_output=$(run_script_and_get_json)
    echo "$json_output" | jq . >/dev/null
}

# Property Test: Empty API responses
@test "Property 18.7: Empty API responses handled gracefully" {
    # Mock empty API responses
    create_mock_curl '[]' '[]'
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    [ "$status" -eq 0 ]
    
    # Get JSON output
    local json_output
    json_output=$(run_script_and_get_json)
    
    # Should produce valid JSON
    echo "$json_output" | jq . >/dev/null
    
    # Should have empty supported versions due to empty API responses
    local nodejs_supported python_supported
    nodejs_supported=$(echo "$json_output" | jq -r '.nodejs.supported_versions | length')
    python_supported=$(echo "$json_output" | jq -r '.python.supported_versions | length')
    
    [[ "$nodejs_supported" -eq 0 ]]
    [[ "$python_supported" -eq 0 ]]
}

# Property Test: Dynamic version detection functionality
@test "Property 18.8: Dynamic version detection works correctly" {
    # Mock realistic API responses
    create_mock_curl \
        '[{"version": "v24.1.0", "lts": false}, {"version": "v22.11.0", "lts": "Hydrogen"}, {"version": "v20.18.1", "lts": "Iron"}, {"version": "v18.19.0", "lts": "Hydrogen"}]' \
        '[{"cycle": "3.13", "latest": "3.13.1", "releaseDate": "2024-10-07"}, {"cycle": "3.12", "latest": "3.12.8", "releaseDate": "2023-10-02"}, {"cycle": "3.11", "latest": "3.11.11", "releaseDate": "2022-10-24"}]'
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    [ "$status" -eq 0 ]
    
    # Get JSON output
    local json_output
    json_output=$(run_script_and_get_json)
    
    # Should detect supported versions dynamically
    local nodejs_supported python_supported
    nodejs_supported=$(echo "$json_output" | jq -r '.nodejs.supported_versions | length')
    python_supported=$(echo "$json_output" | jq -r '.python.supported_versions | length')
    
    # Should support multiple versions (Current + Previous LTS)
    [[ "$nodejs_supported" -ge 2 ]]
    [[ "$python_supported" -ge 2 ]]
    
    # Should have latest versions populated
    local nodejs_latest_count python_latest_count
    nodejs_latest_count=$(echo "$json_output" | jq -r '.nodejs.latest_versions | keys | length')
    python_latest_count=$(echo "$json_output" | jq -r '.python.latest_versions | keys | length')
    
    [[ "$nodejs_latest_count" -eq "$nodejs_supported" ]]
    [[ "$python_latest_count" -eq "$python_supported" ]]
}

# Property Test: Comprehensive integration test
@test "Property 18.9: Complete workflow with realistic scenarios" {
    # Mock GitHub API to show some versions built, some not
    cat > "$TEST_DIR/mocks/curl" << 'EOF'
#!/bin/bash
if [[ "$*" == *"nodejs.org"* ]]; then
    echo '[{"version": "v22.12.0", "lts": "Hydrogen"}, {"version": "v20.19.0", "lts": "Iron"}]'
elif [[ "$*" == *"endoflife.date"* ]]; then
    echo '[{"cycle": "3.12", "latest": "3.12.9", "releaseDate": "2023-10-02"}, {"cycle": "3.11", "latest": "3.11.12", "releaseDate": "2022-10-24"}]'
elif [[ "$*" == *"api.github.com"* && "$*" == *"nodejs"* ]]; then
    # Some Node.js versions built with older versions
    echo '[{"metadata": {"container": {"tags": ["node22.11.0", "node20.18.0"]}}}]'
elif [[ "$*" == *"api.github.com"* && "$*" == *"python"* ]]; then
    # No Python versions built yet
    echo '[]'
else
    echo "Mock curl: Unknown URL $*" >&2
    exit 1
fi
EOF
    chmod +x "$TEST_DIR/mocks/curl"
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    [ "$status" -eq 0 ]
    
    # Get JSON output
    local json_output
    json_output=$(run_script_and_get_json)
    
    # Verify comprehensive outputs
    local updates_detected
    updates_detected=$(echo "$json_output" | jq -r '.summary.updates_detected')
    [[ "$updates_detected" == "true" ]]
    
    # Should have supported versions for both runtimes
    local nodejs_supported python_supported
    nodejs_supported=$(echo "$json_output" | jq -r '.nodejs.supported_versions | length')
    python_supported=$(echo "$json_output" | jq -r '.python.supported_versions | length')
    
    [[ "$nodejs_supported" -gt 0 ]]
    [[ "$python_supported" -gt 0 ]]
    
    # Should have versions to build (Node.js outdated, Python not built)
    local nodejs_updates python_updates
    nodejs_updates=$(echo "$json_output" | jq -r '.nodejs.versions_to_build | length')
    python_updates=$(echo "$json_output" | jq -r '.python.versions_to_build | length')
    
    [[ "$nodejs_updates" -gt 0 ]]
    [[ "$python_updates" -gt 0 ]]
    
    # Check GitHub Actions outputs
    grep -q "updates-detected=true" "$GITHUB_OUTPUT"
}