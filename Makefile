# PHOENIX PANEL — developer tasks
# Usage: make <target>

VERSION ?= dev
BINARY  := phoenix
PKG     := ./cmd/phoenix

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: tidy
tidy: ## Sync go.mod / go.sum
	go mod tidy

.PHONY: build
build: ## Build the server binary into ./bin
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o bin/$(BINARY) $(PKG)

.PHONY: run
run: ## Run the server locally (uses .env)
	go run $(PKG)

.PHONY: test
test: ## Run unit tests
	go test ./... -race -count=1

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: fmt
fmt: ## Format code
	gofmt -s -w .

.PHONY: lint
lint: ## Run golangci-lint (must be installed)
	golangci-lint run ./...

.PHONY: docker-build
docker-build: ## Build the Docker image
	docker build --build-arg VERSION=$(VERSION) -t phoenix-panel:$(VERSION) .

.PHONY: up
up: ## Start the full stack via docker compose
	docker compose up -d --build

.PHONY: down
down: ## Stop the stack
	docker compose down

.PHONY: logs
logs: ## Tail panel logs
	docker compose logs -f panel
