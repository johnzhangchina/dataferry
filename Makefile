VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
BINARY := dataferry
FRONTEND := web/frontend

.PHONY: all build frontend clean test dev release

all: build

## Development

dev: ## Run in dev mode (backend + frontend hot reload)
	@echo "Starting backend..."
	@go run ./cmd/dataferry &
	@echo "Starting frontend dev server..."
	@cd $(FRONTEND) && npm run dev

test: ## Run all tests
	go test ./...

## Build

frontend: ## Build frontend
	cd $(FRONTEND) && npm ci && npm run build

build: frontend ## Build single binary (current platform)
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BINARY) ./cmd/dataferry

## Cross-compilation (requires cross-compile toolchain for CGO)

build-linux-amd64: frontend
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-linux-musl-gcc \
		go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 ./cmd/dataferry

build-linux-arm64: frontend
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-musl-gcc \
		go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 ./cmd/dataferry

build-darwin-amd64: frontend
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 \
		go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 ./cmd/dataferry

build-darwin-arm64: frontend
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 \
		go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 ./cmd/dataferry

release: frontend ## Build all platforms (requires cross-compile toolchains)
	@mkdir -p dist
	$(MAKE) build-darwin-amd64
	$(MAKE) build-darwin-arm64
	@echo "Note: Linux builds require musl cross-compile toolchains."
	@echo "Install via: brew install FiloSottile/musl-cross/musl-cross"
	@echo "Then run: make build-linux-amd64 build-linux-arm64"
	@ls -lh dist/

## Docker

docker: ## Build Docker image
	docker build -t dataferry:$(VERSION) -t dataferry:latest .

docker-run: docker ## Build and run Docker container
	docker run -d -p 8080:8080 -v dataferry-data:/app/data --name dataferry dataferry:latest

## Cleanup

clean: ## Remove build artifacts
	rm -rf $(BINARY) dist/
	rm -rf $(FRONTEND)/dist $(FRONTEND)/node_modules

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
