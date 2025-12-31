#!/bin/bash
set -euo pipefail

# Upstream version detection script for MCP container images
# This script checks for new upstream runtime versions and sets GitHub Actions outputs
# for triggering container rebuilds when updates are detected.
#
# Handles:
# - Node.js version checking via nodejs.org API
# - Python version checking via endoflife.date API
# - Version comparison against versions.json
# - GitHub Actions output generation
# - API failures and network issues
#
# Requirements: 12.1, 12.2, 12.5

# Initialize logging
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >&2
}

error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" >&2
}

# Validate environment
validate_environment() {
    log "Validating environment..."
    
    # Check if versions.json exists
    if [[ ! -f "versions.json" ]]; then
        error "versions.json not found in current directory"
        exit 1
    fi
    
    # Check if jq is available
    if ! command -v jq >/dev/null 2>&1; then
        error "jq is required but not installed"
        exit 1
    fi
    
    # Check if curl is available
    if ! command -v curl >/dev/null 2>&1; then
        error "curl is required but not installed"
        exit 1
    fi
    
    # Check if GITHUB_OUTPUT is set (for GitHub Actions)
    if [[ -z "${GITHUB_OUTPUT:-}" ]]; then
        log "WARNING: GITHUB_OUTPUT not set, outputs will be printed to stdout"
        GITHUB_OUTPUT="/dev/stdout"
    fi
    
    log "Environment validation completed"
}

# Set GitHub Actions output safely
set_output() {
    local key="$1"
    local value="$2"
    
    log "Setting output: $key=$value"
    echo "$key=$value" >> "$GITHUB_OUTPUT"
}

# Check Node.js version for a specific major version
check_nodejs() {
    local major="$1"
    local current="$2"
    
    log "Checking Node.js $major version (current: $current)"
    
    # Query nodejs.org API with retry logic
    local latest=""
    local retry_count=0
    local max_retries=3
    
    while [[ $retry_count -lt $max_retries ]]; do
        if latest=$(curl -s --max-time 30 https://nodejs.org/dist/index.json 2>/dev/null | \
            jq -r --arg maj "$major" '[.[] | select(.version | startswith("v" + $maj + "."))] | .[0].version' 2>/dev/null | tr -d 'v'); then
            
            # Validate the response
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
        set_output "nodejs-$major-update" "false"
        set_output "nodejs-$major-error" "api-failure"
        return 1
    fi
    
    # Compare versions
    if [[ "$latest" != "$current" ]]; then
        log "🆕 New Node.js $major version detected: $current -> $latest"
        set_output "nodejs-$major" "$latest"
        set_output "nodejs-$major-update" "true"
        set_output "nodejs-$major-current" "$current"
        return 0
    else
        log "✅ Node.js $major version $current is current"
        set_output "nodejs-$major-update" "false"
        return 0
    fi
}

# Check Python version for a specific major.minor version
check_python() {
    local major="$1"
    local current="$2"
    
    log "Checking Python $major version (current: $current)"
    
    # Query endoflife.date API with retry logic
    local latest=""
    local retry_count=0
    local max_retries=3
    
    while [[ $retry_count -lt $max_retries ]]; do
        if latest=$(curl -s --max-time 30 https://endoflife.date/api/python.json 2>/dev/null | \
            jq -r --arg maj "$major" '[.[] | select(.cycle == $maj)] | .[0].latest' 2>/dev/null); then
            
            # Validate the response
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
        set_output "python-$major-update" "false"
        set_output "python-$major-error" "api-failure"
        return 1
    fi
    
    # Compare versions
    if [[ "$latest" != "$current" ]]; then
        log "🆕 New Python $major version detected: $current -> $latest"
        set_output "python-$major" "$latest"
        set_output "python-$major-update" "true"
        set_output "python-$major-current" "$current"
        return 0
    else
        log "✅ Python $major version $current is current"
        set_output "python-$major-update" "false"
        return 0
    fi
}

# Check all tracked versions
check_all_versions() {
    log "Starting upstream version checks..."
    
    local updates_detected=false
    local nodejs_updates=""
    local python_updates=""
    local check_errors=false
    
    echo "🔍 UPSTREAM VERSION CHECK" >&2
    echo "========================" >&2
    
    # Check all tracked Node.js versions
    log "Checking Node.js versions..."
    while IFS= read -r major; do
        if [[ -n "$major" ]]; then
            local current
            current=$(jq -r ".nodejs.\"$major\"" versions.json)
            
            if [[ "$current" == "null" || -z "$current" ]]; then
                error "No current version found for Node.js $major in versions.json"
                check_errors=true
                continue
            fi
            
            log "Current Node.js $major: $current"
            if check_nodejs "$major" "$current"; then
                # Check if update was detected
                local update_detected
                update_detected=$(grep "nodejs-$major-update=true" "$GITHUB_OUTPUT" 2>/dev/null || echo "")
                if [[ -n "$update_detected" ]]; then
                    updates_detected=true
                    nodejs_updates="${nodejs_updates}$major,"
                fi
            else
                check_errors=true
            fi
        fi
    done < <(jq -r '.nodejs | keys[]' versions.json)
    
    # Check all tracked Python versions
    log "Checking Python versions..."
    while IFS= read -r major; do
        if [[ -n "$major" ]]; then
            local current
            current=$(jq -r ".python.\"$major\"" versions.json)
            
            if [[ "$current" == "null" || -z "$current" ]]; then
                error "No current version found for Python $major in versions.json"
                check_errors=true
                continue
            fi
            
            log "Current Python $major: $current"
            if check_python "$major" "$current"; then
                # Check if update was detected
                local update_detected
                update_detected=$(grep "python-$major-update=true" "$GITHUB_OUTPUT" 2>/dev/null || echo "")
                if [[ -n "$update_detected" ]]; then
                    updates_detected=true
                    python_updates="${python_updates}$major,"
                fi
            else
                check_errors=true
            fi
        fi
    done < <(jq -r '.python | keys[]' versions.json)
    
    # Remove trailing commas
    nodejs_updates="${nodejs_updates%,}"
    python_updates="${python_updates%,}"
    
    # Set summary outputs
    set_output "updates-detected" "$updates_detected"
    set_output "nodejs-updates" "$nodejs_updates"
    set_output "python-updates" "$python_updates"
    set_output "check-errors" "$check_errors"
    
    # Log summary
    echo "📊 VERSION CHECK SUMMARY" >&2
    echo "=======================" >&2
    
    if [[ "$updates_detected" == "true" ]]; then
        log "🆕 Updates detected!"
        if [[ -n "$nodejs_updates" ]]; then
            log "  Node.js updates: $nodejs_updates"
        fi
        if [[ -n "$python_updates" ]]; then
            log "  Python updates: $python_updates"
        fi
        log "  Action: Container rebuild will be triggered"
    else
        log "✅ All versions are current"
        log "  Action: No rebuilds needed"
    fi
    
    if [[ "$check_errors" == "true" ]]; then
        log "⚠️  Some version checks failed (see errors above)"
        log "  Action: Manual investigation may be required"
    fi
    
    echo "=======================" >&2
    
    return 0
}

# Update last checked timestamp
update_last_checked() {
    log "Updating last checked timestamp..."
    
    local timestamp
    timestamp=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
    
    # Create temporary file with updated timestamp
    local temp_file
    temp_file=$(mktemp)
    
    if jq --arg timestamp "$timestamp" '.lastChecked = $timestamp' versions.json > "$temp_file"; then
        mv "$temp_file" versions.json
        log "Updated lastChecked to: $timestamp"
        set_output "last-checked" "$timestamp"
    else
        error "Failed to update lastChecked timestamp"
        rm -f "$temp_file"
        return 1
    fi
}

# Main execution
main() {
    echo "🔍 MCP CONTAINER UPSTREAM VERSION CHECK" >&2
    echo "======================================" >&2
    log "Starting upstream version detection for MCP container images..."
    log "Execution environment:"
    log "  Script: $0"
    log "  Working directory: $(pwd)"
    log "  Versions file: versions.json"
    log "  GitHub Actions: ${GITHUB_ACTIONS:-'false'}"
    log "  GitHub Output: ${GITHUB_OUTPUT:-'not set'}"
    log "  Current time: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
    echo "" >&2
    
    # Validate environment first
    validate_environment
    
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
    
    # Check all versions
    if check_all_versions; then
        log "Version checks completed"
    else
        error "Some version checks failed"
    fi
    
    # Update last checked timestamp
    if update_last_checked; then
        log "Timestamp update completed"
    else
        error "Failed to update timestamp"
    fi
    
    echo "======================================" >&2
    log "✅ Upstream version check completed"
    log "Final outputs:"
    log "  updates-detected: $(grep 'updates-detected=' "$GITHUB_OUTPUT" | cut -d'=' -f2 || echo 'not set')"
    log "  nodejs-updates: $(grep 'nodejs-updates=' "$GITHUB_OUTPUT" | cut -d'=' -f2 || echo 'not set')"
    log "  python-updates: $(grep 'python-updates=' "$GITHUB_OUTPUT" | cut -d'=' -f2 || echo 'not set')"
    log "  check-errors: $(grep 'check-errors=' "$GITHUB_OUTPUT" | cut -d'=' -f2 || echo 'not set')"
    log "Execution completed at: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
}

# Execute main function
main "$@"