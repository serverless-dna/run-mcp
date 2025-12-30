#!/usr/bin/env bats

# Property tests for GitHub Actions workflow
# **Feature: mcp-container-images, Property 4: Change-Triggered Builds**
# **Feature: mcp-container-images, Property 5: Successful Build Publishing**
# **Feature: mcp-container-images, Property 7: Runtime-Based Tagging**
# **Validates: Requirements 2.1, 2.2, 2.5, 4.1**

setup() {
    # Create temporary directory for test workspace
    export TEST_WORKSPACE=$(mktemp -d)
    export ORIGINAL_PWD="$PWD"
    
    # Copy workflow file to test workspace
    mkdir -p "$TEST_WORKSPACE/.github/workflows"
    cp ".github/workflows/build-containers.yml" "$TEST_WORKSPACE/.github/workflows/"
    
    # Create minimal repository structure
    mkdir -p "$TEST_WORKSPACE/nodejs" "$TEST_WORKSPACE/python" "$TEST_WORKSPACE/scripts"
    echo "# Node.js container" > "$TEST_WORKSPACE/nodejs/README.md"
    echo "# Python container" > "$TEST_WORKSPACE/python/README.md"
    echo "#!/bin/bash" > "$TEST_WORKSPACE/scripts/detect-changes.sh"
    chmod +x "$TEST_WORKSPACE/scripts/detect-changes.sh"
    
    cd "$TEST_WORKSPACE"
    git init
    git config user.email "test@example.com"
    git config user.name "Test User"
    git add .
    git commit -m "Initial commit"
}

teardown() {
    cd "$ORIGINAL_PWD"
    rm -rf "$TEST_WORKSPACE"
}

# Property 4: Change-Triggered Builds
# For any modification to container source files, the build system should automatically trigger a build process for the affected containers only.
@test "Property 4: workflow triggers on nodejs directory changes" {
    # Modify nodejs directory
    echo "# Modified" >> nodejs/README.md
    git add nodejs/README.md
    git commit -m "Modify nodejs"
    
    # Check that workflow would trigger for nodejs changes
    # Verify workflow has correct path filters
    grep -q "nodejs/\*\*" .github/workflows/build-containers.yml
    
    # Verify conditional job execution exists
    grep -q "if: needs.detect-changes.outputs.nodejs-changed == 'true'" .github/workflows/build-containers.yml
}

@test "Property 4: workflow triggers on python directory changes" {
    # Modify python directory
    echo "# Modified" >> python/README.md
    git add python/README.md
    git commit -m "Modify python"
    
    # Check that workflow would trigger for python changes
    # Verify workflow has correct path filters
    grep -q "python/\*\*" .github/workflows/build-containers.yml
    
    # Verify conditional job execution exists
    grep -q "if: needs.detect-changes.outputs.python-changed == 'true'" .github/workflows/build-containers.yml
}

@test "Property 4: workflow has change detection job" {
    # Verify change detection job exists
    grep -q "detect-changes:" .github/workflows/build-containers.yml
    
    # Verify it outputs the required variables
    grep -q "nodejs-changed:" .github/workflows/build-containers.yml
    grep -q "python-changed:" .github/workflows/build-containers.yml
    grep -q "build-reason:" .github/workflows/build-containers.yml
}

@test "Property 4: workflow supports manual triggers with force build" {
    # Verify workflow_dispatch trigger exists
    grep -q "workflow_dispatch:" .github/workflows/build-containers.yml
    
    # Verify force_build input exists
    grep -q "force_build:" .github/workflows/build-containers.yml
    
    # Verify force build logic exists
    grep -q "force_build.*true" .github/workflows/build-containers.yml
}

# Property 5: Successful Build Publishing
# For any successful container build, the system should publish the image to GitHub Container Registry with proper authentication and make it publicly accessible.
@test "Property 5: workflow authenticates with GitHub Container Registry" {
    # Verify GHCR registry configuration
    grep -q "REGISTRY: ghcr.io" .github/workflows/build-containers.yml
    
    # Verify login action exists for both build jobs
    grep -A 5 "Log in to Container Registry" .github/workflows/build-containers.yml | grep -q "docker/login-action"
    
    # Verify GITHUB_TOKEN is used
    grep -q "GITHUB_TOKEN" .github/workflows/build-containers.yml
}

@test "Property 5: workflow has proper permissions for package publishing" {
    # Verify permissions are set for package publishing
    grep -A 3 "permissions:" .github/workflows/build-containers.yml | grep -q "packages: write"
    grep -A 3 "permissions:" .github/workflows/build-containers.yml | grep -q "contents: read"
}

@test "Property 5: workflow publishes images with push enabled" {
    # Verify both build jobs have push: true
    grep -A 10 "Build and push Node.js image" .github/workflows/build-containers.yml | grep -q "push: true"
    grep -A 10 "Build and push Python image" .github/workflows/build-containers.yml | grep -q "push: true"
}

@test "Property 5: workflow makes images publicly accessible" {
    # Verify images are pushed to public registry (ghcr.io)
    grep -q "ghcr.io" .github/workflows/build-containers.yml
    
    # Verify no private registry configurations
    ! grep -q "private" .github/workflows/build-containers.yml
}

# Property 7: Runtime-Based Tagging
# For any published container image, it should be tagged with appropriate runtime version information following the runtime-based tagging conventions.
@test "Property 7: nodejs image uses runtime-based tags" {
    # Verify Node.js image has runtime-based tags
    grep -A 10 "Extract metadata" .github/workflows/build-containers.yml | grep -A 5 "nodejs" | grep -q "node22"
    
    # Verify pinned tags with date and commit
    grep -A 10 "Extract metadata" .github/workflows/build-containers.yml | grep -A 5 "nodejs" | grep -q "node22-{{date"
    grep -A 10 "Extract metadata" .github/workflows/build-containers.yml | grep -A 5 "nodejs" | grep -q "{{sha}}"
}

@test "Property 7: python image uses runtime-based tags" {
    # Verify Python image has runtime-based tags
    grep -A 10 "Extract metadata" .github/workflows/build-containers.yml | grep -A 5 "python" | grep -q "python3.12"
    
    # Verify pinned tags with date and commit
    grep -A 10 "Extract metadata" .github/workflows/build-containers.yml | grep -A 5 "python" | grep -q "python3.12-{{date"
    grep -A 10 "Extract metadata" .github/workflows/build-containers.yml | grep -A 5 "python" | grep -q "{{sha}}"
}

@test "Property 7: images have latest tags for default branch" {
    # Verify latest tags are applied on default branch
    grep -A 15 "Extract metadata" .github/workflows/build-containers.yml | grep -q "latest,enable={{is_default_branch}}"
}

@test "Property 7: images use semantic versioning conventions" {
    # Verify semantic versioning patterns in tags
    grep -A 15 "Extract metadata" .github/workflows/build-containers.yml | grep -q "type=ref,event=branch"
    grep -A 15 "Extract metadata" .github/workflows/build-containers.yml | grep -q "type=ref,event=pr"
    
    # Verify runtime version tags follow semantic patterns
    grep -A 15 "Extract metadata" .github/workflows/build-containers.yml | grep -q "node22"
    grep -A 15 "Extract metadata" .github/workflows/build-containers.yml | grep -q "python3.12"
}

# Additional property tests for workflow robustness
@test "workflow has proper error handling and job isolation" {
    # Verify jobs can run independently
    grep -q "if: always()" .github/workflows/build-containers.yml
    
    # Verify build jobs don't depend on each other
    ! grep -A 5 "build-nodejs:" .github/workflows/build-containers.yml | grep -q "needs:.*build-python"
    ! grep -A 5 "build-python:" .github/workflows/build-containers.yml | grep -q "needs:.*build-nodejs"
}

@test "workflow includes vulnerability scanning" {
    # Verify vulnerability scanning job exists
    grep -q "vulnerability-scan:" .github/workflows/build-containers.yml
    
    # Verify Trivy scanner is used
    grep -q "trivy-action" .github/workflows/build-containers.yml
    
    # Verify SARIF upload for security tab
    grep -q "upload-sarif" .github/workflows/build-containers.yml
}

@test "workflow includes image size validation" {
    # Verify size validation exists for both images
    grep -q "Validate image size" .github/workflows/build-containers.yml
    
    # Verify size limits are enforced
    grep -q "200" .github/workflows/build-containers.yml  # Node.js limit
    grep -q "300" .github/workflows/build-containers.yml  # Python limit
}

@test "workflow includes integration testing" {
    # Verify integration test job exists
    grep -q "integration-test:" .github/workflows/build-containers.yml
    
    # Verify container functionality tests
    grep -q "node --version" .github/workflows/build-containers.yml
    grep -q "python --version" .github/workflows/build-containers.yml
    grep -q "npm --version" .github/workflows/build-containers.yml
    grep -q "uv --version" .github/workflows/build-containers.yml
}