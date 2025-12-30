#!/bin/bash
set -euo pipefail

# Integration test for change detection
# Tests the detect-changes.sh script with real git scenarios

echo "Running change detection integration tests..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
TESTS_RUN=0
TESTS_PASSED=0

run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -e "${BLUE}Running: $test_name${NC}"
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if eval "$test_command"; then
        echo -e "${GREEN}✓ $test_name passed${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ $test_name failed${NC}"
    fi
    echo ""
}

# Test 1: Script runs without error
run_test "Script execution" "GITHUB_OUTPUT=/tmp/integration-test bash scripts/detect-changes.sh"

# Test 2: Output format validation
run_test "Output format validation" "
    GITHUB_OUTPUT=/tmp/integration-format bash scripts/detect-changes.sh &&
    grep -E '^nodejs-changed=(true|false)$' /tmp/integration-format &&
    grep -E '^python-changed=(true|false)$' /tmp/integration-format &&
    grep -E '^build-reason=.+$' /tmp/integration-format
"

# Test 3: Environment variable handling
run_test "Missing GITHUB_OUTPUT handling" "
    unset GITHUB_OUTPUT &&
    bash scripts/detect-changes.sh >/dev/null 2>&1
"

# Test 4: Git repository validation
run_test "Git repository validation" "
    cd /tmp &&
    mkdir -p test-no-git &&
    cd test-no-git &&
    ! bash $OLDPWD/scripts/detect-changes.sh 2>/dev/null &&
    cd $OLDPWD &&
    rm -rf /tmp/test-no-git
"

# Summary
echo -e "${BLUE}Integration Test Summary:${NC}"
echo -e "Tests run: $TESTS_RUN"
echo -e "Tests passed: $TESTS_PASSED"

if [ $TESTS_PASSED -eq $TESTS_RUN ]; then
    echo -e "${GREEN}All integration tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some integration tests failed!${NC}"
    exit 1
fi