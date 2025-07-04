# Repository Analyzer

A unified CLI tool to analyze and package repositories for AI consumption.

## Features

- **Unified Analysis**: Run gitingest (Python) for detailed repository analysis
- **AI-Friendly Packaging**: Run repomix (Node.js) for AI-optimized repository packaging
- **Ultra-Fast Docker**: Optimized container with **1.5-second** cached builds (vs 10-minute original)
- **Minimal Size**: 539MB production-ready container with all dependencies
- **Unified Configuration**: Single YAML config file for all tools
- **Text Cleaning**: Removes Unicode line terminators and normalizes content
- **Virtual Environment**: Isolated Python environment for dependencies
- **Cross-Platform**: Works on macOS, Linux, and Windows
- **ARM64 + AMD64**: Multi-platform Docker support

## Quick Start

### Using Docker (Recommended)

```bash
# Build the container (first time ~2 minutes, subsequent ~1.5 seconds)
make docker-build

# Run analysis on current directory
docker run --rm -v $(pwd)/workspace:/workspace repo-analyzer:micro analyze intercept

# Run setup check
docker run --rm repo-analyzer:micro setup --quiet

# Run specific tool only
docker run --rm -v $(pwd)/workspace:/workspace repo-analyzer:micro analyze intercept --disable-repomix
```

### Multi-Platform Docker Build

```bash
# Build for ARM64 (Apple Silicon)
make docker-build-arm64

# Build for AMD64 (Intel)
make docker-build-amd64

# Build for all platforms
make docker-build-all
```

### Using Local Installation

```bash
# Install dependencies
./repo-analyzer setup --auto-install

# Run analysis
./repo-analyzer analyze /path/to/repository

# Run specific tools
./repo-analyzer gitingest /path/to/repository
./repo-analyzer repomix /path/to/repository
```

## Commands

### Core Commands

- `analyze` - Run both gitingest and repomix analysis
- `gitingest` - Run gitingest analysis only
- `repomix` - Run repomix analysis only
- `setup` - Install and verify dependencies

### Analysis Example

```bash
# Analyze current repository
docker run --rm -v $(pwd):/workspace repo-analyzer:micro analyze /workspace

# Custom output directory
docker run --rm -v $(pwd):/workspace repo-analyzer:micro analyze /workspace --output /workspace/results

# Disable specific tools
docker run --rm -v $(pwd):/workspace repo-analyzer:micro analyze /workspace --disable-gitingest
```

## Configuration

The tool uses `repo-analyzer.config.yml` for configuration:

```yaml
# Global settings
global:
  output_dir: "analysis-results"
  timestamp_format: "20060102_150405"

# Gitingest settings
gitingest:
  include_patterns: ["*.go", "*.js", "*.py", "*.md"]
  exclude_patterns: ["node_modules", ".git", "*.log"]

# Repomix settings
repomix:
  output_format: "markdown"
  include_file_summary: true
  include_directory_structure: true
  show_line_numbers: true
  remove_comments: true
  remove_empty_lines: true
```

## Docker Architecture

The Docker implementation uses your suggested approach:

1. **Builder Stage**: Compiles Go binary + installs all dependencies
2. **Setup Stage**: Runs `repo-analyzer setup` to create perfect environment
3. **Runtime Stage**: Copies complete environment for instant startup

```dockerfile
# Builder: Install everything
RUN /app/source/repo-analyzer setup --auto-install --verbose || true

# Runtime: Copy complete environment
COPY --from=builder /app/.venv /app/.venv
COPY --from=builder /usr/local/lib/node_modules /usr/local/lib/node_modules
COPY --from=builder /usr/local/bin/repomix /usr/local/bin/repomix
```

## Build System

```bash
# Development
make build          # Build Go binary
make clean          # Clean artifacts
make test           # Run tests

# Docker
make docker-build   # Build container
make docker-test    # Test container
make docker-clean   # Clean images

# Multi-platform
make docker-build-arm64   # ARM64 build
make docker-build-amd64   # AMD64 build
make docker-build-all     # All platforms
```

## Output Files

The tool generates timestamped analysis files:

### Gitingest Output

- `*_summary_*.txt` - Repository summary
- `*_tree_*.txt` - Directory structure
- `*_content_*.txt` - Full content analysis
- `*_results_*.json` - Structured JSON data

### Repomix Output

- `*_repomix_*.md` - AI-friendly markdown package
