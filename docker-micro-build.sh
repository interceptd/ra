#!/bin/bash

# Ultra-Minimal Docker Build Script
# Builds the smallest possible repo-analyzer container

set -e

# Default to native platform, but allow override
PLATFORM=${PLATFORM:-"linux/$(uname -m)"}
BUILD_ALL_PLATFORMS=${BUILD_ALL_PLATFORMS:-false}

echo "🚀 Repository Analyzer - Ultra-Minimal Docker Build"
echo "================================================="

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker."
    exit 1
fi

# Show build configuration
echo "📋 Build Configuration:"
echo "   Platform: $PLATFORM"
echo "   Build All Platforms: $BUILD_ALL_PLATFORMS"
echo ""

# Function to build for specific platform
build_for_platform() {
    local platform=$1
    local arch_suffix=$2
    local go_arch=$3
    
    echo "🔨 Building for $platform (GOARCH=$go_arch)..."
    
    # Cross-compile Go binary for the target platform
    CGO_ENABLED=0 GOOS=linux GOARCH=$go_arch go build -ldflags="-w -s -extldflags '-static'" -a -installsuffix cgo -o repo-analyzer-$arch_suffix .
    
    # Verify binary was created
    if [ ! -f repo-analyzer-$arch_suffix ]; then
        echo "❌ Failed to build Go binary for $go_arch"
        return 1
    fi
    
    echo "✅ Linux binary created for $go_arch ($(ls -lh repo-analyzer-$arch_suffix | awk '{print $5}'))"
    
    # Copy the binary to the standard name for Docker build
    cp repo-analyzer-$arch_suffix repo-analyzer
    
    # Build Docker image
    echo "🐳 Building Docker image for $platform..."
    docker build --platform=$platform -f Dockerfile.micro -t repo-analyzer:micro-$arch_suffix .
    
    # Clean up platform-specific binary
    rm -f repo-analyzer-$arch_suffix
}

# Create minimal .dockerignore
cat > .dockerignore.micro <<EOF
# Ignore everything except what we need
*
!repo-analyzer
!repo-analyzer.config.yml
!Dockerfile.micro
EOF

# Build for specific platforms
if [ "$BUILD_ALL_PLATFORMS" = "true" ]; then
    echo "🌍 Building for all platforms..."
    
    build_for_platform "linux/amd64" "amd64" "amd64"
    build_for_platform "linux/arm64" "arm64" "arm64"
    
    # Create multi-platform manifest
    echo "🔗 Creating multi-platform manifest..."
    docker manifest create repo-analyzer:micro \
        repo-analyzer:micro-amd64 \
        repo-analyzer:micro-arm64
    
    docker manifest annotate repo-analyzer:micro repo-analyzer:micro-amd64 --os linux --arch amd64
    docker manifest annotate repo-analyzer:micro repo-analyzer:micro-arm64 --os linux --arch arm64
    
    echo "✅ Multi-platform image created as repo-analyzer:micro"
    
else
    # Build for single platform (native or specified)
    case $PLATFORM in
        "linux/amd64")
            build_for_platform "linux/amd64" "amd64" "amd64"
            docker tag repo-analyzer:micro-amd64 repo-analyzer:micro
            ;;
        "linux/arm64")
            build_for_platform "linux/arm64" "arm64" "arm64"
            docker tag repo-analyzer:micro-arm64 repo-analyzer:micro
            ;;
        "linux/x86_64")
            # Handle x86_64 as amd64
            build_for_platform "linux/amd64" "amd64" "amd64"
            docker tag repo-analyzer:micro-amd64 repo-analyzer:micro
            ;;
        "linux/aarch64")
            # Handle aarch64 as arm64
            build_for_platform "linux/arm64" "arm64" "arm64"
            docker tag repo-analyzer:micro-arm64 repo-analyzer:micro
            ;;
        *)
            echo "⚠️  Unsupported platform: $PLATFORM"
            echo "   Defaulting to linux/amd64"
            build_for_platform "linux/amd64" "amd64" "amd64"
            docker tag repo-analyzer:micro-amd64 repo-analyzer:micro
            ;;
    esac
fi

# Clean up
rm -f repo-analyzer .dockerignore.micro

# Show final image info
echo ""
echo "📊 Docker Image Information:"
docker images repo-analyzer:micro --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"

# Test the container quickly
echo ""
echo "🧪 Quick Container Test:"
if docker run --rm repo-analyzer:micro repo-analyzer --version 2>/dev/null; then
    echo "✅ Container test passed!"
else
    echo "⚠️  Container test failed, but image was built"
fi

echo ""
echo "🎉 Build completed successfully!"
echo ""
echo "Usage examples:"
echo "  # Run setup check"
echo "  docker run --rm repo-analyzer:micro setup --quiet"
echo ""
echo "  # Analyze current directory"
echo "  docker run --rm -v \$(pwd):/workspace repo-analyzer:micro analyze /workspace"
echo ""
echo "  # Build for all platforms:"
echo "  BUILD_ALL_PLATFORMS=true ./docker-micro-build.sh"
echo ""
echo "  # Build for specific platform:"
echo "  PLATFORM=linux/arm64 ./docker-micro-build.sh" 