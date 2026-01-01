#!/bin/bash
set -euo pipefail

# Combined upstream version detection and build trigger script
# This script checks for new upstream runtime versions and triggers container builds
# when updates are detected, eliminating the need for complex GitHub Actions logic.
#
# Handles:
# - Node.js and Python version checking
# - Dynamic version comparison against container registry
# - GitHub Actions workflow dispatch for builds
# - All error handling and logging
#
# Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 12.6

# Supported versions (matching Makefile)
NODEJS_VERSIONS="${NODEJS_VERSIONS:-18 20 22}"
PYTHON_VERSIONS="${PYTHON_VERSIONS:-3.11 3.12 3.13}"

# Container runtime detection (matching Makefile logic)
DOCKER_CMD="${DOCKER_CMD:-$(
    if command -v docker >/dev/null 2>&1; then
        echo "docker"
    elif command -v podman >/dev/null 2>&1; then
        echo "podman"
    elif command -v nerdctl >/dev/null 2>&1; then
        echo "nerdctl"
    elif command -v finch >/dev/null 2>&1; then
        echo "finch"
    elif command -v docker.exe >/dev/null 2>&1; then
        echo "docker.exe"
    else
        echo ""
    fi
)}"

# Initialize logging
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >&2
}

error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" >&2
}

# Configuration
GITHUB_API_URL="${GITHUB_API_URL:-https://api.github.com}"
GITHUB_REPOSITORY="${GITHUB_REPOSITORY:-}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"
FORCE_CHECK="${FORCE_CHECK:-false}"

# Registry configuration (matching Makefile)
REGISTRY="${REGISTRY:-ghcr.io}"
OWNER="${OWNER:-$(git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\).*/\1/' 2>/dev/null || echo "")}"
REPO_NAME="${REPO_NAME:-$(basename $(git rev-parse --show-toplevel 2>/dev/null) 2>/dev/null || echo "mcp-container-images")}"

# Image names (matching Makefile)
NODEJS_IMAGE="${NODEJS_IMAGE:-$REGISTRY/$OWNER/$REPO_NAME-nodejs}"
PYTHON_IMAGE="${PYTHON_IMAGE:-$REGISTRY/$OWNER/$REPO_NAME-python}"

# Validate environment
validate_environment() {
    log "Validating environment..."
    
    # Check required tools
    for tool in jq curl; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            error "$tool is required but not installed"
            exit 1
        fi
    done
    
    # Check GitHub environment for build triggers
    if [[ -n "$GITHUB_REPOSITORY" && -n "$GITHUB_TOKEN" ]]; then
        log "GitHub environment detected - builds will be triggered"
    else
        log "WARNING: GitHub environment not detected - builds will not be triggered"
    fi
    
    log "Environment validation completed"
}

# Check if we should skip due to recent check (simplified without versions.json)
should_skip_check() {
    if [[ "$FORCE_CHECK" == "true" ]]; then
        log "Force check requested - bypassing recent check validation"
        return 1  # Don't skip
    fi
    
    # For now, always run checks since we don't have persistent state
    # In the future, this could check container registry timestamps
    return 1  # Don't skip
}

# Get current version for a runtime/major from container registry or fallback
get_current_version() {
    local runtime="$1"
    local major="$2"
    
    # Try to get version from local container images first
    local image_name
    if [[ "$runtime" == "nodejs" ]]; then
        image_name="$NODEJS_IMAGE"
    else
        image_name="$PYTHON_IMAGE"
    fi
    
    # Check if we have container runtime available
    if [[ -n "$DOCKER_CMD" ]]; then
        # Try to get version from local image tags
        local current_version
        if [[ "$runtime" == "nodejs" ]]; then
            current_version=$($DOCKER_CMD image ls --format "table {{.Tag}}" "$image_name" 2>/dev/null | \
                grep "^node$major\." | head -1 | sed "s/^node//")
        else
            current_version=$($DOCKER_CMD image ls --format "table {{.Tag}}" "$image_name" 2>/dev/null | \
                grep "^python$major\." | head -1 | sed "s/^python//")
        fi
        
        if [[ -n "$current_version" && "$current_version" != "Tag" ]]; then
            echo "$current_version"
            return 0
        fi
    fi
    
    # Fallback to reasonable defaults if no local images found
    if [[ "$runtime" == "nodejs" ]]; then
        case "$major" in
            18) echo "18.19.0" ;;
            20) echo "20.11.0" ;;
            22) echo "22.11.0" ;;
            *) echo "$major.0.0" ;;
        esac
    else
        case "$major" in
            3.11) echo "3.11.8" ;;
            3.12) echo "3.12.8" ;;
            3.13) echo "3.13.1" ;;
            *) echo "$major.0" ;;
        esac
    fi
}

# Check Node.js version for a specific major version
check_nodejs() {
    local major="$1"
    local current="$2"
    
    log "Checking Node.js $major version (current: $current)"
    
    local latest=""
    local retry_count=0
    local max_retries=3
    
    while [[ $retry_count -lt $max_retries ]]; do
        if latest=$(curl -s --max-time 30 https://nodejs.org/dist/index.json 2>/dev/null | \
            jq -r --arg maj "$major" '[.[] | select(.version | startswith("v" + $maj + "."))] | .[0].version' 2>/dev/null | tr -d 'v'); then
            
            if [[ -n "$latest" && "$latest" != "null" && "$latest" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
                log "Successfully retrieved Node.js $major latest version: $latest"
                break
            else
                log "Invalid response for Node.js $major: '$latest'"
                latest=""
            fi
        fi
        
        retry_count=$((retry_count + 1))
        if [[ $retry_count -lt $max_retries ]]; then
            log "Retrying Node.js $major version check (attempt $((retry_count + 1))/$max_retries)..."
            sleep 2
        fi
    done
    
    if [[ -z "$latest" ]]; then
        error "Failed to retrieve Node.js $major version after $max_retries attempts"
        return 1
    fi
    
    # Compare versions and return result
    if [[ "$latest" != "$current" ]]; then
        log "🆕 New Node.js $major version detected: $current -> $latest"
        echo "$latest"  # Return new version
        return 0
    else
        log "✅ Node.js $major version $current is current"
        return 1  # No update
    fi
}

# Check Python version for a specific major.minor version
check_python() {
    local major="$1"
    local current="$2"
    
    log "Checking Python $major version (current: $current)"
    
    local latest=""
    local retry_count=0
    local max_retries=3
    
    while [[ $retry_count -lt $max_retries ]]; do
        if latest=$(curl -s --max-time 30 https://endoflife.date/api/python.json 2>/dev/null | \
            jq -r --arg maj "$major" '[.[] | select(.cycle == $maj)] | .[0].latest' 2>/dev/null); then
            
            if [[ -n "$latest" && "$latest" != "null" && "$latest" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
                log "Successfully retrieved Python $major latest version: $latest"
                break
            else
                log "Invalid response for Python $major: '$latest'"
                latest=""
            fi
        fi
        
        retry_count=$((retry_count + 1))
        if [[ $retry_count -lt $max_retries ]]; then
            log "Retrying Python $major version check (attempt $((retry_count + 1))/$max_retries)..."
            sleep 2
        fi
    done
    
    if [[ -z "$latest" ]]; then
        error "Failed to retrieve Python $major version after $max_retries attempts"
        return 1
    fi
    
    # Compare versions and return result
    if [[ "$latest" != "$current" ]]; then
        log "🆕 New Python $major version detected: $current -> $latest"
        echo "$latest"  # Return new version
        return 0
    else
        log "✅ Python $major version $current is current"
        return 1  # No update
    fi
}

# Trigger container build via GitHub Actions
trigger_build() {
    local runtime="$1"
    local major="$2"
    local new_version="$3"
    
    if [[ -z "$GITHUB_REPOSITORY" || -z "$GITHUB_TOKEN" ]]; then
        log "⚠️  Cannot trigger build - GitHub environment not configured"
        return 1
    fi
    
    log "🚀 Triggering $runtime $major build (version: $new_version)..."
    
    local response
    response=$(curl -s -w "%{http_code}" -X POST \
        -H "Accept: application/vnd.github.v3+json" \
        -H "Authorization: token $GITHUB_TOKEN" \
        -H "Content-Type: application/json" \
        "$GITHUB_API_URL/repos/$GITHUB_REPOSITORY/actions/workflows/build-containers.yml/dispatches" \
        -d "{\"ref\":\"main\",\"inputs\":{\"force_build\":\"true\",\"build_reason\":\"$runtime-$major-update\",\"${runtime}_version\":\"$new_version\"}}")
    
    local http_code="${response: -3}"
    local response_body="${response%???}"
    
    if [[ "$http_code" == "204" ]]; then
        log "✅ Successfully triggered $runtime $major build"
        return 0
    else
        error "Failed to trigger $runtime $major build (HTTP $http_code): $response_body"
        return 1
    fi
}

# Main execution
main() {
    echo "🔍 MCP CONTAINER VERSION CHECK AND BUILD" >&2
    echo "=======================================" >&2
    log "Starting upstream version detection and build triggering..."
    log "Execution environment:"
    log "  Script: $0"
    log "  Working directory: $(pwd)"
    log "  GitHub Repository: ${GITHUB_REPOSITORY:-'not set'}"
    log "  Force check: $FORCE_CHECK"
    log "  Container Runtime: ${DOCKER_CMD:-'not available'}"
    log "  Current time: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
    echo "" >&2
    
    # Validate environment
    validate_environment
    
    # Check if we should skip
    if should_skip_check; then
        log "✅ Version check skipped - no action needed"
        exit 0
    fi
    
    # Show current configuration
    log "Configuration:"
    log "  Node.js versions: $NODEJS_VERSIONS"
    log "  Python versions: $PYTHON_VERSIONS"
    log "  Node.js image: $NODEJS_IMAGE"
    log "  Python image: $PYTHON_IMAGE"
    echo "" >&2
    
    local updates_detected=false
    local builds_triggered=0
    local check_errors=0
    
    echo "🔍 CHECKING VERSIONS" >&2
    echo "===================" >&2
    
    # Check all supported Node.js versions
    log "Checking Node.js versions..."
    for major in $NODEJS_VERSIONS; do
        local current new_version
        current=$(get_current_version "nodejs" "$major")
        
        log "Current Node.js $major: $current"
        
        if new_version=$(check_nodejs "$major" "$current"); then
            updates_detected=true
            
            # Trigger build
            if trigger_build "nodejs" "$major" "$new_version"; then
                ((builds_triggered++))
            fi
        fi
    done
    
    # Check all supported Python versions
    log "Checking Python versions..."
    for major in $PYTHON_VERSIONS; do
        local current new_version
        current=$(get_current_version "python" "$major")
        
        log "Current Python $major: $current"
        
        if new_version=$(check_python "$major" "$current"); then
            updates_detected=true
            
            # Trigger build
            if trigger_build "python" "$major" "$new_version"; then
                ((builds_triggered++))
            fi
        fi
    done
    
    # Summary
    echo "📊 FINAL SUMMARY" >&2
    echo "===============" >&2
    
    if [[ "$updates_detected" == "true" ]]; then
        log "🆕 Updates detected and processed!"
        log "  Builds triggered: $builds_triggered"
        if [[ $builds_triggered -gt 0 ]]; then
            log "  Action: Monitor build progress in GitHub Actions"
        fi
    else
        log "✅ All versions are current"
        log "  Action: No updates needed"
    fi
    
    if [[ $check_errors -gt 0 ]]; then
        log "⚠️  Encountered $check_errors error(s) during version checks"
        log "  Action: Review error messages above"
    fi
    
    echo "=======================================" >&2
    log "✅ Version check and build trigger completed"
    log "Execution completed at: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
    
    # Exit with error code if there were check errors
    if [[ $check_errors -gt 0 ]]; then
        exit 1
    fi
}

# Handle command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --force)
            FORCE_CHECK=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [--force] [--help]"
            echo ""
            echo "Options:"
            echo "  --force    Force version check even if recently checked"
            echo "  --help     Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  GITHUB_REPOSITORY  GitHub repository (owner/repo)"
            echo "  GITHUB_TOKEN       GitHub token for API access"
            echo "  FORCE_CHECK        Set to 'true' to force check"
            echo "  NODEJS_VERSIONS    Space-separated Node.js major versions (default: 18 20 22)"
            echo "  PYTHON_VERSIONS    Space-separated Python major.minor versions (default: 3.11 3.12 3.13)"
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Execute main function
main "$@"