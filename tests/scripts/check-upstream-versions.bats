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
}

teardown() {
    rm -rf "$TEST_DIR"
}

# Helper function to create mock curl responses
create_mock_curl() {
    local nodejs_response="$1"
    local python_response="$2"
    
    cat > "$TEST_DIR/mocks/curl" << EOF
#!/bin/bash
if [[ "\$*" == *"nodejs.org"* ]]; then
    echo '$nodejs_response'
elif [[ "\$*" == *"endoflife.date"* ]]; then
    echo '$python_response'
else
    echo "Mock curl: Unknown URL \$*" >&2
    exit 1
fi
EOF
    chmod +x "$TEST_DIR/mocks/curl"
}

# Property Test: Current versions should not trigger updates
@test "Property 18.1: Current versions detected as no updates needed" {
    # Copy current versions fixture
    cp "$FIXTURES_DIR/versions-current.json" versions.json
    
    # Mock API responses with current versions
    create_mock_curl \
        '[{"version": "v22.11.0"}, {"version": "v20.18.1"}]' \
        '[{"cycle": "3.12", "latest": "3.12.8"}, {"cycle": "3.11", "latest": "3.11.11"}]'
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -eq 0 ]
    grep -q "updates-detected=false" "$GITHUB_OUTPUT"
    grep -q "nodejs-22-update=false" "$GITHUB_OUTPUT"
    grep -q "nodejs-20-update=false" "$GITHUB_OUTPUT"
    grep -q "python-3.12-update=false" "$GITHUB_OUTPUT"
    grep -q "python-3.11-update=false" "$GITHUB_OUTPUT"
}

# Property Test: Outdated versions should trigger updates
@test "Property 18.2: Outdated versions detected as updates needed" {
    # Copy outdated versions fixture
    cp "$FIXTURES_DIR/versions-outdated.json" versions.json
    
    # Mock API responses with newer versions
    create_mock_curl \
        '[{"version": "v22.11.0"}, {"version": "v20.18.1"}]' \
        '[{"cycle": "3.12", "latest": "3.12.8"}, {"cycle": "3.11", "latest": "3.11.11"}]'
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -eq 0 ]
    grep -q "updates-detected=true" "$GITHUB_OUTPUT"
    grep -q "nodejs-22-update=true" "$GITHUB_OUTPUT"
    grep -q "nodejs-20-update=true" "$GITHUB_OUTPUT"
    grep -q "python-3.12-update=true" "$GITHUB_OUTPUT"
    grep -q "python-3.11-update=true" "$GITHUB_OUTPUT"
    
    # Check that new versions are output
    grep -q "nodejs-22=22.11.0" "$GITHUB_OUTPUT"
    grep -q "nodejs-20=20.18.1" "$GITHUB_OUTPUT"
    grep -q "python-3.12=3.12.8" "$GITHUB_OUTPUT"
    grep -q "python-3.11=3.11.11" "$GITHUB_OUTPUT"
}

# Property Test: Partial updates (only Node.js newer)
@test "Property 18.3: Partial updates detected correctly (Node.js only)" {
    # Create versions with only Node.js outdated
    cat > versions.json << 'EOF'
{
  "nodejs": {
    "22": "22.10.0",
    "20": "20.17.0"
  },
  "python": {
    "3.12": "3.12.8",
    "3.11": "3.11.11"
  },
  "lastChecked": "2025-01-01T00:00:00Z"
}
EOF
    
    # Mock API responses with Node.js updates but Python current
    create_mock_curl \
        '[{"version": "v22.11.0"}, {"version": "v20.18.1"}]' \
        '[{"cycle": "3.12", "latest": "3.12.8"}, {"cycle": "3.11", "latest": "3.11.11"}]'
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -eq 0 ]
    grep -q "updates-detected=true" "$GITHUB_OUTPUT"
    grep -q "nodejs-22-update=true" "$GITHUB_OUTPUT"
    grep -q "nodejs-20-update=true" "$GITHUB_OUTPUT"
    grep -q "python-3.12-update=false" "$GITHUB_OUTPUT"
    grep -q "python-3.11-update=false" "$GITHUB_OUTPUT"
    
    # Check that nodejs-updates contains both versions (order may vary)
    local nodejs_updates_line
    nodejs_updates_line=$(grep "nodejs-updates=" "$GITHUB_OUTPUT" || echo "nodejs-updates=")
    [[ "$nodejs_updates_line" == *"22"* ]]
    [[ "$nodejs_updates_line" == *"20"* ]]
    
    # Check that python-updates is empty
    grep -q "python-updates=$" "$GITHUB_OUTPUT" || grep -q "python-updates=\"\"" "$GITHUB_OUTPUT"
}

# Property Test: Partial updates (only Python newer)
@test "Property 18.4: Partial updates detected correctly (Python only)" {
    # Create versions with only Python outdated
    cat > versions.json << 'EOF'
{
  "nodejs": {
    "22": "22.11.0",
    "20": "20.18.1"
  },
  "python": {
    "3.12": "3.12.7",
    "3.11": "3.11.10"
  },
  "lastChecked": "2025-01-01T00:00:00Z"
}
EOF
    
    # Mock API responses with Python updates but Node.js current
    create_mock_curl \
        '[{"version": "v22.11.0"}, {"version": "v20.18.1"}]' \
        '[{"cycle": "3.12", "latest": "3.12.8"}, {"cycle": "3.11", "latest": "3.11.11"}]'
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -eq 0 ]
    grep -q "updates-detected=true" "$GITHUB_OUTPUT"
    grep -q "nodejs-22-update=false" "$GITHUB_OUTPUT"
    grep -q "nodejs-20-update=false" "$GITHUB_OUTPUT"
    grep -q "python-3.12-update=true" "$GITHUB_OUTPUT"
    grep -q "python-3.11-update=true" "$GITHUB_OUTPUT"
    grep -q "nodejs-updates=" "$GITHUB_OUTPUT"
    
    # Check that python-updates contains both versions (order may vary)
    local python_updates_line
    python_updates_line=$(grep "python-updates=" "$GITHUB_OUTPUT" || echo "python-updates=")
    [[ "$python_updates_line" == *"3.12"* ]]
    [[ "$python_updates_line" == *"3.11"* ]]
}

# Property Test: API failure handling for Node.js
@test "Property 18.5: Node.js API failure handled gracefully" {
    # Copy current versions fixture
    cp "$FIXTURES_DIR/versions-current.json" versions.json
    
    # Create mock curl that fails for Node.js API
    cat > "$TEST_DIR/mocks/curl" << 'EOF'
#!/bin/bash
if [[ "$*" == *"nodejs.org"* ]]; then
    echo "API Error" >&2
    exit 1
elif [[ "$*" == *"endoflife.date"* ]]; then
    echo '[{"cycle": "3.12", "latest": "3.12.8"}, {"cycle": "3.11", "latest": "3.11.11"}]'
else
    echo "Mock curl: Unknown URL $*" >&2
    exit 1
fi
EOF
    chmod +x "$TEST_DIR/mocks/curl"
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -eq 0 ]
    grep -q "check-errors=true" "$GITHUB_OUTPUT"
    grep -q "nodejs-22-error=api-failure" "$GITHUB_OUTPUT"
    grep -q "nodejs-20-error=api-failure" "$GITHUB_OUTPUT"
    # Python should still work
    grep -q "python-3.12-update=false" "$GITHUB_OUTPUT"
    grep -q "python-3.11-update=false" "$GITHUB_OUTPUT"
}

# Property Test: API failure handling for Python
@test "Property 18.6: Python API failure handled gracefully" {
    # Copy current versions fixture
    cp "$FIXTURES_DIR/versions-current.json" versions.json
    
    # Create mock curl that fails for Python API
    cat > "$TEST_DIR/mocks/curl" << 'EOF'
#!/bin/bash
if [[ "$*" == *"nodejs.org"* ]]; then
    echo '[{"version": "v22.11.0"}, {"version": "v20.18.1"}]'
elif [[ "$*" == *"endoflife.date"* ]]; then
    echo "API Error" >&2
    exit 1
else
    echo "Mock curl: Unknown URL $*" >&2
    exit 1
fi
EOF
    chmod +x "$TEST_DIR/mocks/curl"
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -eq 0 ]
    grep -q "check-errors=true" "$GITHUB_OUTPUT"
    grep -q "python-3.12-error=api-failure" "$GITHUB_OUTPUT"
    grep -q "python-3.11-error=api-failure" "$GITHUB_OUTPUT"
    # Node.js should still work
    grep -q "nodejs-22-update=false" "$GITHUB_OUTPUT"
    grep -q "nodejs-20-update=false" "$GITHUB_OUTPUT"
}

# Property Test: Invalid JSON response handling
@test "Property 18.7: Invalid JSON responses handled gracefully" {
    # Copy current versions fixture
    cp "$FIXTURES_DIR/versions-current.json" versions.json
    
    # Mock API responses with invalid JSON
    create_mock_curl \
        'invalid json response' \
        'also invalid json'
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -eq 0 ]
    grep -q "check-errors=true" "$GITHUB_OUTPUT"
    grep -q "nodejs-22-error=api-failure" "$GITHUB_OUTPUT"
    grep -q "nodejs-20-error=api-failure" "$GITHUB_OUTPUT"
    grep -q "python-3.12-error=api-failure" "$GITHUB_OUTPUT"
    grep -q "python-3.11-error=api-failure" "$GITHUB_OUTPUT"
}

# Property Test: Missing versions.json file
@test "Property 18.8: Missing versions.json file causes failure" {
    # Don't create versions.json file
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"versions.json not found"* ]]
}

# Property Test: Malformed versions.json file
@test "Property 18.9: Malformed versions.json handled gracefully" {
    # Create malformed versions.json
    echo "invalid json" > versions.json
    
    # Mock valid API responses
    create_mock_curl \
        '[{"version": "v22.11.0"}, {"version": "v20.18.1"}]' \
        '[{"cycle": "3.12", "latest": "3.12.8"}, {"cycle": "3.11", "latest": "3.11.11"}]'
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -ne 0 ]
    [[ "$output" == *"parse error"* ]] || [[ "$output" == *"Invalid"* ]]
}

# Property Test: Empty API responses
@test "Property 18.10: Empty API responses handled gracefully" {
    # Copy current versions fixture
    cp "$FIXTURES_DIR/versions-current.json" versions.json
    
    # Mock empty API responses
    create_mock_curl '[]' '[]'
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -eq 0 ]
    grep -q "check-errors=true" "$GITHUB_OUTPUT"
    grep -q "nodejs-22-error=api-failure" "$GITHUB_OUTPUT"
    grep -q "nodejs-20-error=api-failure" "$GITHUB_OUTPUT"
    grep -q "python-3.12-error=api-failure" "$GITHUB_OUTPUT"
    grep -q "python-3.11-error=api-failure" "$GITHUB_OUTPUT"
}

# Property Test: Timestamp update functionality
@test "Property 18.11: lastChecked timestamp updated correctly" {
    # Copy current versions fixture
    cp "$FIXTURES_DIR/versions-current.json" versions.json
    
    # Store original timestamp
    ORIGINAL_TIMESTAMP=$(jq -r '.lastChecked' versions.json)
    
    # Mock API responses with current versions
    create_mock_curl \
        '[{"version": "v22.11.0"}, {"version": "v20.18.1"}]' \
        '[{"cycle": "3.12", "latest": "3.12.8"}, {"cycle": "3.11", "latest": "3.11.11"}]'
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -eq 0 ]
    
    # Check that timestamp was updated
    NEW_TIMESTAMP=$(jq -r '.lastChecked' versions.json)
    [[ "$NEW_TIMESTAMP" != "$ORIGINAL_TIMESTAMP" ]]
    
    # Check that timestamp is in correct format (ISO 8601)
    [[ "$NEW_TIMESTAMP" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$ ]]
    
    # Check that timestamp is output
    grep -q "last-checked=" "$GITHUB_OUTPUT"
}

# Property Test: Version comparison accuracy
@test "Property 18.12: Version comparison works correctly for patch versions" {
    # Create versions with specific patch versions
    cat > versions.json << 'EOF'
{
  "nodejs": {
    "22": "22.11.0"
  },
  "python": {
    "3.12": "3.12.8"
  },
  "lastChecked": "2025-01-01T00:00:00Z"
}
EOF
    
    # Mock API responses with same patch versions (should not trigger update)
    create_mock_curl \
        '[{"version": "v22.11.0"}]' \
        '[{"cycle": "3.12", "latest": "3.12.8"}]'
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -eq 0 ]
    grep -q "updates-detected=false" "$GITHUB_OUTPUT"
    grep -q "nodejs-22-update=false" "$GITHUB_OUTPUT"
    grep -q "python-3.12-update=false" "$GITHUB_OUTPUT"
    
    # Now test with newer patch versions
    create_mock_curl \
        '[{"version": "v22.11.1"}]' \
        '[{"cycle": "3.12", "latest": "3.12.9"}]'
    
    # Clear previous output
    > "$GITHUB_OUTPUT"
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -eq 0 ]
    grep -q "updates-detected=true" "$GITHUB_OUTPUT"
    grep -q "nodejs-22-update=true" "$GITHUB_OUTPUT"
    grep -q "python-3.12-update=true" "$GITHUB_OUTPUT"
    grep -q "nodejs-22=22.11.1" "$GITHUB_OUTPUT"
    grep -q "python-3.12=3.12.9" "$GITHUB_OUTPUT"
}

# Property Test: Multiple major versions handling
@test "Property 18.13: Multiple major versions handled independently" {
    # Create versions with multiple major versions
    cat > versions.json << 'EOF'
{
  "nodejs": {
    "22": "22.11.0",
    "20": "20.18.1",
    "18": "18.19.0"
  },
  "python": {
    "3.12": "3.12.8",
    "3.11": "3.11.11",
    "3.10": "3.10.13"
  },
  "lastChecked": "2025-01-01T00:00:00Z"
}
EOF
    
    # Mock API responses with updates for some versions
    create_mock_curl \
        '[{"version": "v22.12.0"}, {"version": "v20.18.1"}, {"version": "v18.19.0"}]' \
        '[{"cycle": "3.12", "latest": "3.12.9"}, {"cycle": "3.11", "latest": "3.11.11"}, {"cycle": "3.10", "latest": "3.10.13"}]'
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -eq 0 ]
    grep -q "updates-detected=true" "$GITHUB_OUTPUT"
    
    # Check Node.js versions
    grep -q "nodejs-22-update=true" "$GITHUB_OUTPUT"
    grep -q "nodejs-20-update=false" "$GITHUB_OUTPUT"
    grep -q "nodejs-18-update=false" "$GITHUB_OUTPUT"
    grep -q "nodejs-22=22.12.0" "$GITHUB_OUTPUT"
    
    # Check Python versions
    grep -q "python-3.12-update=true" "$GITHUB_OUTPUT"
    grep -q "python-3.11-update=false" "$GITHUB_OUTPUT"
    grep -q "python-3.10-update=false" "$GITHUB_OUTPUT"
    grep -q "python-3.12=3.12.9" "$GITHUB_OUTPUT"
    
    # Check update summaries
    grep -q "nodejs-updates=22" "$GITHUB_OUTPUT"
    grep -q "python-updates=3.12" "$GITHUB_OUTPUT"
}

# Property Test: Network timeout simulation
@test "Property 18.14: Network timeouts handled with retries" {
    # Copy current versions fixture
    cp "$FIXTURES_DIR/versions-current.json" versions.json
    
    # Create mock curl that times out initially then succeeds
    cat > "$TEST_DIR/mocks/curl" << 'EOF'
#!/bin/bash
CALL_COUNT_FILE="/tmp/curl_call_count_$$"
if [[ ! -f "$CALL_COUNT_FILE" ]]; then
    echo "1" > "$CALL_COUNT_FILE"
else
    COUNT=$(cat "$CALL_COUNT_FILE")
    echo $((COUNT + 1)) > "$CALL_COUNT_FILE"
fi

CALL_COUNT=$(cat "$CALL_COUNT_FILE")

if [[ "$*" == *"nodejs.org"* ]]; then
    if [[ $CALL_COUNT -le 2 ]]; then
        echo "Timeout" >&2
        exit 1
    else
        echo '[{"version": "v22.11.0"}, {"version": "v20.18.1"}]'
    fi
elif [[ "$*" == *"endoflife.date"* ]]; then
    echo '[{"cycle": "3.12", "latest": "3.12.8"}, {"cycle": "3.11", "latest": "3.11.11"}]'
else
    echo "Mock curl: Unknown URL $*" >&2
    exit 1
fi
EOF
    chmod +x "$TEST_DIR/mocks/curl"
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -eq 0 ]
    # Should eventually succeed after retries
    grep -q "nodejs-22-update=false" "$GITHUB_OUTPUT"
    grep -q "nodejs-20-update=false" "$GITHUB_OUTPUT"
    grep -q "python-3.12-update=false" "$GITHUB_OUTPUT"
    grep -q "python-3.11-update=false" "$GITHUB_OUTPUT"
}

# Property Test: Comprehensive integration test
@test "Property 18.15: Complete workflow with mixed update scenarios" {
    # Create a realistic scenario with some updates needed
    cat > versions.json << 'EOF'
{
  "nodejs": {
    "22": "22.10.0",
    "20": "20.18.1"
  },
  "python": {
    "3.12": "3.12.8",
    "3.11": "3.11.10"
  },
  "lastChecked": "2025-01-01T00:00:00Z"
}
EOF
    
    # Mock API responses with mixed updates
    create_mock_curl \
        '[{"version": "v22.11.0"}, {"version": "v20.18.1"}]' \
        '[{"cycle": "3.12", "latest": "3.12.8"}, {"cycle": "3.11", "latest": "3.11.11"}]'
    
    run bash "$SCRIPT_DIR/check-upstream-versions.sh"
    
    [ "$status" -eq 0 ]
    
    # Verify comprehensive outputs
    grep -q "updates-detected=true" "$GITHUB_OUTPUT"
    grep -q "check-errors=false" "$GITHUB_OUTPUT"
    
    # Node.js: 22 needs update, 20 is current
    grep -q "nodejs-22-update=true" "$GITHUB_OUTPUT"
    grep -q "nodejs-20-update=false" "$GITHUB_OUTPUT"
    grep -q "nodejs-22=22.11.0" "$GITHUB_OUTPUT"
    
    # Python: 3.12 is current, 3.11 needs update
    grep -q "python-3.12-update=false" "$GITHUB_OUTPUT"
    grep -q "python-3.11-update=true" "$GITHUB_OUTPUT"
    grep -q "python-3.11=3.11.11" "$GITHUB_OUTPUT"
    
    # Update summaries
    grep -q "nodejs-updates=22" "$GITHUB_OUTPUT"
    grep -q "python-updates=3.11" "$GITHUB_OUTPUT"
    
    # Timestamp should be updated
    grep -q "last-checked=" "$GITHUB_OUTPUT"
    
    # Verify versions.json was updated
    NEW_TIMESTAMP=$(jq -r '.lastChecked' versions.json)
    [[ "$NEW_TIMESTAMP" != "2025-01-01T00:00:00Z" ]]
}