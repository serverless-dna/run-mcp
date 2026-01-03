#!/bin/bash
set -euo pipefail

# Generate changelog entry from git commits and PR information
# Usage: ./scripts/generate-changelog.sh [version] [previous-version]

VERSION=${1:-$(git describe --tags --abbrev=0 2>/dev/null || echo "unreleased")}
PREVIOUS_VERSION=${2:-$(git describe --tags --abbrev=0 HEAD~1 2>/dev/null || echo "")}
DATE=$(date +%Y-%m-%d)

echo "Generating changelog for version $VERSION (since $PREVIOUS_VERSION)"

# Function to categorize commits
categorize_commit() {
    local commit_msg="$1"
    local files_changed="$2"
    
    # Check commit message patterns
    if [[ $commit_msg =~ ^feat(\(.+\))?: ]]; then
        echo "Added"
    elif [[ $commit_msg =~ ^fix(\(.+\))?: ]]; then
        echo "Fixed"
    elif [[ $commit_msg =~ ^docs(\(.+\))?: ]]; then
        echo "Documentation"
    elif [[ $commit_msg =~ ^test(\(.+\))?: ]]; then
        echo "Testing"
    elif [[ $commit_msg =~ ^chore(\(.+\))?: ]]; then
        echo "Maintenance"
    elif [[ $commit_msg =~ ^refactor(\(.+\))?: ]]; then
        echo "Changed"
    elif [[ $commit_msg =~ ^perf(\(.+\))?: ]]; then
        echo "Performance"
    elif [[ $commit_msg =~ ^build(\(.+\))?: ]]; then
        echo "Build"
    elif [[ $commit_msg =~ ^ci(\(.+\))?: ]]; then
        echo "CI/CD"
    # Check file patterns if conventional commit format not used
    elif [[ $files_changed =~ Dockerfile|docker-compose ]]; then
        echo "Container Images"
    elif [[ $files_changed =~ cmd/run-mcp ]]; then
        echo "Binary"
    elif [[ $files_changed =~ \.github/workflows ]]; then
        echo "CI/CD"
    elif [[ $files_changed =~ tests/ ]]; then
        echo "Testing"
    elif [[ $files_changed =~ scripts/ ]]; then
        echo "Scripts"
    elif [[ $files_changed =~ README|CONTRIBUTING|docs/ ]]; then
        echo "Documentation"
    else
        echo "Changed"
    fi
}

# Generate commit range
if [[ -n "$PREVIOUS_VERSION" ]]; then
    COMMIT_RANGE="${PREVIOUS_VERSION}..HEAD"
else
    COMMIT_RANGE="HEAD"
fi

echo "## [$VERSION] - $DATE"
echo ""

# Collect commits by category
declare -A categories
declare -A commit_lists

# Get commits with file changes
while IFS='|' read -r hash subject files; do
    category=$(categorize_commit "$subject" "$files")
    
    # Clean up subject (remove conventional commit prefixes)
    clean_subject=$(echo "$subject" | sed -E 's/^(feat|fix|docs|test|chore|refactor|perf|build|ci)(\(.+\))?: //')
    
    # Add to category
    if [[ -z "${commit_lists[$category]:-}" ]]; then
        commit_lists[$category]="- $clean_subject"
    else
        commit_lists[$category]="${commit_lists[$category]}"$'\n'"- $clean_subject"
    fi
    categories[$category]=1
done < <(git log --format="%H|%s|%s" --name-only "$COMMIT_RANGE" | \
         awk '/^[a-f0-9]{40}\|/ {
             if (NR > 1) print hash "|" subject "|" files
             split($0, parts, "|")
             hash = parts[1]
             subject = parts[2]
             files = ""
         }
         /^[^a-f0-9]/ && !/^\|/ {
             if (files) files = files " "
             files = files $0
         }
         END { if (hash) print hash "|" subject "|" files }')

# Output in standard changelog order
changelog_order=("Added" "Changed" "Fixed" "Deprecated" "Removed" "Security" "Performance" "Container Images" "Binary" "Documentation" "Testing" "Scripts" "Build" "CI/CD" "Maintenance")

for category in "${changelog_order[@]}"; do
    if [[ -n "${categories[$category]:-}" ]]; then
        echo "### $category"
        echo "${commit_lists[$category]}"
        echo ""
    fi
done

# Add container image information if relevant
if git diff --name-only "$COMMIT_RANGE" | grep -q "Dockerfile\|docker-compose"; then
    echo "### Container Images"
    echo "- Updated base images and dependencies"
    echo "- Multi-architecture builds (AMD64/ARM64)"
    echo ""
fi

# Add binary information if relevant  
if git diff --name-only "$COMMIT_RANGE" | grep -q "cmd/run-mcp"; then
    echo "### Binary Releases"
    echo "- Windows (AMD64): run-mcp-windows-amd64.exe"
    echo "- macOS (Intel): run-mcp-darwin-amd64"
    echo "- macOS (Apple Silicon): run-mcp-darwin-arm64"
    echo "- Linux (AMD64): run-mcp-linux-amd64"
    echo "- Linux (ARM64): run-mcp-linux-arm64"
    echo ""
fi

# Add link
if [[ "$VERSION" != "unreleased" ]]; then
    echo "[$VERSION]: https://github.com/modelcontextprotocol/mcp-container-images/releases/tag/v$VERSION"
fi