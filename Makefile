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
COMMIT_SHA := $(shell git rev-parse --short HEAD)
DATE := $(shell date +%Y%m%d)

# Image names and tags
NODEJS_IMAGE := $(REGISTRY)/$(OWNER)/$(REPO_NAME)-nodejs
PYTHON_IMAGE := $(REGISTRY)/$(OWNER)/$(REPO_NAME)-python

# Supported versions (can be overridden)
NODEJS_VERSIONS := 18 20 22
PYTHON_VERSIONS := 3.11 3.12 3.13

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

.PHONY: help test clean build push login setup-dev lint validate check-upstream check-upstream-force build-run-mcp install-run-mcp

# Default target
help: ## Show this help message
	@echo "$(BLUE)MCP Container Images - Development Makefile$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-25s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(YELLOW)Configuration:$(NC)"
	@echo "  Registry: $(REGISTRY)"
	@echo "  Owner: $(OWNER)"
	@echo "  Repository: $(REPO_NAME)"
	@echo "  Commit: $(COMMIT_SHA)"
	@echo "  Branch: $(BRANCH)"
	@echo "  Container Runtime: $(DOCKER_CMD)"

# =============================================================================
# VERSION MANAGEMENT TARGETS
# =============================================================================

check-upstream: ## Check for upstream version updates and trigger builds
	@echo "$(BLUE)Checking for upstream version updates...$(NC)"
	@chmod +x scripts/check-and-build-versions.sh
	@GITHUB_REPOSITORY="$(OWNER)/$(REPO_NAME)" \
	 GITHUB_TOKEN="$(GITHUB_TOKEN)" \
	 bash scripts/check-and-build-versions.sh

check-upstream-force: ## Force check for upstream version updates
	@echo "$(BLUE)Force checking for upstream version updates...$(NC)"
	@chmod +x scripts/check-and-build-versions.sh
	@GITHUB_REPOSITORY="$(OWNER)/$(REPO_NAME)" \
	 GITHUB_TOKEN="$(GITHUB_TOKEN)" \
	 bash scripts/check-and-build-versions.sh --force

check-upstream-ci: ## Check upstream versions for CI (outputs to GITHUB_OUTPUT)
	@echo "$(BLUE)Checking upstream versions for CI...$(NC)"
	@chmod +x scripts/check-upstream-versions.sh
	@bash scripts/check-upstream-versions.sh

cleanup-versions: ## Clean up old container versions from registry
	@echo "$(BLUE)Cleaning up old container versions...$(NC)"
	@if [ ! -f "scripts/cleanup-versions.sh" ]; then \
		echo "$(RED)Error: scripts/cleanup-versions.sh not found$(NC)"; \
		exit 1; \
	fi
	@chmod +x scripts/cleanup-versions.sh
	@GITHUB_TOKEN="$(GITHUB_TOKEN)" bash scripts/cleanup-versions.sh

# =============================================================================
# BUILD TARGETS
# =============================================================================

build: build-nodejs build-python ## Build all container images locally (latest versions)

build-matrix: ## Build all supported versions from matrix
	@echo "$(BLUE)Building all supported versions...$(NC)"
	@$(MAKE) build-nodejs-matrix
	@$(MAKE) build-python-matrix

build-nodejs: ## Build Node.js container image locally (latest LTS)
	$(call check_docker)
	@echo "$(BLUE)Building Node.js container image...$(NC)"
	@if [ ! -f nodejs/Dockerfile ]; then \
		echo "$(RED)Error: nodejs/Dockerfile not found$(NC)"; \
		exit 1; \
	fi
	@echo "Getting latest Node.js LTS version..."
	@NODEJS_VERSION=$$(curl -s https://nodejs.org/dist/index.json | jq -r '[.[] | select(.lts != false)] | .[0].version' | tr -d 'v' || echo "22.11.0"); \
	NODEJS_MAJOR=$$(echo "$$NODEJS_VERSION" | cut -d. -f1); \
	echo "Building with Node.js $$NODEJS_VERSION (major: $$NODEJS_MAJOR)"; \
	$(DOCKER_CMD) build \
		--build-arg NODE_VERSION=$$NODEJS_VERSION \
		--tag $(NODEJS_IMAGE):node$$NODEJS_MAJOR \
		--tag $(NODEJS_IMAGE):node$$NODEJS_VERSION \
		--tag $(NODEJS_IMAGE):node$$NODEJS_VERSION-$(DATE) \
		--tag $(NODEJS_IMAGE):latest \
		nodejs/
	@echo "$(GREEN)✓ Node.js image built$(NC)"

build-nodejs-matrix: ## Build all supported Node.js versions
	$(call check_docker)
	@echo "$(BLUE)Building all supported Node.js versions: $(NODEJS_VERSIONS)$(NC)"
	@for major in $(NODEJS_VERSIONS); do \
		echo "Getting latest Node.js $$major.x version..."; \
		NODEJS_VERSION=$$(curl -s https://nodejs.org/dist/index.json | jq -r --arg maj "$$major" '[.[] | select(.version | startswith("v" + $$maj + "."))] | .[0].version' | tr -d 'v' || echo "$$major.0.0"); \
		echo "Building Node.js $$major with version $$NODEJS_VERSION"; \
		$(DOCKER_CMD) build \
			--build-arg NODE_VERSION=$$NODEJS_VERSION \
			--tag $(NODEJS_IMAGE):node$$major \
			--tag $(NODEJS_IMAGE):node$$NODEJS_VERSION \
			--tag $(NODEJS_IMAGE):node$$NODEJS_VERSION-$(DATE) \
			nodejs/ || exit 1; \
	done
	@echo "$(GREEN)✓ All Node.js versions built$(NC)"

build-python: ## Build Python container image locally (latest 3.12.x)
	$(call check_docker)
	@echo "$(BLUE)Building Python container image...$(NC)"
	@if [ ! -f python/Dockerfile ]; then \
		echo "$(RED)Error: python/Dockerfile not found$(NC)"; \
		exit 1; \
	fi
	@echo "Getting latest Python 3.12.x version..."
	@PYTHON_VERSION=$$(curl -s https://endoflife.date/api/python.json | jq -r '[.[] | select(.cycle == "3.12")] | .[0].latest' || echo "3.12.8"); \
	PYTHON_MAJOR_MINOR=$$(echo "$$PYTHON_VERSION" | cut -d. -f1-2); \
	echo "Building with Python $$PYTHON_VERSION (major.minor: $$PYTHON_MAJOR_MINOR)"; \
	$(DOCKER_CMD) build \
		--build-arg PYTHON_VERSION=$$PYTHON_VERSION \
		--tag $(PYTHON_IMAGE):python$$PYTHON_MAJOR_MINOR \
		--tag $(PYTHON_IMAGE):python$$PYTHON_VERSION \
		--tag $(PYTHON_IMAGE):python$$PYTHON_VERSION-$(DATE) \
		--tag $(PYTHON_IMAGE):latest \
		python/
	@echo "$(GREEN)✓ Python image built$(NC)"

build-python-matrix: ## Build all supported Python versions
	$(call check_docker)
	@echo "$(BLUE)Building all supported Python versions: $(PYTHON_VERSIONS)$(NC)"
	@for version in $(PYTHON_VERSIONS); do \
		echo "Getting latest Python $$version.x version..."; \
		PYTHON_VERSION=$$(curl -s https://endoflife.date/api/python.json | jq -r --arg ver "$$version" '[.[] | select(.cycle == $$ver)] | .[0].latest' || echo "$$version.0"); \
		echo "Building Python $$version with version $$PYTHON_VERSION"; \
		$(DOCKER_CMD) build \
			--build-arg PYTHON_VERSION=$$PYTHON_VERSION \
			--tag $(PYTHON_IMAGE):python$$version \
			--tag $(PYTHON_IMAGE):python$$PYTHON_VERSION \
			--tag $(PYTHON_IMAGE):python$$PYTHON_VERSION-$(DATE) \
			python/ || exit 1; \
	done
	@echo "$(GREEN)✓ All Python versions built$(NC)"

# =============================================================================
# GO BINARY TARGETS
# =============================================================================

build-run-mcp: ## Build run-mcp binary for current platform
	@echo "$(BLUE)Building run-mcp binary...$(NC)"
	@if [ ! -d "cmd/run-mcp" ]; then \
		echo "$(RED)Error: cmd/run-mcp directory not found$(NC)"; \
		exit 1; \
	fi
	@mkdir -p build
	@cd cmd/run-mcp && go build -o ../../build/run-mcp .
	@echo "$(GREEN)✓ Built build/run-mcp$(NC)"

build-run-mcp-all: ## Build run-mcp binary for all platforms
	@echo "$(BLUE)Building run-mcp binaries for all platforms...$(NC)"
	@if [ ! -d "cmd/run-mcp" ]; then \
		echo "$(RED)Error: cmd/run-mcp directory not found$(NC)"; \
		exit 1; \
	fi
	@mkdir -p build
	@cd cmd/run-mcp && \
		GOOS=windows GOARCH=amd64 go build -o ../../build/run-mcp-windows-amd64.exe . && \
		GOOS=darwin GOARCH=amd64 go build -o ../../build/run-mcp-darwin-amd64 . && \
		GOOS=darwin GOARCH=arm64 go build -o ../../build/run-mcp-darwin-arm64 . && \
		GOOS=linux GOARCH=amd64 go build -o ../../build/run-mcp-linux-amd64 . && \
		GOOS=linux GOARCH=arm64 go build -o ../../build/run-mcp-linux-arm64 .
	@echo "$(GREEN)✓ Built all platform binaries in build/$(NC)"
	@ls -la build/run-mcp-*

install-run-mcp: build-run-mcp ## Install run-mcp binary to /usr/local/bin
	@echo "$(BLUE)Installing run-mcp binary...$(NC)"
	@sudo cp build/run-mcp /usr/local/bin/
	@echo "$(GREEN)✓ Installed run-mcp to /usr/local/bin/$(NC)"

# =============================================================================
# REGISTRY TARGETS
# =============================================================================

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

push-matrix: login push-nodejs-matrix push-python-matrix ## Push all matrix images to registry

build-and-push: ## Build and push latest versions (combined operation)
	@echo "$(BLUE)Building and pushing latest versions...$(NC)"
	$(MAKE) build
	$(MAKE) push
	@echo "$(GREEN)✓ Build and push completed$(NC)"

build-and-push-matrix: ## Build and push all matrix versions (combined operation)
	@echo "$(BLUE)Building and pushing all matrix versions...$(NC)"
	$(MAKE) build-matrix
	$(MAKE) push-matrix
	@echo "$(GREEN)✓ Matrix build and push completed$(NC)"

push-nodejs: ## Push Node.js image to registry
	$(call check_docker)
	@echo "$(BLUE)Pushing Node.js image to registry...$(NC)"
	@NODEJS_VERSION=$$(curl -s https://nodejs.org/dist/index.json | jq -r '[.[] | select(.lts != false)] | .[0].version' | tr -d 'v' || echo "22.11.0"); \
	NODEJS_MAJOR=$$(echo "$$NODEJS_VERSION" | cut -d. -f1); \
	$(DOCKER_CMD) push $(NODEJS_IMAGE):node$$NODEJS_MAJOR; \
	$(DOCKER_CMD) push $(NODEJS_IMAGE):node$$NODEJS_VERSION; \
	$(DOCKER_CMD) push $(NODEJS_IMAGE):latest
	@echo "$(GREEN)✓ Node.js image pushed to $(REGISTRY)$(NC)"

push-nodejs-matrix: ## Push all Node.js matrix images to registry
	$(call check_docker)
	@echo "$(BLUE)Pushing all Node.js matrix images to registry...$(NC)"
	@for major in $(NODEJS_VERSIONS); do \
		echo "Pushing Node.js $$major images..."; \
		$(DOCKER_CMD) push $(NODEJS_IMAGE):node$$major || exit 1; \
		$(DOCKER_CMD) image ls $(NODEJS_IMAGE):node$$major.* --format "table {{.Tag}}" | grep -v TAG | while read tag; do \
			if [ -n "$$tag" ]; then \
				$(DOCKER_CMD) push $(NODEJS_IMAGE):$$tag || exit 1; \
			fi; \
		done; \
	done
	@echo "$(GREEN)✓ All Node.js matrix images pushed to $(REGISTRY)$(NC)"

push-python: ## Push Python image to registry
	$(call check_docker)
	@echo "$(BLUE)Pushing Python image to registry...$(NC)"
	@PYTHON_VERSION=$$(curl -s https://endoflife.date/api/python.json | jq -r '[.[] | select(.cycle == "3.12")] | .[0].latest' || echo "3.12.8"); \
	PYTHON_MAJOR_MINOR=$$(echo "$$PYTHON_VERSION" | cut -d. -f1-2); \
	$(DOCKER_CMD) push $(PYTHON_IMAGE):python$$PYTHON_MAJOR_MINOR; \
	$(DOCKER_CMD) push $(PYTHON_IMAGE):python$$PYTHON_VERSION; \
	$(DOCKER_CMD) push $(PYTHON_IMAGE):latest
	@echo "$(GREEN)✓ Python image pushed to $(REGISTRY)$(NC)"

push-python-matrix: ## Push all Python matrix images to registry
	$(call check_docker)
	@echo "$(BLUE)Pushing all Python matrix images to registry...$(NC)"
	@for version in $(PYTHON_VERSIONS); do \
		echo "Pushing Python $$version images..."; \
		$(DOCKER_CMD) push $(PYTHON_IMAGE):python$$version || exit 1; \
		$(DOCKER_CMD) image ls $(PYTHON_IMAGE):python$$version.* --format "table {{.Tag}}" | grep -v TAG | while read tag; do \
			if [ -n "$$tag" ]; then \
				$(DOCKER_CMD) push $(PYTHON_IMAGE):$$tag || exit 1; \
			fi; \
		done; \
	done
	@echo "$(GREEN)✓ All Python matrix images pushed to $(REGISTRY)$(NC)"

# =============================================================================
# TESTING TARGETS
# =============================================================================

test: test-scripts test-run-mcp ## Run all tests

test-scripts: ## Run shell script tests with Bats
	@echo "$(BLUE)Running shell script tests...$(NC)"
	@if ! command -v bats >/dev/null 2>&1; then \
		echo "$(RED)Error: bats not found. Install with: sudo apt-get install bats$(NC)"; \
		exit 1; \
	fi
	@if [ -d "tests/scripts" ]; then \
		bats tests/scripts/; \
	else \
		echo "$(YELLOW)No script tests found$(NC)"; \
	fi
	@echo "$(GREEN)✓ Shell script tests passed$(NC)"

test-containers: ## Run container tests with Bats
	@echo "$(BLUE)Running container tests...$(NC)"
	@if ! command -v bats >/dev/null 2>&1; then \
		echo "$(RED)Error: bats not found. Install with: sudo apt-get install bats$(NC)"; \
		exit 1; \
	fi
	@if [ -d "tests/containers" ]; then \
		bats tests/containers/; \
	else \
		echo "$(YELLOW)No container tests found$(NC)"; \
	fi
	@echo "$(GREEN)✓ Container tests passed$(NC)"

test-run-mcp: ## Test the run-mcp binary
	@echo "$(BLUE)Testing run-mcp binary...$(NC)"
	@if [ -d "cmd/run-mcp" ]; then \
		cd cmd/run-mcp && go test -v .; \
	else \
		echo "$(YELLOW)No run-mcp tests found$(NC)"; \
	fi
	@echo "$(GREEN)✓ run-mcp tests passed$(NC)"

# =============================================================================
# VALIDATION TARGETS
# =============================================================================

lint: validate-dockerfiles ## Run all linting checks

validate-dockerfiles: ## Validate Dockerfiles with hadolint
	$(call check_docker)
	@echo "$(BLUE)Validating Dockerfiles with hadolint...$(NC)"
	@if [ -f nodejs/Dockerfile ]; then \
		echo "Linting nodejs/Dockerfile..."; \
		$(DOCKER_CMD) run --rm -i -v "$$(pwd)/nodejs/.hadolint.yaml:/hadolint.yaml:ro" \
			hadolint/hadolint:latest hadolint --config /hadolint.yaml - < nodejs/Dockerfile; \
	fi
	@if [ -f python/Dockerfile ]; then \
		echo "Linting python/Dockerfile..."; \
		$(DOCKER_CMD) run --rm -i -v "$$(pwd)/python/.hadolint.yaml:/hadolint.yaml:ro" \
			hadolint/hadolint:latest hadolint --config /hadolint.yaml - < python/Dockerfile; \
	fi
	@echo "$(GREEN)✓ Dockerfile validation completed$(NC)"

# =============================================================================
# DEVELOPMENT TARGETS
# =============================================================================

setup-dev: ## Set up development environment
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@echo "$(YELLOW)Installing required tools...$(NC)"
	@if ! command -v bats >/dev/null 2>&1; then \
		echo "Installing bats..."; \
		sudo apt-get update && sudo apt-get install -y bats; \
	fi
	@if ! command -v jq >/dev/null 2>&1; then \
		echo "Installing jq..."; \
		sudo apt-get install -y jq; \
	fi
	@if ! command -v curl >/dev/null 2>&1; then \
		echo "Installing curl..."; \
		sudo apt-get install -y curl; \
	fi
	@echo "$(GREEN)✓ Development environment setup complete$(NC)"

check-tools: ## Check if required tools are installed
	@echo "$(BLUE)Checking required tools...$(NC)"
	@echo -n "Container Runtime: "; \
		if [ -n "$(DOCKER_CMD)" ]; then \
			echo "$(GREEN)✓ $(DOCKER_CMD)$(NC)"; \
		else \
			echo "$(RED)✗ None found$(NC)"; \
		fi
	@echo -n "Bats: "; command -v bats >/dev/null 2>&1 && echo "$(GREEN)✓$(NC)" || echo "$(RED)✗$(NC)"
	@echo -n "jq: "; command -v jq >/dev/null 2>&1 && echo "$(GREEN)✓$(NC)" || echo "$(RED)✗$(NC)"
	@echo -n "curl: "; command -v curl >/dev/null 2>&1 && echo "$(GREEN)✓$(NC)" || echo "$(RED)✗$(NC)"
	@echo -n "Git: "; command -v git >/dev/null 2>&1 && echo "$(GREEN)✓$(NC)" || echo "$(RED)✗$(NC)"
	@echo -n "Go: "; command -v go >/dev/null 2>&1 && echo "$(GREEN)✓$(NC)" || echo "$(RED)✗$(NC)"

# =============================================================================
# WORKFLOW SIMULATION TARGETS
# =============================================================================

ci: ## Run CI workflow locally
	@echo "$(BLUE)Running CI workflow...$(NC)"
	$(MAKE) check-tools
	$(MAKE) test
	$(MAKE) validate-dockerfiles
	@echo "$(GREEN)✓ CI workflow completed$(NC)"

ci-build: ## Run CI build workflow locally
	@echo "$(BLUE)Running CI build workflow...$(NC)"
	$(MAKE) ci
	$(MAKE) build
	$(MAKE) build-run-mcp
	@echo "$(GREEN)✓ CI build workflow completed$(NC)"

ci-matrix: ## Run CI matrix build workflow locally
	@echo "$(BLUE)Running CI matrix build workflow...$(NC)"
	$(MAKE) ci
	$(MAKE) build-matrix
	$(MAKE) build-run-mcp-all
	@echo "$(GREEN)✓ CI matrix build workflow completed$(NC)"

# =============================================================================
# UTILITY TARGETS
# =============================================================================

clean: ## Clean up temporary files and build artifacts
	@echo "$(BLUE)Cleaning up...$(NC)"
	@rm -rf build/
	@rm -f /tmp/makefile-*
	@echo "$(GREEN)✓ Cleanup completed$(NC)"

info: ## Show build information
	@echo "$(BLUE)Build Information:$(NC)"
	@echo "  Registry: $(REGISTRY)"
	@echo "  Owner: $(OWNER)"
	@echo "  Repository: $(REPO_NAME)"
	@echo "  Branch: $(BRANCH)"
	@echo "  Commit: $(COMMIT_SHA)"
	@echo "  Date: $(DATE)"
	@echo "  Container Runtime: $(DOCKER_CMD)"
	@echo ""
	@echo "$(BLUE)Image Names:$(NC)"
	@echo "  Node.js: $(NODEJS_IMAGE)"
	@echo "  Python: $(PYTHON_IMAGE)"
	@echo ""
	@echo "$(BLUE)Supported Versions:$(NC)"
	@echo "  Node.js: $(NODEJS_VERSIONS)"
	@echo "  Python: $(PYTHON_VERSIONS)"

# =============================================================================
# CONTAINER LIFECYCLE MANAGEMENT
# =============================================================================

containers: ## Complete container lifecycle: cleanup → build → push (latest versions)
	@echo "$(BLUE)Running complete container lifecycle...$(NC)"
	@echo "$(YELLOW)Step 1: Cleaning up old versions$(NC)"
	$(MAKE) cleanup-versions
	@echo "$(YELLOW)Step 2: Building latest versions$(NC)"
	$(MAKE) build
	@echo "$(YELLOW)Step 3: Pushing to registry$(NC)"
	$(MAKE) push
	@echo "$(GREEN)✅ Container lifecycle completed$(NC)"

containers-matrix: ## Complete container lifecycle: cleanup → build → push (all matrix versions)
	@echo "$(BLUE)Running complete container lifecycle with matrix builds...$(NC)"
	@echo "$(YELLOW)Step 1: Cleaning up old versions$(NC)"
	$(MAKE) cleanup-versions
	@echo "$(YELLOW)Step 2: Building all matrix versions$(NC)"
	$(MAKE) build-matrix
	@echo "$(YELLOW)Step 3: Pushing all matrix versions$(NC)"
	$(MAKE) push-matrix
	@echo "$(GREEN)✅ Matrix container lifecycle completed$(NC)"

# =============================================================================
# ALL-IN-ONE TARGETS
# =============================================================================

all: ## Build everything (containers and binary)
	@echo "$(BLUE)Building everything...$(NC)"
	$(MAKE) build
	$(MAKE) build-run-mcp
	@echo "$(GREEN)✓ Built all components$(NC)"

all-matrix: ## Build everything with matrix versions
	@echo "$(BLUE)Building everything with matrix versions...$(NC)"
	$(MAKE) build-matrix
	$(MAKE) build-run-mcp-all
	@echo "$(GREEN)✓ Built all components with matrix versions$(NC)"

release: ## Full release process (CI → cleanup → build → push)
	@echo "$(BLUE)Running full release process...$(NC)"
	$(MAKE) ci
	$(MAKE) containers
	$(MAKE) build-run-mcp-all
	@echo "$(GREEN)✅ Release process completed$(NC)"

release-matrix: ## Full release process with matrix builds (CI → cleanup → build → push)
	@echo "$(BLUE)Running full release process with matrix builds...$(NC)"
	$(MAKE) ci
	$(MAKE) containers-matrix
	$(MAKE) build-run-mcp-all
	@echo "$(GREEN)✅ Matrix release process completed$(NC)"