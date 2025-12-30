# Makefile for MCP Container Images
# Manages testing, building, and publishing container images

# Configuration
REGISTRY := ghcr.io
OWNER := $(shell git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\).*/\1/')
REPO_NAME := $(shell basename `git rev-parse --show-toplevel`)
COMMIT_SHA := $(shell git rev-parse --short HEAD)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
DATE := $(shell date +%Y%m%d)

# Image names and tags
NODEJS_IMAGE := $(REGISTRY)/$(OWNER)/mcp-nodejs
PYTHON_IMAGE := $(REGISTRY)/$(OWNER)/mcp-python

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

build-nodejs: ## Build Node.js container image
	@echo "$(BLUE)Building Node.js container image...$(NC)"
	@if [ ! -f nodejs/Dockerfile ]; then \
		echo "$(RED)Error: nodejs/Dockerfile not found$(NC)"; \
		exit 1; \
	fi
	docker build \
		--platform linux/amd64,linux/arm64 \
		--tag $(NODEJS_IMAGE):$(NODEJS_TAG) \
		--tag $(NODEJS_IMAGE):$(NODEJS_PINNED_TAG) \
		--tag $(NODEJS_IMAGE):latest \
		nodejs/
	@echo "$(GREEN)✓ Node.js image built: $(NODEJS_IMAGE):$(NODEJS_TAG)$(NC)"

build-python: ## Build Python container image
	@echo "$(BLUE)Building Python container image...$(NC)"
	@if [ ! -f python/Dockerfile ]; then \
		echo "$(RED)Error: python/Dockerfile not found$(NC)"; \
		exit 1; \
	fi
	docker build \
		--platform linux/amd64,linux/arm64 \
		--tag $(PYTHON_IMAGE):$(PYTHON_TAG) \
		--tag $(PYTHON_IMAGE):$(PYTHON_PINNED_TAG) \
		--tag $(PYTHON_IMAGE):latest \
		python/
	@echo "$(GREEN)✓ Python image built: $(PYTHON_IMAGE):$(PYTHON_TAG)$(NC)"

# Registry targets
login: ## Login to GitHub Container Registry
	@echo "$(BLUE)Logging into GitHub Container Registry...$(NC)"
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "$(RED)Error: GITHUB_TOKEN environment variable not set$(NC)"; \
		echo "$(YELLOW)Set it with: export GITHUB_TOKEN=your_token$(NC)"; \
		exit 1; \
	fi
	@echo "$$GITHUB_TOKEN" | docker login $(REGISTRY) -u $(OWNER) --password-stdin
	@echo "$(GREEN)✓ Logged into $(REGISTRY)$(NC)"

push: login push-nodejs push-python ## Push all images to registry

push-nodejs: ## Push Node.js image to registry
	@echo "$(BLUE)Pushing Node.js image to registry...$(NC)"
	docker push $(NODEJS_IMAGE):$(NODEJS_TAG)
	docker push $(NODEJS_IMAGE):$(NODEJS_PINNED_TAG)
	docker push $(NODEJS_IMAGE):latest
	@echo "$(GREEN)✓ Node.js image pushed to $(REGISTRY)$(NC)"

push-python: ## Push Python image to registry
	@echo "$(BLUE)Pushing Python image to registry...$(NC)"
	docker push $(PYTHON_IMAGE):$(PYTHON_TAG)
	docker push $(PYTHON_IMAGE):$(PYTHON_PINNED_TAG)
	docker push $(PYTHON_IMAGE):latest
	@echo "$(GREEN)✓ Python image pushed to $(REGISTRY)$(NC)"

# Development targets
setup-dev: ## Set up development environment
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@echo "$(YELLOW)Installing required tools...$(NC)"
	@if ! command -v bats >/dev/null 2>&1; then \
		echo "Installing bats..."; \
		sudo apt-get update && sudo apt-get install -y bats; \
	fi
	@if ! command -v hadolint >/dev/null 2>&1; then \
		echo "Installing hadolint..."; \
		wget -O /tmp/hadolint https://github.com/hadolint/hadolint/releases/latest/download/hadolint-Linux-x86_64; \
		chmod +x /tmp/hadolint; \
		sudo mv /tmp/hadolint /usr/local/bin/hadolint; \
	fi
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "$(RED)Warning: Docker not found. Please install Docker.$(NC)"; \
	fi
	@echo "$(GREEN)✓ Development environment setup complete$(NC)"

check-tools: ## Check if required tools are installed
	@echo "$(BLUE)Checking required tools...$(NC)"
	@echo -n "Docker: "; command -v docker >/dev/null 2>&1 && echo "$(GREEN)✓$(NC)" || echo "$(RED)✗$(NC)"
	@echo -n "Bats: "; command -v bats >/dev/null 2>&1 && echo "$(GREEN)✓$(NC)" || echo "$(RED)✗$(NC)"
	@echo -n "Hadolint: "; command -v hadolint >/dev/null 2>&1 && echo "$(GREEN)✓$(NC)" || echo "$(RED)✗$(NC)"
	@echo -n "Git: "; command -v git >/dev/null 2>&1 && echo "$(GREEN)✓$(NC)" || echo "$(RED)✗$(NC)"

# Validation targets
lint: validate-dockerfiles ## Run all linting checks

validate-dockerfiles: ## Validate Dockerfiles with hadolint
	@echo "$(BLUE)Validating Dockerfiles...$(NC)"
	@if command -v hadolint >/dev/null 2>&1; then \
		if [ -f nodejs/Dockerfile ]; then \
			echo "Linting nodejs/Dockerfile..."; \
			hadolint nodejs/Dockerfile; \
		fi; \
		if [ -f python/Dockerfile ]; then \
			echo "Linting python/Dockerfile..."; \
			hadolint python/Dockerfile; \
		fi; \
		echo "$(GREEN)✓ Dockerfile validation completed$(NC)"; \
	else \
		echo "$(YELLOW)Warning: hadolint not found, skipping Dockerfile validation$(NC)"; \
	fi

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
	@docker image prune -f || true
	@echo "$(GREEN)✓ Cleanup completed$(NC)"

clean-all: ## Clean up everything including built images
	@echo "$(BLUE)Cleaning up everything...$(NC)"
	$(MAKE) clean
	@echo "$(YELLOW)Removing built images...$(NC)"
	@docker rmi $(NODEJS_IMAGE):$(NODEJS_TAG) $(NODEJS_IMAGE):latest || true
	@docker rmi $(PYTHON_IMAGE):$(PYTHON_TAG) $(PYTHON_IMAGE):latest || true
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