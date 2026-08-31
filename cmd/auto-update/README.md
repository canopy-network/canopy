# Canopy Auto-Update Package

This package implements an automatic update mechanism for the Canopy CLI application. It periodically checks for new releases and automatically updates the binary when available, ensuring users always have the latest version of the software.

## Overview

The auto-update package provides a robust mechanism to:
- Check for new releases on GitHub
- Download and install updates automatically
- Manage the running CLI process
- Handle graceful updates without disrupting the user experience

## Architecture

### Flow of Operation

1. **Initialization and Configuration**
   - The system starts by checking if auto-update is enabled in the configuration
   - Sets up necessary data directories and binary paths
   - Initializes thread-safe state management mechanisms
   - Creates communication channels for process coordination

2. **Version Check and Update Detection**
   - Every 30 minutes, the system queries GitHub's API for the latest release
   - Compares the current version with the latest release
   - If a new version is found, sets a flag to prevent concurrent updates
   - The system identifies the appropriate platform-specific binary for the current OS/architecture

3. **Update Process**
   - When a new version is detected, the system:
     1. Downloads the new binary while maintaining exclusive access
     2. Replaces the existing binary with proper permissions
     3. Adds a random delay (1-30 minutes) to prevent update storms
     4. Sets an update-in-progress flag to manage the update state

4. **Process Management**
   - The system maintains two main goroutines:
     1. Process Manager: Handles the CLI process lifecycle
     2. Update Checker: Periodically checks for new versions
   - When an update is ready:
     1. Sends termination signal to the current process
     2. Waits for process termination
     3. Starts the new version
     4. Maintains process state and handles errors

5. **Thread Safety and Coordination**
   - Uses atomic flags for state management:
     - Update-in-progress flag: Prevents process exit during updates
     - Update-detection flag: Prevents concurrent update handling
   - Implements mutex for binary file access
   - Uses buffered channels for process coordination:
     - Start signal channel: Signals process start
     - End signal channel: Signals process end

## How It Works

1. **Initialization**
   - Creates necessary data directories
   - Loads configuration
   - Sets up binary path

2. **Update Check Loop**
   - Runs every 30 minutes
   - Fetches latest release from GitHub
   - Compares versions
   - Triggers update if needed

3. **Update Process**
   - Downloads new binary
   - Adds random delay (1-30 minutes)
   - Gracefully terminates current process
   - Starts new version
   - Maintains process state

4. **Error Handling**
   - Manages download failures
   - Handles process termination errors
   - Recovers from failed updates
   - Logs error conditions

## Configuration

The auto-update mechanism can be enabled/disabled through the Canopy configuration:

```json
{
    "autoUpdate": true
}
```

## Environment Variables

- `BIN_PATH`: Specifies the path to the CLI binary (defaults to "./cli")

### Binary persistence

`BIN_PATH` is only ever read, never written to. It points at the binary shipped with the
build, which in a container lives on the ephemeral filesystem and is discarded whenever
the container is recreated.

Updates are downloaded to `<data-dir>/canopy_updated` instead, since the data directory
is the volume that persists across restarts. On every start, and on every restart after
an update, the coordinator runs `<data-dir>/canopy_updated` when it exists and is
executable, and falls back to `BIN_PATH` when it does not. Removing that file rolls the
node back to the version shipped with the image.

### Plugin persistence

For the same reason, plugin artifacts are stored under the data directory rather than in
the image. The coordinator downloads (and `pluginctl.sh` extracts and runs) the plugin
from `<data-dir>/plugin/<lang>`, which is exported to the launcher as `CANOPY_PLUGIN_HOME`.
Only `pluginctl.sh` ships in the image; the downloaded binary/tarball live on the
persistent volume, so a restart reuses the already-downloaded plugin instead of
re-fetching it from GitHub. Removing that directory forces a fresh download on next start.

## Dependencies

- Go standard library
- GitHub API for release information
- Canopy core packages

## Usage

For deployment examples, refer to the `Dockerfile` in the root of the repository. The Dockerfile demonstrates:
- How to build the auto-update enabled binary
- Proper environment setup
- Required permissions
- Volume mounting for persistence
- Process management in a containerized environment

Example Dockerfile usage:
```bash
# Build the image
docker build --build-arg BIN_PATH=./cli -t canopy .

# Run the container
docker run -it --env-file=.env -p 50000:50000 -p 50001:50001 -p 50002:50002 -p 50003:50003 -p 9001:9001 --name canopy canopy
```
