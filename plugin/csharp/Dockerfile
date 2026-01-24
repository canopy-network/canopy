# Dockerfile for Canopy + C# Plugin
# Build from repository root: docker build -f plugin/csharp/Dockerfile -t canopy-csharp .

# Stage 1: Build Canopy (Go)
FROM golang:1.24-alpine AS canopy-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /canopy ./cmd/main/...

# Stage 2: Build C# Plugin (using Debian-based image for glibc compatibility with grpc.tools)
FROM mcr.microsoft.com/dotnet/sdk:8.0 AS csharp-builder
WORKDIR /app
COPY plugin/csharp/ ./
RUN dotnet publish CanopyPlugin.csproj -c Release -o /out

# Stage 3: Runtime
FROM mcr.microsoft.com/dotnet/runtime:8.0
WORKDIR /app
RUN mkdir -p /tmp/plugin /root/.canopy plugin/csharp

COPY --from=canopy-builder /canopy .
COPY --from=csharp-builder /out ./plugin/csharp/bin/

# Create pluginctl.sh for Docker environment
RUN printf '#!/bin/bash\n\
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"\n\
BINARY_PATH="$SCRIPT_DIR/bin/CanopyPlugin.dll"\n\
PID_FILE="/tmp/plugin/csharp-plugin.pid"\n\
LOG_FILE="/tmp/plugin/csharp-plugin.log"\n\
PLUGIN_DIR="/tmp/plugin"\n\
STOP_TIMEOUT=10\n\
is_running() {\n\
    if [ ! -f "$PID_FILE" ]; then return 1; fi\n\
    local pid=$(cat "$PID_FILE" 2>/dev/null)\n\
    if [ -z "$pid" ] || ! [[ "$pid" =~ ^[0-9]+$ ]]; then return 1; fi\n\
    if ps -p "$pid" > /dev/null 2>&1; then\n\
        if ps -p "$pid" -o cmd= | grep -q "CanopyPlugin.dll"; then return 0; fi\n\
    fi\n\
    return 1\n\
}\n\
cleanup_pid() { [ -f "$PID_FILE" ] && rm -f "$PID_FILE"; }\n\
start() {\n\
    if is_running; then echo "csharp-plugin is already running"; return 1; fi\n\
    cleanup_pid\n\
    mkdir -p "$PLUGIN_DIR"\n\
    echo "Starting csharp-plugin..."\n\
    nohup dotnet "$BINARY_PATH" > "$LOG_FILE" 2>&1 &\n\
    local pid=$!\n\
    echo "$pid" > "$PID_FILE"\n\
    sleep 1\n\
    if is_running; then echo "csharp-plugin started (PID: $pid)"; return 0\n\
    else echo "Error: csharp-plugin failed to start"; cleanup_pid; return 1; fi\n\
}\n\
stop() {\n\
    if ! is_running; then echo "csharp-plugin is not running"; cleanup_pid; return 0; fi\n\
    local pid=$(cat "$PID_FILE")\n\
    echo "Stopping csharp-plugin (PID: $pid)..."\n\
    kill -TERM "$pid" 2>/dev/null\n\
    local count=0\n\
    while [ $count -lt $STOP_TIMEOUT ]; do\n\
        if ! ps -p "$pid" > /dev/null 2>&1; then\n\
            echo "csharp-plugin stopped"; cleanup_pid; return 0\n\
        fi\n\
        sleep 1; count=$((count + 1))\n\
    done\n\
    kill -KILL "$pid" 2>/dev/null; sleep 1; cleanup_pid\n\
}\n\
case "${1:-}" in\n\
    start) start ;;\n\
    stop) stop ;;\n\
    status) is_running && echo "running" || echo "stopped" ;;\n\
    *) echo "Usage: $0 {start|stop|status}"; exit 1 ;;\n\
esac\n' > plugin/csharp/pluginctl.sh && chmod +x plugin/csharp/pluginctl.sh

# Set plugin type for Canopy to start
RUN printf '{"plugin":"csharp"}\n' > /root/.canopy/config.json

# Default: run canopy (start plugin separately with: dotnet plugin/CanopyPlugin.dll)
# Mount config at runtime: -v ~/.canopy:/root/.canopy
CMD ["./canopy", "start"]
