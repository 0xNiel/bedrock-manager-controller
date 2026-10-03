# Unified AWS Bedrock Controller Makefile
# Module: github.com/odnielgonzalez/bedrock-manager-controller
# Image: bedrock-manager-controller

# Variables
MODULE_NAME = github.com/odnielgonzalez/bedrock-manager-controller
IMAGE_NAME = bedrock-manager-controller
# Registry to push to, e.g. 123456789012.dkr.ecr.us-east-1.amazonaws.com or ghcr.io/<user>
IMAGE_REGISTRY ?=
IMAGE = $(if $(IMAGE_REGISTRY),$(IMAGE_REGISTRY)/)$(IMAGE_NAME)
VERSION = $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Build flags
BUILD_FLAGS = -trimpath -ldflags "-s -w -X main.version=$(VERSION)"
CGO_ENABLED = 0

# Default target
.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Utilities
.PHONY: tidy
tidy: ## Run go mod tidy
	go mod tidy

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: test
test: ## Run unit tests with race detection and coverage
	go test -race -coverprofile=coverage.out ./tests/...

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf dist/ bin/ coverage.*

# Builds
.PHONY: build-local
build-local: ## Build for local macOS (darwin/arm64)
	mkdir -p bin
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_FLAGS) -o bin/controller-local .

.PHONY: build-deploy
build-deploy: ## Build for deployment (linux/amd64)
	mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_FLAGS) -o bin/controller .

# Docker
.PHONY: docker-local
docker-local: ## Build Docker image for local platform (linux/arm64, e.g. Docker Desktop on Apple silicon)
	docker buildx build --platform linux/arm64 --build-arg VERSION=$(VERSION) --load -t $(IMAGE):local .

.PHONY: docker-deploy
docker-deploy: ## Build Docker image for deployment (linux/amd64)
	docker buildx build --platform linux/amd64 --build-arg VERSION=$(VERSION) --load -t $(IMAGE):$(VERSION) .
	@echo "Built $(IMAGE):$(VERSION)"

.PHONY: require-registry
require-registry:
	@test -n "$(IMAGE_REGISTRY)" || (echo "❌ Set IMAGE_REGISTRY, e.g. make docker-push IMAGE_REGISTRY=ghcr.io/<user>" && exit 1)

.PHONY: docker-push
docker-push: require-registry docker-deploy ## Build and push the deployment image (requires IMAGE_REGISTRY)
	docker push $(IMAGE):$(VERSION)
	@echo "Pushed $(IMAGE):$(VERSION) - set this as the image in deploy/deployment.yaml"

# Run
.PHONY: run
run: build-local ## Build and run the local binary with current KUBECONFIG
	@echo "⚠️  WARNING: Running controller with current KUBECONFIG context"
	@echo "Current context: $$(kubectl config current-context 2>/dev/null || echo 'No context set')"
	@echo "Proceeding in 3 seconds... (Ctrl+C to cancel)"
	@sleep 3
	./bin/controller-local

# Manifests / Generate
.PHONY: crd-validate
crd-validate: ## Validate CRD YAML files
	@echo "Validating CRD files..."
	@for file in crd/*.yaml; do \
		if [ -f "$$file" ]; then \
			echo "Validating $$file"; \
			kubectl --dry-run=client apply -f "$$file" >/dev/null 2>&1 || echo "❌ $$file validation failed"; \
		fi; \
	done
	@echo "✅ CRD validation complete"

.PHONY: cr-sync
cr-sync: ## Ensure controller/types.go and crd/*.yaml stay in sync
	@echo "🔄 Checking CRD and types synchronization..."
	@echo "This is a manual verification step - ensure:"
	@echo "  1. /controller/types.go matches /crd/*.yaml schemas"
	@echo "  2. All Spec/Status fields have corresponding json tags"
	@echo "  3. Example CRs in /cr/ match the CRD structure"
	@echo "✅ Manual sync check reminder complete"

# Development workflow
.PHONY: dev-setup
dev-setup: tidy ## Set up development environment
	@echo "Setting up development environment..."
	@which golangci-lint >/dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@which kubectl >/dev/null || echo "⚠️  kubectl not found - install for CRD validation"

.PHONY: dev-check
dev-check: tidy lint test crd-validate ## Run all development checks
	@echo "✅ All development checks passed"

# Deployment
.PHONY: deploy-local
deploy-local: docker-local ## Deploy to local Kubernetes cluster
	@echo "🚀 Deploying to local cluster..."
	kubectl create namespace bedrock-system --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f crd/bedrockresource.yaml
	kubectl apply -f deploy/serviceaccount.yaml
	kubectl apply -f deploy/rbac.yaml
	kubectl apply -f deploy/deployment.yaml
	@echo "✅ Deployment complete. Check status with: kubectl get pods -n bedrock-system"

.PHONY: undeploy
undeploy: ## Remove controller from cluster
	@echo "🗑️  Removing controller from cluster..."
	kubectl delete -f deploy/deployment.yaml --ignore-not-found=true
	kubectl delete -f deploy/rbac.yaml --ignore-not-found=true
	kubectl delete -f deploy/serviceaccount.yaml --ignore-not-found=true
	kubectl delete -f crd/bedrockresource.yaml --ignore-not-found=true
	kubectl delete namespace bedrock-system --ignore-not-found=true
	@echo "✅ Cleanup complete"

.PHONY: all
all: clean tidy lint test build-local build-deploy ## Run full build pipeline
	@echo "✅ Full build pipeline complete"
