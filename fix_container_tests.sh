#!/bin/bash

# Fix container test files to use dynamic runtime detection

for file in tests/containers/*.bats; do
    echo "Fixing $file..."
    
    # Add helper function if not already present
    if ! grep -q "get_runtime()" "$file"; then
        # Add helper function after the shebang and comments
        sed -i '/^# Validates:/a\\n# Helper function to get the container runtime\nget_runtime() {\n    echo "${CONTAINER_RUNTIME:-docker}"\n}' "$file"
    fi
    
    # Replace docker commands with $(get_runtime)
    sed -i 's/docker run/$(get_runtime) run/g' "$file"
    sed -i 's/docker build/$(get_runtime) build/g' "$file"
    sed -i 's/docker image/$(get_runtime) image/g' "$file"
    sed -i 's/docker rmi/$(get_runtime) rmi/g' "$file"
    sed -i 's/docker exec/$(get_runtime) exec/g' "$file"
    
    # Fix any remaining standalone docker commands
    sed -i 's/\bdocker\b/$(get_runtime)/g' "$file"
    
    echo "Fixed $file"
done

echo "All container test files updated!"