#!/bin/bash
set -euo pipefail

# Change detection script for MCP container images
# This script detects which container directories have changed and sets GitHub Actions outputs
# for conditional building of container images.
#
# Handles edge cases:
# - Shallow clones and first commits
# - Deleted directories and merge conflicts
# - Missing GITHUB_OUTPUT environment variable
# - Git command failures
#
# Requirements: 4.1, 4.3, 4.4

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
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        error "Not in a git repository"
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

# Get the comparison commit safely
get_comparison_commit() {
    local comparison_commit=""
    
    # Try different strategies to find a comparison commit
    if git rev-parse HEAD~1 >/dev/null 2>&1; then
        comparison_commit="HEAD~1"
        log "Using HEAD~1 as comparison commit"
    elif git rev-parse HEAD^ >/dev/null 2>&1; then
        comparison_commit="HEAD^"
        log "Using HEAD^ as comparison commit"
    elif [[ -n "${GITHUB_BASE_REF:-}" ]] && git rev-parse "origin/${GITHUB_BASE_REF}" >/dev/null 2>&1; then
        comparison_commit="origin/${GITHUB_BASE_REF}"
        log "Using origin/${GITHUB_BASE_REF} as comparison commit (PR base)"
    elif git rev-list --count HEAD >/dev/null 2>&1 && [[ $(git rev-list --count HEAD) -gt 1 ]]; then
        # Try to get the first commit if we have more than one
        comparison_commit=$(git rev-list --max-parents=0 HEAD)
        log "Using first commit $comparison_commit as comparison"
    else
        log "No comparison commit available - this appears to be the first commit or a shallow clone"
        return 1
    fi
    
    echo "$comparison_commit"
    return 0
}

# Get changed files with error handling
get_changed_files() {
    local comparison_commit="$1"
    local changed_files=""
    
    log "Getting changed files between $comparison_commit and HEAD"
    
    # Try git diff with error handling
    if changed_files=$(git diff --name-only "$comparison_commit" HEAD 2>&1); then
        # Check if the result contains error messages
        if [[ "$changed_files" == *"fatal:"* ]] || [[ "$changed_files" == *"error:"* ]]; then
            error "Git diff returned error: $changed_files"
            return 1
        fi
        log "Successfully got git diff between $comparison_commit and HEAD"
    else
        error "Failed to get git diff between $comparison_commit and HEAD"
        
        # Try alternative approach for merge scenarios
        if changed_files=$(git diff --name-only "$comparison_commit"...HEAD 2>&1); then
            if [[ "$changed_files" == *"fatal:"* ]] || [[ "$changed_files" == *"error:"* ]]; then
                error "Git three-dot diff returned error: $changed_files"
                return 1
            fi
            log "Successfully used three-dot diff syntax for merge scenario"
        else
            error "Failed to get git diff with three-dot syntax"
            return 1
        fi
    fi
    
    # Handle empty diff (no changes)
    if [[ -z "$changed_files" ]]; then
        log "No files changed between $comparison_commit and HEAD"
        return 2
    fi
    
    echo "$changed_files"
    return 0
}

# Log detected changes
log_detected_changes() {
    echo "🔍 CHANGE DETECTION SUMMARY" >&2
    echo "=========================" >&2
    echo "Repository: $(git remote get-url origin 2>/dev/null || echo 'local')" >&2
    echo "Branch: $(git branch --show-current 2>/dev/null || echo 'unknown')" >&2
    echo "Commit: $(git rev-parse HEAD 2>/dev/null || echo 'unknown')" >&2
    echo "Comparison: $comparison_commit -> HEAD" >&2
    echo "Total changed files: $(echo "$changed_files" | wc -l)" >&2
    echo "" >&2
}

# Analyze changed files and determine what to build
analyze_changes() {
    local changed_files="$1"
    
    log "Analyzing changed files..."
    log_detected_changes
    
    log "Changed files:"
    echo "$changed_files" | while IFS= read -r file; do
        log "  - $file"
    done
    
    # Detect changes in runtime directories
    local nodejs_changed=""
    local python_changed=""
    local common_changed=""
    
    nodejs_changed=$(echo "$changed_files" | grep "^nodejs/" || true)
    python_changed=$(echo "$changed_files" | grep "^python/" || true)
    
    # Handle common files that affect all containers
    # Include workflows, scripts, root config files, and documentation
    common_changed=$(echo "$changed_files" | grep -E "^(\.github/workflows/|scripts/|README\.md|CONTRIBUTING\.md|VERSIONING\.md|versions\.json)" || true)
    
    # Log analysis results with detailed breakdown
    echo "📊 CHANGE ANALYSIS BREAKDOWN" >&2
    echo "============================" >&2
    
    if [[ -n "$nodejs_changed" ]]; then
        log "✅ Node.js directory changes detected:"
        echo "$nodejs_changed" | while IFS= read -r file; do
            log "  📄 $file"
        done
        echo "" >&2
    else
        log "⏭️  No Node.js directory changes detected" >&2
        echo "" >&2
    fi
    
    if [[ -n "$python_changed" ]]; then
        log "✅ Python directory changes detected:"
        echo "$python_changed" | while IFS= read -r file; do
            log "  📄 $file"
        done
        echo "" >&2
    else
        log "⏭️  No Python directory changes detected" >&2
        echo "" >&2
    fi
    
    if [[ -n "$common_changed" ]]; then
        log "🔄 Common files changes detected (affects ALL containers):"
        echo "$common_changed" | while IFS= read -r file; do
            log "  📄 $file"
        done
        echo "" >&2
    else
        log "⏭️  No common file changes detected" >&2
        echo "" >&2
    fi
    
    # Determine build decisions with detailed reasoning
    echo "🚀 BUILD DECISION MATRIX" >&2
    echo "=======================" >&2
    
    if [[ -n "$common_changed" ]]; then
        log "🔄 DECISION: Common files changed - building ALL containers"
        log "   Reason: Changes to workflows, scripts, or documentation affect all containers"
        log "   Action: Both Node.js and Python containers will be built"
        set_output "nodejs-changed" "true"
        set_output "python-changed" "true"
        set_output "build-reason" "common-files-changed"
    else
        local build_reason=""
        local containers_to_build=""
        
        if [[ -n "$nodejs_changed" ]]; then
            log "✅ DECISION: Building Node.js container"
            log "   Reason: Node.js-specific files were modified"
            log "   Files: $(echo "$nodejs_changed" | tr '\n' ' ')"
            set_output "nodejs-changed" "true"
            build_reason="${build_reason}nodejs-files-changed,"
            containers_to_build="${containers_to_build}Node.js "
        else
            log "⏭️  DECISION: Skipping Node.js container (no relevant changes)"
            set_output "nodejs-changed" "false"
        fi
        
        if [[ -n "$python_changed" ]]; then
            log "✅ DECISION: Building Python container"
            log "   Reason: Python-specific files were modified"
            log "   Files: $(echo "$python_changed" | tr '\n' ' ')"
            set_output "python-changed" "true"
            build_reason="${build_reason}python-files-changed,"
            containers_to_build="${containers_to_build}Python "
        else
            log "⏭️  DECISION: Skipping Python container (no relevant changes)"
            set_output "python-changed" "false"
        fi
        
        # Remove trailing comma and set build reason
        build_reason="${build_reason%,}"
        if [[ -z "$build_reason" ]]; then
            build_reason="no-relevant-changes"
            log "⏭️  DECISION: No containers will be built (no relevant changes detected)"
        else
            log "📦 SUMMARY: Building containers: ${containers_to_build}"
        fi
        set_output "build-reason" "$build_reason"
    fi
    
    echo "=======================" >&2
    log "Build decision completed successfully"
}

# Main execution
main() {
    echo "🔍 MCP CONTAINER CHANGE DETECTION" >&2
    echo "=================================" >&2
    log "Starting change detection for MCP container images..."
    log "Execution environment:"
    log "  Script: $0"
    log "  Working directory: $(pwd)"
    log "  Git repository: $(git remote get-url origin 2>/dev/null || echo 'local repository')"
    log "  Current branch: $(git branch --show-current 2>/dev/null || echo 'unknown')"
    log "  Current commit: $(git rev-parse HEAD 2>/dev/null || echo 'unknown')"
    log "  GitHub Actions: ${GITHUB_ACTIONS:-'false'}"
    log "  GitHub Output: ${GITHUB_OUTPUT:-'not set'}"
    echo "" >&2
    
    # Validate environment first
    validate_environment
    
    # Try to get comparison commit
    local comparison_commit=""
    if ! comparison_commit=$(get_comparison_commit); then
        log "⚠️  No comparison commit available - building all containers as fallback"
        log "   This typically happens on:"
        log "   - First commit in repository"
        log "   - Shallow clone (common in CI)"
        log "   - Repository with single commit"
        set_output "nodejs-changed" "true"
        set_output "python-changed" "true"
        set_output "build-reason" "first-commit-or-shallow-clone"
        echo "=================================" >&2
        log "✅ Change detection completed (fallback mode)"
        return 0
    fi
    
    # Get changed files
    local changed_files=""
    local get_files_result=0
    changed_files=$(get_changed_files "$comparison_commit") || get_files_result=$?
    
    case $get_files_result in
        0)
            # Success - analyze changes
            analyze_changes "$changed_files"
            ;;
        1)
            # Git diff failed - build all as fallback
            error "Git diff failed - building all containers as fallback"
            log "   This can happen due to:"
            log "   - Git repository corruption"
            log "   - Network issues in CI"
            log "   - Merge conflicts"
            set_output "nodejs-changed" "true"
            set_output "python-changed" "true"
            set_output "build-reason" "git-diff-failed"
            ;;
        2)
            # No changes detected
            log "📭 No files changed - skipping all builds"
            log "   Comparison: $comparison_commit -> HEAD"
            log "   Result: No container builds needed"
            set_output "nodejs-changed" "false"
            set_output "python-changed" "false"
            set_output "build-reason" "no-changes"
            ;;
        *)
            # Unknown error - build all as fallback
            error "Unknown error in change detection - building all containers as fallback"
            log "   Exit code: $get_files_result"
            log "   Fallback action: Build all containers"
            set_output "nodejs-changed" "true"
            set_output "python-changed" "true"
            set_output "build-reason" "unknown-error"
            ;;
    esac
    
    echo "=================================" >&2
    log "✅ Change detection completed successfully"
    log "Final outputs:"
    log "  nodejs-changed: $(grep 'nodejs-changed=' "$GITHUB_OUTPUT" | cut -d'=' -f2 || echo 'not set')"
    log "  python-changed: $(grep 'python-changed=' "$GITHUB_OUTPUT" | cut -d'=' -f2 || echo 'not set')"
    log "  build-reason: $(grep 'build-reason=' "$GITHUB_OUTPUT" | cut -d'=' -f2 || echo 'not set')"
    log "Execution completed at: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
}

# Execute main function
main "$@"