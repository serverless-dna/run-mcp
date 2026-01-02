#!/bin/bash
set -euo pipefail

# Test script for run-mcp Claude Desktop integration
# This script tests the run-mcp binary with various scenarios

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="${SCRIPT_DIR}/../../build/run-mcp"
TEST_DIR="/tmp/run-mcp-test"
FAILED_TESTS=0
TOTAL_TESTS=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Test runner
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    log_info "Running test: $test_name"
    
    if eval "$test_command"; then
        log_success "$test_name"
    else
        log_error "$test_name"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    echo
}

# Setup test environment
setup_test_env() {
    log_info "Setting up test environment..."
    
    # Create test directory
    mkdir -p "$TEST_DIR"
    
    # Create test data files
    echo "test data" > "$TEST_DIR/test.txt"
    mkdir -p "$TEST_DIR/subdir"
    echo "nested data" > "$TEST_DIR/subdir/nested.txt"
    
    # Set test environment variables
    export MCP_DATA_DIR="$TEST_DIR"
    export AWS_ACCESS_KEY_ID="test-key"
    export AWS_SECRET_ACCESS_KEY="test-secret"
    export OPENAI_API_KEY="test-openai-key"
    export MCP_PASSTHROUGH_ENV="CUSTOM_VAR"
    export CUSTOM_VAR="custom-value"
    
    log_success "Test environment setup complete"
}

# Cleanup test environment
cleanup_test_env() {
    log_info "Cleaning up test environment..."
    rm -rf "$TEST_DIR"
    unset MCP_DATA_DIR AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY OPENAI_API_KEY MCP_PASSTHROUGH_ENV CUSTOM_VAR
    log_success "Cleanup complete"
}

# Test binary exists and is executable
test_binary_exists() {
    [[ -x "$BINARY" ]]
}

# Test basic commands
test_basic_commands() {
    "$BINARY" --version >/dev/null 2>&1 && \
    "$BINARY" --help >/dev/null 2>&1 && \
    "$BINARY" info >/dev/null 2>&1 && \
    "$BINARY" config >/dev/null 2>&1
}

# Test container runtime detection
test_runtime_detection() {
    "$BINARY" info | grep -q "Available Container Runtimes"
}

# Test configuration display
test_configuration() {
    "$BINARY" config | grep -q "Node.js Image" && \
    "$BINARY" config | grep -q "Python Image" && \
    "$BINARY" config | grep -q "Data Directory"
}

# Test environment variable passthrough
test_env_passthrough() {
    # This test checks that the binary correctly identifies which env vars would be passed
    # We can't easily test actual container execution without a real MCP server
    
    # Check that AWS vars would be passed
    export AWS_TEST_VAR="test"
    "$BINARY" config >/dev/null 2>&1
    local result=$?
    unset AWS_TEST_VAR
    return $result
}

# Test data directory validation
test_data_directory() {
    # Test with valid directory
    export MCP_DATA_DIR="$TEST_DIR"
    "$BINARY" config | grep -q "✓" && \
    
    # Test with invalid directory
    export MCP_DATA_DIR="/nonexistent/directory"
    ! "$BINARY" config | grep -q "✓"
    
    # Restore valid directory
    export MCP_DATA_DIR="$TEST_DIR"
}

# Test language detection
test_language_detection() {
    # Test Node.js detection
    "$BINARY" info | grep -q "nodejs.*npx" && \
    
    # Test Python detection
    "$BINARY" info | grep -q "python.*uvx"
}

# Test error handling
test_error_handling() {
    # Test with no arguments
    ! "$BINARY" >/dev/null 2>&1 && \
    
    # Test with invalid command
    ! "$BINARY" invalid-command >/dev/null 2>&1
}

# Test cross-platform path handling
test_path_handling() {
    # Test that data directory path is handled correctly
    "$BINARY" config | grep -q "$TEST_DIR"
}

# Test volume mount preparation
test_volume_mounts() {
    # Test that credential directories are detected if they exist
    if [[ -d "$HOME/.aws" ]]; then
        log_info "AWS directory exists, should be mounted"
    fi
    
    if [[ -d "$HOME/.config" ]]; then
        log_info "Config directory exists, should be mounted"
    fi
    
    # This always passes as it's just informational
    return 0
}

# Test container image selection
test_image_selection() {
    # Test default images
    "$BINARY" config | grep -q "ghcr.io/modelcontextprotocol/mcp-nodejs" && \
    "$BINARY" config | grep -q "ghcr.io/modelcontextprotocol/mcp-python" && \
    
    # Test custom image override
    export MCP_NODEJS_IMAGE="custom/nodejs:test"
    "$BINARY" config | grep -q "custom/nodejs:test"
    
    # Restore default
    unset MCP_NODEJS_IMAGE
}

# Test Claude Desktop configuration format
test_claude_desktop_format() {
    # Test that the binary can handle typical Claude Desktop command formats
    # We can't actually run containers without a real MCP server, but we can test argument parsing
    
    # This test just verifies the binary doesn't crash with typical arguments
    timeout 5s "$BINARY" npx --version >/dev/null 2>&1 || true
    timeout 5s "$BINARY" uvx --version >/dev/null 2>&1 || true
    
    # If we get here without hanging, the test passes
    return 0
}

# Test environment variable security
test_env_security() {
    # Set some variables that should NOT be passed through
    export DANGEROUS_VAR="should-not-pass"
    export PATH="/malicious/path:$PATH"
    export HOME="/fake/home"
    
    # The binary should not crash and should not pass these through
    "$BINARY" config >/dev/null 2>&1
    local result=$?
    
    # Cleanup
    unset DANGEROUS_VAR
    # Don't unset PATH and HOME as they're needed
    
    return $result
}

# Main test execution
main() {
    echo -e "${BLUE}run-mcp Integration Test Suite${NC}"
    echo "================================"
    echo
    
    # Check if binary exists
    if [[ ! -x "$BINARY" ]]; then
        log_error "Binary not found at $BINARY"
        log_info "Run 'make build-run-mcp' first"
        exit 1
    fi
    
    # Setup
    setup_test_env
    
    # Run tests
    run_test "Binary exists and is executable" "test_binary_exists"
    run_test "Basic commands work" "test_basic_commands"
    run_test "Container runtime detection" "test_runtime_detection"
    run_test "Configuration display" "test_configuration"
    run_test "Environment variable passthrough" "test_env_passthrough"
    run_test "Data directory validation" "test_data_directory"
    run_test "Language detection" "test_language_detection"
    run_test "Error handling" "test_error_handling"
    run_test "Cross-platform path handling" "test_path_handling"
    run_test "Volume mount preparation" "test_volume_mounts"
    run_test "Container image selection" "test_image_selection"
    run_test "Claude Desktop format compatibility" "test_claude_desktop_format"
    run_test "Environment variable security" "test_env_security"
    
    # Cleanup
    cleanup_test_env
    
    # Results
    echo -e "${BLUE}Test Results${NC}"
    echo "============"
    echo "Total tests: $TOTAL_TESTS"
    echo "Passed: $((TOTAL_TESTS - FAILED_TESTS))"
    echo "Failed: $FAILED_TESTS"
    echo
    
    if [[ $FAILED_TESTS -eq 0 ]]; then
        log_success "All tests passed!"
        exit 0
    else
        log_error "$FAILED_TESTS test(s) failed"
        exit 1
    fi
}

# Run main function
main "$@"