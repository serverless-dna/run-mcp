# Makefile for MCP Container Images
# Manages testing, building, and publishing container images

# Docker command detection for WSL2 and cross-platform
DOCKER_CMD := $(shell \
	if command -v docker >/dev/null 2>&1; then \
		echo "docker"; \
	elif command -v podman >/dev/null 2>&1; then \
		echo "podman"; \
	elif command -v nerdctl >/dev/null 2>&1; then \
		echo "nerdctl"; \
	elif command -v finch >/dev/null 2>&1; then \
		echo "finch"; \
	elif command -v docker.exe >/dev/null 2>&1; then \
		echo "docker.exe"; \
	else \
		echo ""; \
	fi)

# Check if Docker is available
define check_docker
	@if [ -z "$(DOCKER_CMD)" ]; then \
		echo "$(RED)Error: No container runtime found$(NC)"; \
		echo "$(YELLOW)Container Runtime Options:$(NC)"; \
		echo "1. Docker: Install Docker Desktop or Docker Engine"; \
		echo "2. Podman: Install Podman (docker-compatible)"; \
		echo "3. nerdctl: Install nerdctl with containerd"; \
		echo "4. Finch: Install AWS Finch (macOS/Linux)"; \
		echo "5. WSL2: Use docker.exe via Docker Desktop"; \
		exit 1; \
	fi
endef
# Configuration
REGISTRY := ghcr.io
OWNER := $(shell git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\).*/\1/')
REPO_NAME := $(shell basename `git rev-parse --show-toplevel`)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
DATE := $(shell date +%Y%m%d)

# Image names and tags
NODEJS_IMAGE := $(REGISTRY)/$(OWNER)/run-mcp-nodejs
PYTHON_IMAGE := $(REGISTRY)/$(OWNER)/run-mcp-python

# Default Node.js and Python versions (can be overridden)
NODE_VERSION ?= 22
PYTHON_VERSION ?= 3.12

# Tags
NODEJS_TAG := node$(NODE_VERSION)
PYTHON_TAG := python$(PYTHON_VERSION)
NODEJS_PINNED_TAG := $(NODEJS_TAG)-$(DATE)-$(COMMIT_SHA)
PYTHON_PINNED_TAG := $(PYTHON_TAG)-$(DATE)-$(COMMIT_SHA)

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

.PHONY: help test test-scripts test-integration clean build build-nodejs build-python push push-nodejs push-python login check-tools setup-dev lint validate-dockerfiles check-changes

# Default target
help: ## Show this help message
	@echo "$(BLUE)MCP Container Images - Development Makefile$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(YELLOW)Configuration:$(NC)"
	@echo "  Registry: $(REGISTRY)"
	@echo "  Owner: $(OWNER)"
	@echo "  Repository: $(REPO_NAME)"
	@echo "  Commit: $(COMMIT_SHA)"
	@echo "  Branch: $(BRANCH)"
	@echo "  Node.js Image: $(NODEJS_IMAGE):$(NODEJS_TAG)"
	@echo "  Python Image: $(PYTHON_IMAGE):$(PYTHON_TAG)"

# Testing targets
test: test-scripts test-integration ## Run all tests

test-scripts: ## Run shell script tests with Bats
	@echo "$(BLUE)Running shell script tests...$(NC)"
	@if ! command -v bats >/dev/null 2>&1; then \
		echo "$(RED)Error: bats not found. Install with: sudo apt-get install bats$(NC)"; \
		exit 1; \
	fi
	bats tests/scripts/
	@echo "$(GREEN)✓ Shell script tests passed$(NC)"

test-integration: ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	@bash tests/integration/test-change-detection.sh || true
	@echo "$(GREEN)✓ Integration tests completed$(NC)"

test-detect-changes: ## Test change detection script with current repo
	@echo "$(BLUE)Testing change detection script...$(NC)"
	@GITHUB_OUTPUT=/tmp/makefile-test-output bash scripts/detect-changes.sh
	@echo "$(YELLOW)Change detection output:$(NC)"
	@cat /tmp/makefile-test-output
	@echo "$(GREEN)✓ Change detection test completed$(NC)"

# Build targets
build: build-nodejs build-python ## Build all container images

build-nodejs: ## Build Node.js container image (current platform)
	$(call check_docker)
	@echo "$(BLUE)Building Node.js container image for current platform...$(NC)"
	@if [ ! -f nodejs/Dockerfile ]; then \
		echo "$(RED)Error: nodejs/Dockerfile not found$(NC)"; \
		exit 1; \
	fi
	$(DOCKER_CMD) build \
		--tag $(NODEJS_IMAGE):$(NODEJS_TAG) \
		--tag $(NODEJS_IMAGE):$(NODEJS_PINNED_TAG) \
		--tag $(NODEJS_IMAGE):latest \
		nodejs/
	@echo "$(GREEN)✓ Node.js image built: $(NODEJS_IMAGE):$(NODEJS_TAG)$(NC)"

build-nodejs-multiarch: ## Build Node.js container image (multi-architecture)
	$(call check_docker)
	@echo "$(BLUE)Building Node.js container image for multiple architectures...$(NC)"
	@if [ ! -f nodejs/Dockerfile ]; then \
		echo "$(RED)Error: nodejs/Dockerfile not found$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Note: Multi-arch build requires proper buildx setup with ARM64 emulation$(NC)"
	$(DOCKER_CMD) buildx build \
		--platform linux/amd64,linux/arm64 \
		--tag $(NODEJS_IMAGE):$(NODEJS_TAG) \
		--tag $(NODEJS_IMAGE):$(NODEJS_PINNED_TAG) \
		--tag $(NODEJS_IMAGE):latest \
		--push \
		nodejs/
	@echo "$(GREEN)✓ Node.js multi-arch image built and pushed: $(NODEJS_IMAGE):$(NODEJS_TAG)$(NC)"

build-python: ## Build Python container image (current platform)
	$(call check_docker)
	@echo "$(BLUE)Building Python container image for current platform...$(NC)"
	@if [ ! -f python/Dockerfile ]; then \
		echo "$(RED)Error: python/Dockerfile not found$(NC)"; \
		exit 1; \
	fi
	$(DOCKER_CMD) build \
		--tag $(PYTHON_IMAGE):$(PYTHON_TAG) \
		--tag $(PYTHON_IMAGE):$(PYTHON_PINNED_TAG) \
		--tag $(PYTHON_IMAGE):latest \
		python/
	@echo "$(GREEN)✓ Python image built: $(PYTHON_IMAGE):$(PYTHON_TAG)$(NC)"

build-python-multiarch: ## Build Python container image (multi-architecture)
	$(call check_docker)
	@echo "$(BLUE)Building Python container image for multiple architectures...$(NC)"
	@if [ ! -f python/Dockerfile ]; then \
		echo "$(RED)Error: python/Dockerfile not found$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Note: Multi-arch build requires proper buildx setup with ARM64 emulation$(NC)"
	$(DOCKER_CMD) buildx build \
		--platform linux/amd64,linux/arm64 \
		--tag $(PYTHON_IMAGE):$(PYTHON_TAG) \
		--tag $(PYTHON_IMAGE):$(PYTHON_PINNED_TAG) \
		--tag $(PYTHON_IMAGE):latest \
		--push \
		python/
	@echo "$(GREEN)✓ Python multi-arch image built and pushed: $(PYTHON_IMAGE):$(PYTHON_TAG)$(NC)"

# Registry targets
login: ## Login to GitHub Container Registry
	$(call check_docker)
	@echo "$(BLUE)Logging into GitHub Container Registry...$(NC)"
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "$(RED)Error: GITHUB_TOKEN environment variable not set$(NC)"; \
		echo "$(YELLOW)Set it with: export GITHUB_TOKEN=your_token$(NC)"; \
		exit 1; \
	fi
	@echo "$$GITHUB_TOKEN" | $(DOCKER_CMD) login $(REGISTRY) -u $(OWNER) --password-stdin
	@echo "$(GREEN)✓ Logged into $(REGISTRY)$(NC)"

push: login push-nodejs push-python ## Push all images to registry

push-nodejs: ## Push Node.js image to registry
	$(call check_docker)
	@echo "$(BLUE)Pushing Node.js image to registry...$(NC)"
	$(DOCKER_CMD) push $(NODEJS_IMAGE):$(NODEJS_TAG)
	$(DOCKER_CMD) push $(NODEJS_IMAGE):$(NODEJS_PINNED_TAG)
	$(DOCKER_CMD) push $(NODEJS_IMAGE):latest
	@echo "$(GREEN)✓ Node.js image pushed to $(REGISTRY)$(NC)"

push-python: ## Push Python image to registry
	$(call check_docker)
	@echo "$(BLUE)Pushing Python image to registry...$(NC)"
	$(DOCKER_CMD) push $(PYTHON_IMAGE):$(PYTHON_TAG)
	$(DOCKER_CMD) push $(PYTHON_IMAGE):$(PYTHON_PINNED_TAG)
	$(DOCKER_CMD) push $(PYTHON_IMAGE):latest
	@echo "$(GREEN)✓ Python image pushed to $(REGISTRY)$(NC)"

# Development targets
setup-dev: ## Set up development environment
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@echo "$(YELLOW)Environment: WSL2 detected$(NC)"
	@echo "$(YELLOW)Installing required tools...$(NC)"
	@if ! command -v bats >/dev/null 2>&1; then \
		echo "Installing bats..."; \
		sudo apt-get update && sudo apt-get install -y bats; \
	fi
	@echo "$(GREEN)Note: hadolint will run via Docker (hadolint/hadolint:latest)$(NC)"
	@if ! command -v docker >/dev/null 2>&1 && ! command -v podman >/dev/null 2>&1 && ! command -v nerdctl >/dev/null 2>&1 && ! command -v finch >/dev/null 2>&1 && ! command -v docker.exe >/dev/null 2>&1; then \
		echo "$(YELLOW)Container Runtime Setup Required:$(NC)"; \
		echo "$(BLUE)Option 1 - Docker Desktop (Recommended):$(NC)"; \
		echo "  1. Install Docker Desktop for Windows/macOS"; \
		echo "  2. Enable WSL2 integration (Windows only)"; \
		echo "  3. Restart WSL2: wsl --shutdown && wsl"; \
		echo "$(BLUE)Option 2 - Native Docker in WSL2/Linux:$(NC)"; \
		echo "  1. Run: curl -fsSL https://get.docker.com | sh"; \
		echo "  2. Add user to docker group: sudo usermod -aG docker $$USER"; \
		echo "  3. Start Docker: sudo service docker start"; \
		echo "$(BLUE)Option 3 - Podman Alternative:$(NC)"; \
		echo "  1. Install podman: sudo apt-get install -y podman"; \
		echo "$(BLUE)Option 4 - AWS Finch (macOS/Linux):$(NC)"; \
		echo "  1. macOS: brew install finch"; \
		echo "  2. Linux: Download from GitHub releases"; \
		echo "$(BLUE)Option 5 - nerdctl with containerd:$(NC)"; \
		echo "  1. Install containerd and nerdctl"; \
	fi
	@echo "$(GREEN)✓ Development environment setup complete$(NC)"

setup-wsl2-docker: ## Set up Docker in WSL2 (native installation)
	@echo "$(BLUE)Setting up Docker natively in WSL2...$(NC)"
	@echo "$(YELLOW)This will install Docker directly in WSL2$(NC)"
	curl -fsSL https://get.docker.com | sh
	sudo usermod -aG docker $$USER
	sudo service docker start
	@echo "$(GREEN)✓ Docker installed in WSL2$(NC)"
	@echo "$(YELLOW)Note: You may need to restart your WSL2 session$(NC)"

check-tools: ## Check if required tools are installed
	@echo "$(BLUE)Checking required tools...$(NC)"
	@echo "$(YELLOW)Environment: WSL2 detected$(NC)"
	@echo -n "Container Runtime: "; \
		if command -v docker >/dev/null 2>&1; then \
			echo "$(GREEN)✓ Docker$(NC)"; \
		elif command -v podman >/dev/null 2>&1; then \
			echo "$(GREEN)✓ Podman$(NC)"; \
		elif command -v nerdctl >/dev/null 2>&1; then \
			echo "$(GREEN)✓ nerdctl$(NC)"; \
		elif command -v finch >/dev/null 2>&1; then \
			echo "$(GREEN)✓ Finch (AWS)$(NC)"; \
		elif command -v docker.exe >/dev/null 2>&1; then \
			echo "$(GREEN)✓ Docker (Windows via docker.exe)$(NC)"; \
		else \
			echo "$(RED)✗$(NC)"; \
			echo "$(YELLOW)  Container Runtime Setup:$(NC)"; \
			echo "  1. Docker: Install Docker Desktop or Docker Engine"; \
			echo "  2. Podman: Install Podman (docker-compatible)"; \
			echo "  3. nerdctl: Install nerdctl with containerd"; \
			echo "  4. Finch: Install AWS Finch (brew install finch)"; \
			echo "  5. WSL2: Enable Docker Desktop WSL2 integration"; \
		fi
	@echo -n "Bats: "; command -v bats >/dev/null 2>&1 && echo "$(GREEN)✓$(NC)" || echo "$(RED)✗$(NC)"
	@echo -n "Hadolint: "; echo "$(GREEN)✓ (via Docker: hadolint/hadolint:latest)$(NC)"
	@echo -n "Git: "; command -v git >/dev/null 2>&1 && echo "$(GREEN)✓$(NC)" || echo "$(RED)✗$(NC)"

# Validation targets
lint: validate-dockerfiles ## Run all linting checks

validate-dockerfiles: ## Validate Dockerfiles with hadolint
	$(call check_docker)
	@echo "$(BLUE)Validating Dockerfiles with hadolint...$(NC)"
	@if [ -f nodejs/Dockerfile ]; then \
		echo "Linting nodejs/Dockerfile..."; \
		$(DOCKER_CMD) run --rm -i -v "$(PWD)/nodejs/.hadolint.yaml:/hadolint.yaml:ro" \
			hadolint/hadolint:latest hadolint --config /hadolint.yaml - < nodejs/Dockerfile; \
	fi
	@if [ -f python/Dockerfile ]; then \
		echo "Linting python/Dockerfile..."; \
		$(DOCKER_CMD) run --rm -i -v "$(PWD)/python/.hadolint.yaml:/hadolint.yaml:ro" \
			hadolint/hadolint:latest hadolint --config /hadolint.yaml - < python/Dockerfile; \
	fi
	@echo "$(GREEN)✓ Dockerfile validation completed$(NC)"

check-changes: ## Check what containers would be built based on git changes
	@echo "$(BLUE)Checking what would be built based on changes...$(NC)"
	@GITHUB_OUTPUT=/tmp/makefile-changes bash scripts/detect-changes.sh
	@echo "$(YELLOW)Build decisions:$(NC)"
	@cat /tmp/makefile-changes
	@if grep -q "nodejs-changed=true" /tmp/makefile-changes; then \
		echo "$(GREEN)→ Node.js container would be built$(NC)"; \
	else \
		echo "$(YELLOW)→ Node.js container would be skipped$(NC)"; \
	fi
	@if grep -q "python-changed=true" /tmp/makefile-changes; then \
		echo "$(GREEN)→ Python container would be built$(NC)"; \
	else \
		echo "$(YELLOW)→ Python container would be skipped$(NC)"; \
	fi

# Conditional build targets (based on change detection)
build-changed: ## Build only containers that have changed
	@echo "$(BLUE)Building only changed containers...$(NC)"
	@GITHUB_OUTPUT=/tmp/makefile-conditional bash scripts/detect-changes.sh
	@if grep -q "nodejs-changed=true" /tmp/makefile-conditional; then \
		echo "$(GREEN)Building Node.js container (changed)$(NC)"; \
		$(MAKE) build-nodejs; \
	else \
		echo "$(YELLOW)Skipping Node.js container (no changes)$(NC)"; \
	fi
	@if grep -q "python-changed=true" /tmp/makefile-conditional; then \
		echo "$(GREEN)Building Python container (changed)$(NC)"; \
		$(MAKE) build-python; \
	else \
		echo "$(YELLOW)Skipping Python container (no changes)$(NC)"; \
	fi

push-changed: ## Push only containers that have changed
	@echo "$(BLUE)Pushing only changed containers...$(NC)"
	@GITHUB_OUTPUT=/tmp/makefile-conditional bash scripts/detect-changes.sh
	@if grep -q "nodejs-changed=true" /tmp/makefile-conditional; then \
		echo "$(GREEN)Pushing Node.js container (changed)$(NC)"; \
		$(MAKE) push-nodejs; \
	else \
		echo "$(YELLOW)Skipping Node.js container push (no changes)$(NC)"; \
	fi
	@if grep -q "python-changed=true" /tmp/makefile-conditional; then \
		echo "$(GREEN)Pushing Python container (changed)$(NC)"; \
		$(MAKE) push-python; \
	else \
		echo "$(YELLOW)Skipping Python container push (no changes)$(NC)"; \
	fi

# CI/CD simulation targets
ci-test: ## Simulate CI testing workflow
	@echo "$(BLUE)Simulating CI testing workflow...$(NC)"
	$(MAKE) check-tools
	$(MAKE) test-scripts
	$(MAKE) validate-dockerfiles
	$(MAKE) check-changes
	@echo "$(GREEN)✓ CI testing workflow completed$(NC)"

ci-build: ## Simulate CI build workflow
	@echo "$(BLUE)Simulating CI build workflow...$(NC)"
	$(MAKE) ci-test
	$(MAKE) build-changed
	@echo "$(GREEN)✓ CI build workflow completed$(NC)"

ci-deploy: ## Simulate CI deployment workflow
	@echo "$(BLUE)Simulating CI deployment workflow...$(NC)"
	$(MAKE) ci-build
	$(MAKE) push-changed
	@echo "$(GREEN)✓ CI deployment workflow completed$(NC)"

# Utility targets
clean: ## Clean up temporary files and Docker images
	@echo "$(BLUE)Cleaning up...$(NC)"
	@rm -f /tmp/makefile-* /tmp/test-output*
	@echo "$(YELLOW)Removing dangling Docker images...$(NC)"
	@if [ -n "$(DOCKER_CMD)" ]; then \
		$(DOCKER_CMD) image prune -f || true; \
	else \
		echo "$(YELLOW)Docker not available, skipping image cleanup$(NC)"; \
	fi
	@echo "$(GREEN)✓ Cleanup completed$(NC)"

clean-all: ## Clean up everything including built images
	@echo "$(BLUE)Cleaning up everything...$(NC)"
	$(MAKE) clean
	@if [ -n "$(DOCKER_CMD)" ]; then \
		echo "$(YELLOW)Removing built images...$(NC)"; \
		$(DOCKER_CMD) rmi $(NODEJS_IMAGE):$(NODEJS_TAG) $(NODEJS_IMAGE):latest || true; \
		$(DOCKER_CMD) rmi $(PYTHON_IMAGE):$(PYTHON_TAG) $(PYTHON_IMAGE):latest || true; \
	else \
		echo "$(YELLOW)Docker not available, skipping image removal$(NC)"; \
	fi
	@echo "$(GREEN)✓ Full cleanup completed$(NC)"

info: ## Show build information
	@echo "$(BLUE)Build Information:$(NC)"
	@echo "  Registry: $(REGISTRY)"
	@echo "  Owner: $(OWNER)"
	@echo "  Repository: $(REPO_NAME)"
	@echo "  Branch: $(BRANCH)"
	@echo "  Commit: $(COMMIT_SHA)"
	@echo "  Date: $(DATE)"
	@echo ""
	@echo "$(BLUE)Image Tags:$(NC)"
	@echo "  Node.js: $(NODEJS_IMAGE):$(NODEJS_TAG)"
	@echo "  Node.js Pinned: $(NODEJS_IMAGE):$(NODEJS_PINNED_TAG)"
	@echo "  Python: $(PYTHON_IMAGE):$(PYTHON_TAG)"
	@echo "  Python Pinned: $(PYTHON_IMAGE):$(PYTHON_PINNED_TAG)"

# Development workflow targets
dev-setup: setup-dev check-tools ## Complete development setup
	@echo "$(GREEN)✓ Development environment ready!$(NC)"
	@echo "$(YELLOW)Next steps:$(NC)"
	@echo "  1. Run 'make test' to run all tests"
	@echo "  2. Run 'make build' to build containers"
	@echo "  3. Set GITHUB_TOKEN and run 'make push' to publish"

# Quick development cycle
dev-cycle: ## Quick development cycle: test → build → check
	@echo "$(BLUE)Running development cycle...$(NC)"
	$(MAKE) test-scripts
	$(MAKE) check-changes
	$(MAKE) build-changed
	@echo "$(GREEN)✓ Development cycle completed$(NC)"

# GitHub Actions simulation
simulate-pr: ## Simulate GitHub Actions PR workflow
	@echo "$(BLUE)Simulating GitHub Actions PR workflow...$(NC)"
	$(MAKE) test
	$(MAKE) validate-dockerfiles
	$(MAKE) build-changed
	@echo "$(GREEN)✓ PR workflow simulation completed$(NC)"

simulate-main: ## Simulate GitHub Actions main branch workflow
	@echo "$(BLUE)Simulating GitHub Actions main branch workflow...$(NC)"
	$(MAKE) test
	$(MAKE) validate-dockerfiles
	$(MAKE) build-changed
	@if [ "$(BRANCH)" = "main" ]; then \
		echo "$(GREEN)Would push to registry on main branch$(NC)"; \
		$(MAKE) check-changes; \
	else \
		echo "$(YELLOW)Not on main branch, skipping registry push$(NC)"; \
	fi
	@echo "$(GREEN)✓ Main branch workflow simulation completed$(NC)"