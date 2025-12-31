#!/bin/bash
set -euo pipefail

# Combined upstream version detection and build trigger script
# This script checks for new upstream runtime versions and triggers container builds
# when updates are detected, eliminating the need for complex GitHub Actions logic.
#
# Handles:
# - Node.js and Python version checking
# - Version comparison against versions.json
# - GitHub Actions workflow dispatch for builds
# - All error handling and logging
#
# Requirements: 12.1, 12.2, 12.3, 12.4, 12.5, 12.6

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

# Validate environment
validate_environment() {
    log "Validating environment..."
    
    # Check if versions.json exists
    if [[ ! -f "versions.json" ]]; then
        error "versions.json not found in current directory"
        exit 1
    fi
    
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

# Check if we should skip due to recent check
should_skip_check() {
    if [[ "$FORCE_CHECK" == "true" ]]; then
        log "Force check requested - bypassing recent check validation"
        return 1  # Don't skip
    fi
    
    if [[ -f "versions.json" ]]; then
        local last_checked
        last_checked=$(jq -r '.lastChecked' versions.json)
        
        if [[ "$last_checked" != "null" && -n "$last_checked" ]]; then
            local last_checked_epoch current_epoch hours_since_check
            last_checked_epoch=$(date -d "$last_checked" +%s 2>/dev/null || echo "0")
            current_epoch=$(date +%s)
            hours_since_check=$(( (current_epoch - last_checked_epoch) / 3600 ))
            
            if [[ $hours_since_check -lt 144 ]]; then  # 6 days = 144 hours
                log "⏭️  Skipping check - last checked $hours_since_check hours ago (less than 144 hours)"
                log "   Last check: $last_checked"
                log "   Use FORCE_CHECK=true to override this behavior"
                return 0  # Skip
            fi
        fi
    fi
    
    return 1  # Don't skip
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

# Update versions.json with new version
update_version_file() {
    local runtime="$1"
    local major="$2"
    local new_version="$3"
    
    log "📝 Updating versions.json: $runtime $major -> $new_version"
    
    local temp_file
    temp_file=$(mktemp)
    
    if jq --arg runtime "$runtime" --arg major "$major" --arg version "$new_version" \
        '.[$runtime][$major] = $version' versions.json > "$temp_file"; then
        mv "$temp_file" versions.json
        log "✅ Updated $runtime $major to $new_version in versions.json"
        return 0
    else
        error "Failed to update versions.json"
        rm -f "$temp_file"
        return 1
    fi
}

# Update last checked timestamp
update_last_checked() {
    log "Updating last checked timestamp..."
    
    local timestamp
    timestamp=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
    
    local temp_file
    temp_file=$(mktemp)
    
    if jq --arg timestamp "$timestamp" '.lastChecked = $timestamp' versions.json > "$temp_file"; then
        mv "$temp_file" versions.json
        log "Updated lastChecked to: $timestamp"
        return 0
    else
        error "Failed to update lastChecked timestamp"
        rm -f "$temp_file"
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
    log "  Versions file: versions.json"
    log "  GitHub Repository: ${GITHUB_REPOSITORY:-'not set'}"
    log "  Force check: $FORCE_CHECK"
    log "  Current time: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
    echo "" >&2
    
    # Validate environment
    validate_environment
    
    # Check if we should skip
    if should_skip_check; then
        log "✅ Version check skipped - no action needed"
        exit 0
    fi
    
    # Show current versions
    log "Current tracked versions:"
    jq -r '.nodejs | to_entries[] | "  Node.js \(.key): \(.value)"' versions.json | while IFS= read -r line; do
        log "$line"
    done
    jq -r '.python | to_entries[] | "  Python \(.key): \(.value)"' versions.json | while IFS= read -r line; do
        log "$line"
    done
    
    local last_checked
    last_checked=$(jq -r '.lastChecked' versions.json)
    log "  Last checked: $last_checked"
    echo "" >&2
    
    local updates_detected=false
    local builds_triggered=0
    local check_errors=0
    
    echo "🔍 CHECKING VERSIONS" >&2
    echo "===================" >&2
    
    # Check all tracked Node.js versions
    log "Checking Node.js versions..."
    while IFS= read -r major; do
        if [[ -n "$major" ]]; then
            local current new_version
            current=$(jq -r ".nodejs.\"$major\"" versions.json)
            
            if [[ "$current" == "null" || -z "$current" ]]; then
                error "No current version found for Node.js $major in versions.json"
                ((check_errors++))
                continue
            fi
            
            if new_version=$(check_nodejs "$major" "$current"); then
                updates_detected=true
                
                # Update versions.json
                if update_version_file "nodejs" "$major" "$new_version"; then
                    # Trigger build
                    if trigger_build "nodejs" "$major" "$new_version"; then
                        ((builds_triggered++))
                    fi
                fi
            fi
        fi
    done < <(jq -r '.nodejs | keys[]' versions.json)
    
    # Check all tracked Python versions
    log "Checking Python versions..."
    while IFS= read -r major; do
        if [[ -n "$major" ]]; then
            local current new_version
            current=$(jq -r ".python.\"$major\"" versions.json)
            
            if [[ "$current" == "null" || -z "$current" ]]; then
                error "No current version found for Python $major in versions.json"
                ((check_errors++))
                continue
            fi
            
            if new_version=$(check_python "$major" "$current"); then
                updates_detected=true
                
                # Update versions.json
                if update_version_file "python" "$major" "$new_version"; then
                    # Trigger build
                    if trigger_build "python" "$major" "$new_version"; then
                        ((builds_triggered++))
                    fi
                fi
            fi
        fi
    done < <(jq -r '.python | keys[]' versions.json)
    
    # Update last checked timestamp
    update_last_checked
    
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