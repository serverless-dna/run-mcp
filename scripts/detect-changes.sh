#!/bin/bash
set -euo pipefail

# Change detection script for MCP container images
# This script detects which container directories have changed and sets GitHub Actions outputs
# for conditional building of container images.

echo "Starting change detection..."

# Handle shallow clones and first commits
if ! git rev-parse HEAD~1 >/dev/null 2>&1; then
    echo "First commit or shallow clone detected, building all containers"
    echo "nodejs-changed=true" >> $GITHUB_OUTPUT
    echo "python-changed=true" >> $GITHUB_OUTPUT
    exit 0
fi

# Get list of changed files between HEAD and previous commit
CHANGED_FILES=$(git diff --name-only HEAD~1 HEAD)
echo "Changed files: $CHANGED_FILES"

# Detect changes in runtime directories
NODEJS_CHANGED=$(echo "$CHANGED_FILES" | grep "^nodejs/" || echo "")
PYTHON_CHANGED=$(echo "$CHANGED_FILES" | grep "^python/" || echo "")

# Handle common files that affect all containers
COMMON_CHANGED=$(echo "$CHANGED_FILES" | grep -E "^(\.github/workflows/|scripts/|README\.md)" || echo "")

if [[ -n "$COMMON_CHANGED" ]]; then
    echo "Common files changed, building all containers"
    echo "Common files that changed: $COMMON_CHANGED"
    echo "nodejs-changed=true" >> $GITHUB_OUTPUT
    echo "python-changed=true" >> $GITHUB_OUTPUT
else
    # Set outputs based on specific directory changes
    if [[ -n "$NODEJS_CHANGED" ]]; then
        echo "Node.js files changed: $NODEJS_CHANGED"
        echo "nodejs-changed=true" >> $GITHUB_OUTPUT
    else
        echo "nodejs-changed=false" >> $GITHUB_OUTPUT
    fi
    
    if [[ -n "$PYTHON_CHANGED" ]]; then
        echo "Python files changed: $PYTHON_CHANGED"
        echo "python-changed=true" >> $GITHUB_OUTPUT
    else
        echo "python-changed=false" >> $GITHUB_OUTPUT
    fi
fi

echo "Change detection completed successfully"