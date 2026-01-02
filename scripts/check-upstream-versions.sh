#!/bin/bash
set -euo pipefail

# Dynamic upstream version detection script for MCP container images
# Follows the rule: Support Current LTS + Previous LTS only
# No hardcoded version lists - determines supported versions dynamically

# Container runtime detection
DOCKER_CMD="${DOCKER_CMD:-$(
    if command -v docker >/dev/null 2>&1; then
        echo "docker"
    elif command -v podman >/dev/null 2>&1; then
        echo "podman"
    elif command -v nerdctl >/dev/null 2>&1; then
        echo "nerdctl"
    elif command -v finch >/dev/null 2>&1; then
        echo "finch"
    else
        echo ""
    fi
)}"

# Registry configuration (matching Makefile)
REGISTRY="${REGISTRY:-ghcr.io}"
OWNER="${OWNER:-$(git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\).*/\1/' 2>/dev/null || echo "")}"
REPO_NAME="${REPO_NAME:-$(basename $(git rev-parse --show-toplevel 2>/dev/null) 2>/dev/null || echo "mcp-container-images")}"

NODEJS_IMAGE="${NODEJS_IMAGE:-$REGISTRY/$OWNER/$REPO_NAME-nodejs}"
PYTHON_IMAGE="${PYTHON_IMAGE:-$REGISTRY/$OWNER/$REPO_NAME-python}"

# Initialize logging
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >&2
}

error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" >&2
}

# Set GitHub Actions output safely
set_output() {
    local key="$1"
    local value="$2"
    
    if [[ -n "${GITHUB_OUTPUT:-}" && "$GITHUB_OUTPUT" != "/dev/stdout" ]]; then
        echo "$key=$value" >> "$GITHUB_OUTPUT"
    else
        # When not in GitHub Actions, send to stderr to keep stdout clean for JSON
        echo "$key=$value" >&2
    fi
}

# Get what we've published to the registry for a runtime/major version
get_built_version() {
    local image="$1"
    local major="$2"
    local runtime="$3"
    
    # Extract owner and repo from image name
    if [[ "$image" =~ ^ghcr\.io/([^/]+)/([^:]+) ]]; then
        local owner="${BASH_REMATCH[1]}"
        local repo="${BASH_REMATCH[2]}"
        
        # Try GitHub API first (works for public packages)
        local github_api_url="https://api.github.com/users/${owner}/packages/container/${repo}/versions"
        
        # Get published versions from GitHub API
        local published_versions
        published_versions=$(curl -s --max-time 10 "$github_api_url" 2>/dev/null | jq -r '.[].metadata.container.tags[]?' 2>/dev/null)
        
        # If API call fails or returns empty, assume nothing is published yet
        if [[ -z "$published_versions" || "$published_versions" == "null" ]]; then
            log "No published versions found for $repo (package may not exist yet)"
            echo ""
            return
        fi
        
        # Look for specific version tags in published versions
        local built_version=""
        if [[ "$runtime" == "nodejs" ]]; then
            # Look for node22.x.x pattern in published tags
            built_version=$(echo "$published_versions" | grep -E "^node${major}\.[0-9]+\.[0-9]+$" | head -1 | sed "s/^node//")
        elif [[ "$runtime" == "python" ]]; then
            # Look for python3.12.x pattern in published tags
            built_version=$(echo "$published_versions" | grep -E "^python${major}\.[0-9]+$" | head -1 | sed "s/^python//")
        fi
        
        echo "$built_version"
    else
        echo ""
    fi
}

# Get supported Node.js versions (Current LTS + Previous LTS)
get_supported_nodejs_versions() {
    log "Determining supported Node.js versions (Current + Previous 2 LTS)..."
    
    # Get all Node.js versions with LTS info
    local nodejs_data
    nodejs_data=$(curl -s --max-time 30 https://nodejs.org/dist/index.json 2>/dev/null)
    
    if [[ -z "$nodejs_data" ]]; then
        error "Failed to fetch Node.js version data"
        return 1
    fi
    
    # Get unique LTS major versions, sorted numerically descending, take first 3
    local lts_versions
    lts_versions=$(echo "$nodejs_data" | jq -r '[.[] | select(.lts != false) | .version] | map(ltrimstr("v") | split(".")[0] | tonumber) | unique | sort | reverse | .[0:3] | map(tostring) | .[]' | tr '\n' ' ')
    
    if [[ -z "$lts_versions" ]]; then
        error "Could not determine LTS versions"
        return 1
    fi
    
    log "Node.js LTS versions: $lts_versions"
    echo "$lts_versions"
}

# Get supported Python versions (Current + Previous maintained)
get_supported_python_versions() {
    log "Determining supported Python versions (Current + Previous 2 maintained)..."
    
    # Get Python version data from endoflife.date
    local python_data
    python_data=$(curl -s --max-time 30 https://endoflife.date/api/python.json 2>/dev/null)
    
    if [[ -z "$python_data" ]]; then
        error "Failed to fetch Python version data"
        return 1
    fi
    
    # Get versions that are released (release date is in the past) and have a valid latest version
    local supported_versions=""
    local count=0
    
    while IFS= read -r version_info; do
        if [[ $count -ge 3 ]]; then  # Changed from 2 to 3 for Python
            break
        fi
        
        local cycle release_date latest
        cycle=$(echo "$version_info" | jq -r '.cycle')
        release_date=$(echo "$version_info" | jq -r '.releaseDate')
        latest=$(echo "$version_info" | jq -r '.latest')
        
        # Check if release date is in the past and latest version exists
        if [[ -n "$latest" && "$latest" != "null" && "$latest" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            # Verify we can actually get this version from the API
            local test_latest
            test_latest=$(get_python_latest "$cycle")
            
            if [[ -n "$test_latest" && "$test_latest" != "" ]]; then
                log "Found valid Python version: $cycle (latest: $latest)"
                supported_versions="${supported_versions}$cycle "
                ((count++))
            else
                log "Skipping Python $cycle - cannot retrieve latest version"
            fi
        else
            log "Skipping Python $cycle - invalid or missing latest version"
        fi
    done < <(echo "$python_data" | jq -c '.[]')
    
    if [[ -z "$supported_versions" ]]; then
        error "Could not determine supported Python versions"
        return 1
    fi
    
    log "Python supported versions: $supported_versions"
    echo "$supported_versions"
}

# Get latest upstream version for Node.js major version
get_nodejs_latest() {
    local major="$1"
    
    curl -s --max-time 30 https://nodejs.org/dist/index.json 2>/dev/null | \
        jq -r --arg maj "$major" '[.[] | select(.version | startswith("v" + $maj + "."))] | .[0].version' 2>/dev/null | \
        tr -d 'v' || echo ""
}

# Get latest upstream version for Python major.minor version
get_python_latest() {
    local major="$1"
    
    curl -s --max-time 30 https://endoflife.date/api/python.json 2>/dev/null | \
        jq -r --arg maj "$major" '[.[] | select(.cycle == $maj)] | .[0].latest' 2>/dev/null || echo ""
}

# Check what Node.js versions we need to build
check_nodejs_versions() {
    local updates_needed=""
    
    log "Checking Node.js versions..."
    
    # Get supported versions dynamically
    local supported_versions
    if ! supported_versions=$(get_supported_nodejs_versions); then
        log "⚠️  Could not determine supported Node.js versions - skipping Node.js checks"
        echo ""
        return 0
    fi
    
    if [[ -z "$supported_versions" ]]; then
        log "⚠️  No supported Node.js versions found"
        echo ""
        return 0
    fi
    
    for major in $supported_versions; do
        local built_version latest_version
        
        built_version=$(get_built_version "$NODEJS_IMAGE" "$major" "nodejs")
        latest_version=$(get_nodejs_latest "$major")
        
        if [[ -z "$latest_version" ]]; then
            log "⚠️  Could not get latest Node.js $major version"
            continue
        fi
        
        if [[ -z "$built_version" ]]; then
            log "📦 Node.js $major: Not built locally, latest is $latest_version"
            updates_needed="${updates_needed}$major,"
            set_output "nodejs-$major" "$latest_version"
            set_output "nodejs-$major-status" "not-built"
        elif [[ "$built_version" != "$latest_version" ]]; then
            log "🆕 Node.js $major: Built $built_version, latest is $latest_version"
            updates_needed="${updates_needed}$major,"
            set_output "nodejs-$major" "$latest_version"
            set_output "nodejs-$major-current" "$built_version"
            set_output "nodejs-$major-status" "outdated"
        else
            log "✅ Node.js $major: Built $built_version is current"
            set_output "nodejs-$major-status" "current"
        fi
    done
    
    # Remove trailing comma
    updates_needed="${updates_needed%,}"
    echo "$updates_needed"
}

# Check what Python versions we need to build
check_python_versions() {
    local updates_needed=""
    
    log "Checking Python versions..."
    
    # Get supported versions dynamically
    local supported_versions
    if ! supported_versions=$(get_supported_python_versions); then
        log "⚠️  Could not determine supported Python versions - skipping Python checks"
        echo ""
        return 0
    fi
    
    if [[ -z "$supported_versions" ]]; then
        log "⚠️  No supported Python versions found"
        echo ""
        return 0
    fi
    
    for major in $supported_versions; do
        local built_version latest_version
        
        built_version=$(get_built_version "$PYTHON_IMAGE" "$major" "python")
        latest_version=$(get_python_latest "$major")
        
        if [[ -z "$latest_version" ]]; then
            log "⚠️  Could not get latest Python $major version"
            continue
        fi
        
        if [[ -z "$built_version" ]]; then
            log "📦 Python $major: Not built locally, latest is $latest_version"
            updates_needed="${updates_needed}$major,"
            set_output "python-$major" "$latest_version"
            set_output "python-$major-status" "not-built"
        elif [[ "$built_version" != "$latest_version" ]]; then
            log "🆕 Python $major: Built $built_version, latest is $latest_version"
            updates_needed="${updates_needed}$major,"
            set_output "python-$major" "$latest_version"
            set_output "python-$major-current" "$built_version"
            set_output "python-$major-status" "outdated"
        else
            log "✅ Python $major: Built $built_version is current"
            set_output "python-$major-status" "current"
        fi
    done
    
    # Remove trailing comma
    updates_needed="${updates_needed%,}"
    echo "$updates_needed"
}

# Main execution
main() {
    echo "🔍 MCP CONTAINER VERSION CHECK" >&2
    echo "=============================" >&2
    log "Dynamically checking versions based on LTS support policy..."
    log "Rule: Support Current + Previous 2 LTS versions"
    log "Container runtime: ${DOCKER_CMD:-'none available'}"
    log "Node.js image: $NODEJS_IMAGE"
    log "Python image: $PYTHON_IMAGE"
    echo "" >&2
    
    # Check versions
    local nodejs_updates python_updates
    
    nodejs_updates=$(check_nodejs_versions)
    python_updates=$(check_python_versions)
    
    # Get all supported versions (not just updates)
    local nodejs_supported python_supported
    if ! nodejs_supported=$(get_supported_nodejs_versions); then
        nodejs_supported=""
    fi
    if ! python_supported=$(get_supported_python_versions); then
        python_supported=""
    fi
    
    # Set summary outputs for GitHub Actions
    if [[ -n "$nodejs_updates" || -n "$python_updates" ]]; then
        set_output "updates-detected" "true"
        log "📋 SUMMARY: Updates needed"
    else
        set_output "updates-detected" "false"
        log "📋 SUMMARY: All supported versions current"
    fi
    
    set_output "nodejs-updates" "$nodejs_updates"
    set_output "python-updates" "$python_updates"
    
    # Create JSON output for Makefile consumption
    local nodejs_versions_json python_versions_json
    local nodejs_updates_json python_updates_json
    local nodejs_latest_json python_latest_json
    
    # Build Node.js arrays and objects
    if [[ -n "$nodejs_supported" ]]; then
        nodejs_versions_json=$(echo "$nodejs_supported" | tr ' ' '\n' | grep -v '^$' | jq -R . | jq -s .)
    else
        nodejs_versions_json='[]'
    fi
    
    if [[ -n "$nodejs_updates" ]]; then
        nodejs_updates_json=$(echo "$nodejs_updates" | tr ',' '\n' | grep -v '^$' | jq -R . | jq -s .)
    else
        nodejs_updates_json='[]'
    fi
    
    nodejs_latest_json="{"
    local first=true
    for version in $nodejs_supported; do
        if [[ "$first" == "false" ]]; then
            nodejs_latest_json="$nodejs_latest_json,"
        fi
        latest=$(get_nodejs_latest "$version")
        nodejs_latest_json="$nodejs_latest_json\"$version\":\"$latest\""
        first=false
    done
    nodejs_latest_json="$nodejs_latest_json}"
    
    # Build Python arrays and objects
    if [[ -n "$python_supported" ]]; then
        python_versions_json=$(echo "$python_supported" | tr ' ' '\n' | grep -v '^$' | jq -R . | jq -s .)
    else
        python_versions_json='[]'
    fi
    
    if [[ -n "$python_updates" ]]; then
        python_updates_json=$(echo "$python_updates" | tr ',' '\n' | grep -v '^$' | jq -R . | jq -s .)
    else
        python_updates_json='[]'
    fi
    
    python_latest_json="{"
    first=true
    for version in $python_supported; do
        if [[ "$first" == "false" ]]; then
            python_latest_json="$python_latest_json,"
        fi
        latest=$(get_python_latest "$version")
        python_latest_json="$python_latest_json\"$version\":\"$latest\""
        first=false
    done
    python_latest_json="$python_latest_json}"
    
    # Create final JSON
    local updates_detected
    if [[ -n "$nodejs_updates" || -n "$python_updates" ]]; then
        updates_detected="true"
    else
        updates_detected="false"
    fi
    
    local total_supported
    total_supported=$(($(echo "$nodejs_supported" | wc -w) + $(echo "$python_supported" | wc -w)))
    
    local json_output
    json_output=$(jq -n \
        --argjson nodejs_versions "$nodejs_versions_json" \
        --argjson nodejs_updates "$nodejs_updates_json" \
        --argjson nodejs_latest "$nodejs_latest_json" \
        --argjson python_versions "$python_versions_json" \
        --argjson python_updates "$python_updates_json" \
        --argjson python_latest "$python_latest_json" \
        --arg updates_detected "$updates_detected" \
        --arg total_supported "$total_supported" \
        '{
            nodejs: {
                supported_versions: $nodejs_versions,
                versions_to_build: $nodejs_updates,
                latest_versions: $nodejs_latest
            },
            python: {
                supported_versions: $python_versions,
                versions_to_build: $python_updates,
                latest_versions: $python_latest
            },
            summary: {
                updates_detected: ($updates_detected == "true"),
                total_supported: ($total_supported | tonumber)
            }
        }')
    
    # Output JSON to stdout for Makefile consumption
    echo "$json_output"
    
    # Show summary to stderr
    echo "📋 BUILD SUMMARY" >&2
    echo "===============" >&2
    
    if [[ -n "$nodejs_updates" ]]; then
        log "Node.js versions to build: $nodejs_updates"
    else
        log "Node.js: All supported versions current"
    fi
    
    if [[ -n "$python_updates" ]]; then
        log "Python versions to build: $python_updates"
    else
        log "Python: All supported versions current"
    fi
    
    echo "=============================" >&2
    log "✅ Version check completed"
}

# Execute main function
main "$@"
exit 0