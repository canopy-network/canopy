#!/bin/bash
# pluginctl.sh - Control script for managing the kotlin-plugin
# Usage: ./pluginctl.sh {start|stop|status|restart|build}
# Configuration variables for paths and files
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
JAR_PATH="$SCRIPT_DIR/build/libs/canopy-plugin-kotlin-1.0.0-all.jar"
PID_FILE="/tmp/plugin/kotlin-plugin.pid"
LOG_FILE="/tmp/plugin/kotlin-plugin.log"
PLUGIN_DIR="/tmp/plugin"
# Timeout in seconds for graceful shutdown
STOP_TIMEOUT=10
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
    # Check if process exists and is the kotlin-plugin
    if ps -p "$pid" > /dev/null 2>&1; then
        # Verify it's actually our process
        if ps -p "$pid" -o cmd= | grep -q "canopy-plugin-kotlin"; then
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
# Build the kotlin-plugin fat JAR
build() {
    echo "Building kotlin-plugin..."
    cd "$SCRIPT_DIR"
    ./gradlew fatJar --no-daemon
    if [ $? -eq 0 ]; then
        echo "Build successful"
        return 0
    else
        echo "Build failed"
        return 1
    fi
}
# Start the kotlin-plugin
start() {
    # Check if already running
    if is_running; then
        echo "kotlin-plugin is already running (PID: $(cat "$PID_FILE"))"
        return 1
    fi
    # Clean up any stale PID file
    cleanup_pid
    # Check if JAR exists
    if [ ! -f "$JAR_PATH" ]; then
        echo "JAR not found at $JAR_PATH, building..."
        build
        if [ $? -ne 0 ]; then
            echo "Error: Build failed"
            return 1
        fi
    fi
    # Ensure plugin directory exists
    mkdir -p "$PLUGIN_DIR"
    # Start the plugin in background with nohup
    echo "Starting kotlin-plugin..."
    nohup java -jar "$JAR_PATH" > "$LOG_FILE" 2>&1 &
    local pid=$!
    # Save PID to file
    echo "$pid" > "$PID_FILE"
    # Give it a moment to start
    sleep 2
    # Verify it started successfully
    if is_running; then
        echo "kotlin-plugin started successfully (PID: $pid)"
        echo "Log file: $LOG_FILE"
        return 0
    else
        echo "Error: kotlin-plugin failed to start"
        echo "Check log file: $LOG_FILE"
        cleanup_pid
        return 1
    fi
}
# Stop the kotlin-plugin
stop() {
    # Check if running
    if ! is_running; then
        echo "kotlin-plugin is not running"
        cleanup_pid
        return 0
    fi
    # Read PID from file
    local pid=$(cat "$PID_FILE")
    echo "Stopping kotlin-plugin (PID: $pid)..."
    # Send SIGTERM for graceful shutdown
    kill -TERM "$pid" 2>/dev/null
    # Wait for process to exit with timeout
    local count=0
    while [ $count -lt $STOP_TIMEOUT ]; do
        if ! ps -p "$pid" > /dev/null 2>&1; then
            echo "kotlin-plugin stopped successfully"
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
        echo "kotlin-plugin stopped (forced)"
        cleanup_pid
        return 0
    else
        echo "Error: Failed to stop kotlin-plugin"
        return 1
    fi
}
# Check status of kotlin-plugin
status() {
    # Check if running
    if is_running; then
        local pid=$(cat "$PID_FILE")
        echo "kotlin-plugin is running (PID: $pid)"
        return 0
    else
        echo "kotlin-plugin is not running"
        cleanup_pid
        return 3
    fi
}
# Restart the kotlin-plugin
restart() {
    echo "Restarting kotlin-plugin..."
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
    build)
        build
        ;;
    *)
        echo "Usage: $0 {start|stop|status|restart|build}"
        exit 1
        ;;
esac
