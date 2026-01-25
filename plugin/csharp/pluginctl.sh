#!/bin/bash
# pluginctl.sh - Control script for managing the csharp-plugin binary
# Usage: ./pluginctl.sh {start|stop|status|restart}
# Configuration variables for paths and files
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_PATH="$SCRIPT_DIR/bin/CanopyPlugin.dll"
PID_FILE="/tmp/plugin/csharp-plugin.pid"
LOG_FILE="/tmp/plugin/csharp-plugin.log"
PLUGIN_DIR="/tmp/plugin"
# Timeout in seconds for graceful shutdown
STOP_TIMEOUT=10

# Detect system architecture
get_arch() {
    local arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)
            echo "x64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            echo "x64"  # Default to x64
            ;;
    esac
}

# Extract tarball if binary doesn't exist
extract_if_needed() {
    # If DLL already exists, nothing to do
    if [ -f "$BINARY_PATH" ]; then
        return 0
    fi
    
    # Check for architecture-specific tarball
    local arch=$(get_arch)
    local tarball="$SCRIPT_DIR/csharp-plugin-linux-${arch}.tar.gz"
    
    if [ -f "$tarball" ]; then
        echo "Extracting $tarball..."
        mkdir -p "$SCRIPT_DIR/bin"
        tar -xzf "$tarball" -C "$SCRIPT_DIR/bin"
        if [ $? -eq 0 ] && [ -f "$BINARY_PATH" ]; then
            echo "Extraction complete"
            return 0
        else
            echo "Error: Failed to extract from $tarball"
            return 1
        fi
    fi
    
    return 1
}

# Check if the process is running based on PID file
is_running() {
    # Return 1 if PID file doesn't exist
    if [ ! -f "$PID_FILE" ]; then
        return 1
    fi
    # Read PID from file
    local pid=$(cat "$PID_FILE" 2>/dev/null)
    # Return 1 if PID is empty or not a number
    if [ -z "$pid" ] || ! [[ "$pid" =~ ^[0-9]+$ ]]; then
        return 1
    fi
    # Check if process exists and is the CanopyPlugin binary
    if ps -p "$pid" > /dev/null 2>&1; then
        # Verify it's actually our binary (running via dotnet)
        if ps -p "$pid" -o cmd= | grep -q "CanopyPlugin.dll"; then
            return 0
        fi
    fi
    # Process not running
    return 1
}
# Clean up stale PID file
cleanup_pid() {
    # Remove PID file if it exists
    if [ -f "$PID_FILE" ]; then
        rm -f "$PID_FILE"
    fi
}
# Start the csharp-plugin binary
start() {
    # Check if already running
    if is_running; then
        echo "csharp-plugin is already running (PID: $(cat "$PID_FILE"))"
        return 1
    fi
    # Clean up any stale PID file
    cleanup_pid
    # Try to extract from tarball if binary doesn't exist
    extract_if_needed
    # Check if binary exists
    if [ ! -f "$BINARY_PATH" ]; then
        echo "Error: Binary not found at $BINARY_PATH"
        echo "Run 'make build' or download csharp-plugin-linux-$(get_arch).tar.gz"
        return 1
    fi
    # Ensure plugin directory exists
    mkdir -p "$PLUGIN_DIR"
    # Start the binary in background with nohup (use dotnet to run the DLL)
    echo "Starting csharp-plugin..."
    nohup dotnet "$BINARY_PATH" > "$LOG_FILE" 2>&1 &
    local pid=$!
    # Save PID to file
    echo "$pid" > "$PID_FILE"
    # Give it a moment to start
    sleep 1
    # Verify it started successfully
    if is_running; then
        echo "csharp-plugin started successfully (PID: $pid)"
        echo "Log file: $LOG_FILE"
        return 0
    else
        echo "Error: csharp-plugin failed to start"
        cleanup_pid
        return 1
    fi
}
# Stop the csharp-plugin binary
stop() {
    # Check if running
    if ! is_running; then
        echo "csharp-plugin is not running"
        cleanup_pid
        return 0
    fi
    # Read PID from file
    local pid=$(cat "$PID_FILE")
    echo "Stopping csharp-plugin (PID: $pid)..."
    # Send SIGTERM for graceful shutdown
    kill -TERM "$pid" 2>/dev/null
    # Wait for process to exit with timeout
    local count=0
    while [ $count -lt $STOP_TIMEOUT ]; do
        if ! ps -p "$pid" > /dev/null 2>&1; then
            echo "csharp-plugin stopped successfully"
            cleanup_pid
            return 0
        fi
        sleep 1
        count=$((count + 1))
    done
    # If still running after timeout, force kill
    echo "Process did not stop gracefully, forcing shutdown..."
    kill -KILL "$pid" 2>/dev/null
    sleep 1
    # Verify it's stopped
    if ! ps -p "$pid" > /dev/null 2>&1; then
        echo "csharp-plugin stopped (forced)"
        cleanup_pid
        return 0
    else
        echo "Error: Failed to stop csharp-plugin"
        return 1
    fi
}
# Check status of csharp-plugin binary
status() {
    # Check if running
    if is_running; then
        local pid=$(cat "$PID_FILE")
        echo "csharp-plugin is running (PID: $pid)"
        return 0
    else
        echo "csharp-plugin is not running"
        cleanup_pid
        return 3
    fi
}
# Restart the csharp-plugin binary
restart() {
    echo "Restarting csharp-plugin..."
    # Stop the process
    stop
    # Brief pause between stop and start
    sleep 2
    # Start the process
    start
}
# Main command routing
case "${1:-}" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    status)
        status
        ;;
    restart)
        restart
        ;;
    *)
        echo "Usage: $0 {start|stop|status|restart}"
        exit 1
        ;;
esac
