#!/bin/bash
set -euo pipefail

# Version retention cleanup script for MCP container images
# Implements the retention policy defined in VERSIONING.md

REGISTRY="ghcr.io"
REPO_OWNER="${GITHUB_REPOSITORY_OWNER:-}"
IMAGE_NAME="${GITHUB_REPOSITORY:-}"

# Retention limits from VERSIONING.md
MAX_MAJOR_VERSIONS=2
MAX_MINOR_VERSIONS=3
MAX_PATCH_VERSIONS=5
MAX_DATE_TAGS=10
DEV_TAG_RETENTION_DAYS=30

log() {
    echo "[$(date -u '+%Y-%m-%d %H:%M:%S UTC')] $*"
}

error() {
    echo "[$(date -u '+%Y-%m-%d %H:%M:%S UTC')] ERROR: $*" >&2
}

warning() {
    echo "[$(date -u '+%Y-%m-%d %H:%M:%S UTC')] WARNING: $*" >&2
}

# Function to get all versions for a package
get_package_versions() {
    local package_name="$1"
    
    log "Fetching versions for package: $package_name"
    
    gh api "/users/${REPO_OWNER}/packages/container/${package_name}/versions" \
        --jq 'sort_by(.created_at) | reverse | .[] | {id: .id, tags: .metadata.container.tags, created_at: .created_at}' \
        2>/dev/null || {
        error "Failed to fetch versions for package: $package_name"
        return 1
    }
}

# Function to delete a package version by ID
delete_version() {
    local package_name="$1"
    local version_id="$2"
    local tag_name="$3"
    
    log "Deleting version: $tag_name (ID: $version_id)"
    
    gh api -X DELETE "/users/${REPO_OWNER}/packages/container/${package_name}/versions/${version_id}" \
        2>/dev/null || {
        warning "Failed to delete version: $tag_name (ID: $version_id)"
        return 1
    }
    
    log "Successfully deleted version: $tag_name"
}

# Function to cleanup date-stamped tags
cleanup_date_tags() {
    local package_name="$1"
    local versions="$2"
    local tag_pattern="$3"  # e.g., '^node[0-9]+\.[0-9]+\.[0-9]+-[0-9]{8}$'
    
    log "Cleaning up date-stamped tags for $package_name (keeping latest $MAX_DATE_TAGS per version)"
    
    # Get date-stamped tags matching the pattern
    local date_tags
    date_tags=$(echo "$versions" | jq -r '.tags[]?' | grep -E "$tag_pattern" | sort -r || true)
    
    if [[ -z "$date_tags" ]]; then
        log "No date-stamped tags found for cleanup"
        return 0
    fi
    
    # Group by version prefix and keep only latest MAX_DATE_TAGS per group
    local version_prefixes
    version_prefixes=$(echo "$date_tags" | sed 's/-[0-9]\{8\}$//' | sort -u)
    
    for version_prefix in $version_prefixes; do
        log "Processing version prefix: $version_prefix"
        
        # Get tags for this version, sorted by date (newest first)
        local version_date_tags
        version_date_tags=$(echo "$date_tags" | grep "^${version_prefix}-" | head -20)
        
        # Count tags for this version
        local tag_count
        tag_count=$(echo "$version_date_tags" | wc -l)
        
        if [[ $tag_count -gt $MAX_DATE_TAGS ]]; then
            # Get tags to delete (beyond the first MAX_DATE_TAGS)
            local tags_to_delete
            tags_to_delete=$(echo "$version_date_tags" | tail -n +$((MAX_DATE_TAGS + 1)))
            
            log "Found $tag_count date-stamped tags for $version_prefix, deleting oldest $((tag_count - MAX_DATE_TAGS))"
            
            for tag in $tags_to_delete; do
                # Find version ID for this tag
                local version_id
                version_id=$(echo "$versions" | jq -r --arg tag "$tag" 'select(.tags[]? == $tag) | .id' | head -1)
                
                if [[ -n "$version_id" && "$version_id" != "null" ]]; then
                    delete_version "$package_name" "$version_id" "$tag"
                else
                    warning "Could not find version ID for tag: $tag"
                fi
            done
        else
            log "Version $version_prefix has $tag_count date-stamped tags (within limit of $MAX_DATE_TAGS)"
        fi
    done
}

# Function to cleanup development tags (pr-*, main, etc.)
cleanup_dev_tags() {
    local package_name="$1"
    local versions="$2"
    
    log "Cleaning up development tags older than $DEV_TAG_RETENTION_DAYS days"
    
    # Calculate cutoff date
    local cutoff_date
    cutoff_date=$(date -u -d "$DEV_TAG_RETENTION_DAYS days ago" '+%Y-%m-%dT%H:%M:%SZ')
    
    # Get development tags (pr-*, main, branch names)
    local dev_tags
    dev_tags=$(echo "$versions" | jq -r --arg cutoff "$cutoff_date" '
        select(.created_at < $cutoff) | 
        select(.tags[]? | test("^(pr-[0-9]+|main|[a-zA-Z0-9_-]+)$")) |
        select(.tags[]? | test("^(latest|node|python|lts)") | not) |
        {id: .id, tags: .tags, created_at: .created_at}
    ' || true)
    
    if [[ -z "$dev_tags" ]]; then
        log "No old development tags found for cleanup"
        return 0
    fi
    
    echo "$dev_tags" | jq -r '.id' | while read -r version_id; do
        if [[ -n "$version_id" && "$version_id" != "null" ]]; then
            local tag_names
            tag_names=$(echo "$dev_tags" | jq -r --arg id "$version_id" 'select(.id == $id) | .tags[]?' | tr '\n' ',' | sed 's/,$//')
            delete_version "$package_name" "$version_id" "$tag_names"
        fi
    done
}

# Function to cleanup a specific container package
cleanup_package() {
    local package_name="$1"
    local tag_pattern="$2"
    
    log "Starting cleanup for package: $package_name"
    
    # Get all versions for the package
    local versions
    versions=$(get_package_versions "$package_name") || {
        error "Failed to get versions for package: $package_name"
        return 1
    }
    
    if [[ -z "$versions" ]]; then
        log "No versions found for package: $package_name"
        return 0
    fi
    
    log "Found $(echo "$versions" | jq -s 'length') versions for $package_name"
    
    # Cleanup date-stamped tags
    cleanup_date_tags "$package_name" "$versions" "$tag_pattern"
    
    # Cleanup development tags
    cleanup_dev_tags "$package_name" "$versions"
    
    log "Completed cleanup for package: $package_name"
}

# Main execution
main() {
    log "Starting version retention cleanup"
    
    # Validate required environment variables
    if [[ -z "$REPO_OWNER" ]]; then
        error "GITHUB_REPOSITORY_OWNER environment variable is required"
        exit 1
    fi
    
    if [[ -z "$IMAGE_NAME" ]]; then
        error "GITHUB_REPOSITORY environment variable is required"
        exit 1
    fi
    
    # Check if GitHub CLI is authenticated
    if ! gh auth status >/dev/null 2>&1; then
        error "GitHub CLI is not authenticated"
        exit 1
    fi
    
    log "Repository: $IMAGE_NAME"
    log "Owner: $REPO_OWNER"
    log "Retention limits: Major=$MAX_MAJOR_VERSIONS, Minor=$MAX_MINOR_VERSIONS, Patch=$MAX_PATCH_VERSIONS, Date=$MAX_DATE_TAGS"
    
    # Cleanup Node.js container
    if cleanup_package "${IMAGE_NAME}-nodejs" '^node[0-9]+\.[0-9]+\.[0-9]+-[0-9]{8}$'; then
        log "✅ Node.js container cleanup completed successfully"
    else
        warning "Node.js container cleanup encountered issues"
    fi
    
    # Cleanup Python container
    if cleanup_package "${IMAGE_NAME}-python" '^python[0-9]+\.[0-9]+\.[0-9]+-[0-9]{8}$'; then
        log "✅ Python container cleanup completed successfully"
    else
        warning "Python container cleanup encountered issues"
    fi
    
    log "Version retention cleanup completed"
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi