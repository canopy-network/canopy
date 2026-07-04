# Docker Build Optimization Guide

## Overview

This guide explains the optimizations available for building Canopy containers more efficiently.

## Current Dockerfile Issues

The current `.docker/Dockerfile` has several performance issues:

1. **No dependency caching** - Downloads all Go modules on every build
2. **No build cache** - Go build cache is lost between builds
3. **Poor layer ordering** - Full source copy invalidates cache on any file change
4. **Slow rebuilds** - Even small code changes trigger full rebuilds

## Optimization Levels

### Level 1: Optimized Dockerfile (Recommended)

**File**: `.docker/Dockerfile.optimized`

**Speed improvement**: 3-5x faster rebuilds

**Key optimizations**:
- Separate `go.mod`/`go.sum` copy for dependency caching
- Mount Go module cache (`/go/pkg/mod`)
- Mount Go build cache (`/root/.cache/go-build`)
- Proper layer ordering

**Build command**:
```bash
docker build -f .docker/Dockerfile.optimized -t canopy-node:latest .
```

**Cache behavior**:
- ✅ Dependencies cached until `go.mod`/`go.sum` changes
- ✅ Build cache persists across builds
- ✅ Only rebuilds changed packages
- ⚠️ Still copies full source tree

---

### Level 2: Ultra-Optimized Dockerfile (Maximum Performance)

**File**: `.docker/Dockerfile.ultra-optimized`

**Speed improvement**: 5-10x faster rebuilds

**Additional optimizations**:
- Selective source copying (only required directories)
- Static binary compilation (`CGO_ENABLED=0`)
- Binary size reduction (`-ldflags="-s -w"`)
- Minimal `FROM scratch` runtime (smaller image)
- Build trimming for reproducible builds

**Build command**:
```bash
docker build -f .docker/Dockerfile.ultra-optimized -t canopy-node:latest .
```

**Trade-offs**:
- ✅ Fastest possible builds
- ✅ Smallest possible image (~10-20MB)
- ⚠️ `FROM scratch` means no shell/debugging tools
- ⚠️ May need adjustments if runtime dependencies required

---

## Detailed Optimizations Explained

### 1. Dependency Layer Caching

**Before**:
```dockerfile
COPY . .
RUN go build -a -o bin ./cmd/main/...
```

**After**:
```dockerfile
# Only copy dependency files first
COPY go.mod go.sum ./
RUN go mod download

# Then copy source code
COPY . .
RUN go build -a -o bin ./cmd/main/...
```

**Benefit**: Dependencies only re-download when `go.mod` or `go.sum` changes.

---

### 2. Build Cache Mounts

**Before**:
```dockerfile
RUN go build -a -o bin ./cmd/main/...
```

**After**:
```dockerfile
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -o bin ./cmd/main/...
```

**Benefit**:
- Go build cache persists between builds
- Modules cached even across different builds
- Incremental compilation instead of full rebuilds
- Requires BuildKit (enabled by default in Docker 23.0+)

---

### 3. Selective Source Copying

**Before**:
```dockerfile
COPY . .
```

**After**:
```dockerfile
COPY cmd ./cmd
COPY bft ./bft
COPY cache ./cache
# ... only what's needed
```

**Benefit**:
- Excludes test files, docs, `.git`, etc.
- Smaller build context
- More predictable cache invalidation

---

### 4. Build Optimizations

```dockerfile
CGO_ENABLED=0 \           # Static binary, no C dependencies
go build \
  -trimpath \             # Remove build paths for reproducibility
  -ldflags="-s -w" \      # Strip debug info and symbol table
  -o bin \
  ./cmd/main/...
```

**Benefits**:
- Smaller binary size (30-50% reduction)
- Truly static binary (runs on `FROM scratch`)
- Faster builds with `-trimpath`
- Better security (no debug symbols)

---

### 5. Multi-Stage Build Runtime

**Standard** (alpine:3.19):
```dockerfile
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /path/to/bin ./bin
```
- Size: ~8-15MB base + your binary
- Includes: shell, package manager, debugging tools
- Good for: development, debugging

**Minimal** (scratch):
```dockerfile
FROM scratch
WORKDIR /app
COPY --from=builder /path/to/bin /app/bin
```
- Size: ~0MB base + your binary only
- Includes: nothing
- Good for: production, minimal attack surface

---

## Performance Comparison

### First Build (No Cache)

| Version | Time | Image Size |
|---------|------|------------|
| Original | ~5-8 min | ~150-200MB |
| Optimized | ~4-6 min | ~150-200MB |
| Ultra-Optimized | ~3-5 min | ~10-30MB |

### Rebuild (Code Change Only)

| Version | Time | Image Size |
|---------|------|------------|
| Original | ~5-8 min | ~150-200MB |
| Optimized | ~30-90s | ~150-200MB |
| Ultra-Optimized | ~20-60s | ~10-30MB |

### Rebuild (No Changes)

| Version | Time |
|---------|------|
| Original | ~2-3 min |
| Optimized | ~5-10s |
| Ultra-Optimized | ~3-5s |

---

## Migration Guide

### Option A: Replace Current Dockerfile

```bash
# Backup current
cp .docker/Dockerfile .docker/Dockerfile.backup

# Replace with optimized version
cp .docker/Dockerfile.optimized .docker/Dockerfile

# Test build
docker build -f .docker/Dockerfile -t canopy-node:test .
```

### Option B: Use Alongside Current

```bash
# Build with optimized version
docker build -f .docker/Dockerfile.optimized -t canopy-node:latest .

# Update oracle-compose.yaml to use new image name if needed
```

### Option C: Gradual Migration

1. Start with `Dockerfile.optimized` for local dev
2. Test thoroughly
3. Update CI/CD pipelines
4. Switch production builds

---

## BuildKit Requirements

The cache mounts (`--mount=type=cache`) require Docker BuildKit.

### Enable BuildKit

**Temporary** (single build):
```bash
DOCKER_BUILDKIT=1 docker build -f .docker/Dockerfile.optimized -t canopy-node:latest .
```

**Permanent** (recommended):
```bash
# Add to ~/.bashrc or ~/.zshrc
export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1
```

**Or configure Docker daemon**:
```json
// /etc/docker/daemon.json
{
  "features": {
    "buildkit": true
  }
}
```

### Check if BuildKit is Available

```bash
docker buildx version
# Should output buildx version info
```

---

## Best Practices

### 1. Use .dockerignore

Ensure `.dockerignore` excludes unnecessary files:

```
.git
.github
*.md
.docker
*.tar
*.tar.gz
volumes/
node_modules/
*.test
coverage.txt
```

### 2. Prune Build Cache Periodically

```bash
# Remove unused build cache
docker builder prune

# Remove all cache
docker builder prune -a

# Check cache usage
docker system df
```

### 3. Layer Ordering Best Practices

Order layers from least to most frequently changed:

1. Base image and system packages
2. Go module dependencies (`go.mod`, `go.sum`)
3. Source code
4. Build step

### 4. Multi-Architecture Builds

For building multiple architectures efficiently:

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f .docker/Dockerfile.optimized \
  -t canopy-node:latest \
  --push \
  .
```

---

## Troubleshooting

### Cache Not Working

**Check BuildKit is enabled**:
```bash
docker buildx inspect default | grep Driver
# Should show "Driver: docker-container" or "Driver: docker"
```

**Force clean build**:
```bash
docker build --no-cache -f .docker/Dockerfile.optimized -t canopy-node:latest .
```

### "FROM scratch" Runtime Issues

If you need shell access or debugging:

**Option 1**: Use `alpine:3.19` instead of `scratch`
```dockerfile
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
```

**Option 2**: Use debug variants
```dockerfile
FROM gcr.io/distroless/static-debian11
```

### Module Download Failures

If behind proxy or in restricted network:

```dockerfile
# Set proxy during build
RUN --mount=type=cache,target=/go/pkg/mod \
    HTTP_PROXY=http://proxy:8080 \
    HTTPS_PROXY=http://proxy:8080 \
    go mod download
```

---

## Integration with Deployment Script

Update `.docker/deploy.sh` to use optimized Dockerfile:

```bash
# Option 1: Add flag to deploy.sh
./docker/deploy.sh -h remote-host --dockerfile Dockerfile.optimized

# Option 2: Set default in deploy.sh
DOCKERFILE="${DOCKERFILE:-.docker/Dockerfile.optimized}"
```

---

## Measuring Build Performance

### Time a build:
```bash
time docker build -f .docker/Dockerfile.optimized -t canopy-node:latest .
```

### Check cache usage:
```bash
docker system df -v | grep buildcache
```

### Analyze build layers:
```bash
docker history canopy-node:latest
```

### Use dive tool to inspect image:
```bash
# Install dive
brew install dive  # macOS
# or download from: https://github.com/wagoodman/dive

# Analyze image
dive canopy-node:latest
```

---

## Recommendations

1. **For Development**: Use `Dockerfile.optimized` with `alpine:3.19` base
   - Fast rebuilds
   - Shell access for debugging
   - Familiar tooling

2. **For Production**: Use `Dockerfile.ultra-optimized` with `FROM scratch`
   - Smallest attack surface
   - Minimal image size
   - Fastest runtime startup

3. **For CI/CD**: Use `Dockerfile.optimized`
   - Reliable caching in CI environments
   - Faster pipeline execution
   - Good balance of speed and compatibility

---

## Next Steps

1. Choose optimization level (optimized or ultra-optimized)
2. Enable BuildKit globally
3. Test build performance with your codebase
4. Update `oracle-compose.yaml` if using different Dockerfile
5. Update CI/CD pipelines
6. Monitor build times and adjust as needed
