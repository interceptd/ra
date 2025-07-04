# Repository Analyzer Makefile
# Simple build automation for Go CLI tool

.PHONY: build clean test docker-build docker-build-all docker-test docker-clean help

# Default target
all: build

# Show help
help:
	@echo "Repository Analyzer - Build Commands"
	@echo "===================================="
	@echo "build              - Build the Go binary"
	@echo "clean              - Clean build artifacts"
	@echo "test               - Run tests"
	@echo "docker-build       - Build minimal Docker container (native platform)"
	@echo "docker-build-all   - Build Docker container for all platforms"
	@echo "docker-build-arm64 - Build Docker container for ARM64"
	@echo "docker-build-amd64 - Build Docker container for AMD64"
	@echo "docker-test        - Test Docker container"
	@echo "docker-clean       - Clean Docker images"
	@echo "help               - Show this help"

# Build the Go binary
build:
	@echo "🔨 Building repo-analyzer..."
	go build -ldflags "-w -s" -o repo-analyzer .
	@echo "✅ Built repo-analyzer"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -f repo-analyzer repo-analyzer-*
	@echo "✅ Cleaned"

# Run tests
test:
	@echo "🧪 Running tests..."
	go test ./...
	@echo "✅ Tests completed"

# Build minimal Docker container (native platform)
docker-build:
	@echo "🐳 Building Docker container for native platform..."
	./docker-micro-build.sh

# Build Docker container for all platforms
docker-build-all:
	@echo "🌍 Building Docker container for all platforms..."
	BUILD_ALL_PLATFORMS=true ./docker-micro-build.sh

# Build Docker container for ARM64
docker-build-arm64:
	@echo "🔧 Building Docker container for ARM64..."
	PLATFORM=linux/arm64 ./docker-micro-build.sh

# Build Docker container for AMD64
docker-build-amd64:
	@echo "🔧 Building Docker container for AMD64..."
	PLATFORM=linux/amd64 ./docker-micro-build.sh

# Test Docker container
docker-test:
	@echo "🧪 Testing Docker container..."
	@echo "Testing help command..."
	docker run --rm repo-analyzer:micro --help
	@echo ""
	@echo "Testing setup command..."
	docker run --rm repo-analyzer:micro setup --quiet
	@echo ""
	@echo "✅ Docker tests completed"

# Clean Docker images
docker-clean:
	@echo "🧹 Cleaning Docker images..."
	docker rmi -f repo-analyzer:micro repo-analyzer:micro-amd64 repo-analyzer:micro-arm64 2>/dev/null || true
	docker system prune -f
	@echo "✅ Docker cleanup completed" 