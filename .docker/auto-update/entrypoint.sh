#!/bin/sh

# if BIN_PATH its defined we make a link to it from its volume to our system
if [ -z "${BIN_PATH}" ]; then
  echo "BIN_PATH not provided using /bin/cli as default"
  export BIN_PATH="/bin/cli"
else
  echo "using existing BIN_PATH $BIN_PATH"
fi

# resolve data dir: --data-dir flag, then DATA_DIR env, then default
DATA_DIR="${DATA_DIR:-${HOME:-/root}/.canopy}"
prev=""
for arg in "$@"; do
  case "$prev" in
    --data-dir) DATA_DIR="$arg" ;;
  esac
  case "$arg" in
    --data-dir=*) DATA_DIR="${arg#--data-dir=}" ;;
  esac
  prev="$arg"
done
mkdir -p "$DATA_DIR"
# normalize to an absolute path so relative values (e.g. ./data) don't create a
# dangling symlink at $BIN_PATH (symlink targets resolve relative to the link)
DATA_DIR=$(cd "$DATA_DIR" && pwd) || { echo "failed to resolve data dir: $DATA_DIR"; exit 1; }
echo "using data directory $DATA_DIR"

# Persisting current version
# TODO: remove this block as the auto-updater persists updates on its own
# Check if it exist
if [ -f "$DATA_DIR/cli" ]; then
  echo "Found existing persistent cli version"
else
  echo "Persisting build version for current cli"
  cp "$BIN_PATH" "$DATA_DIR/cli"
fi
ln -sf "$DATA_DIR/cli" "$BIN_PATH"

exec /app/canopy "$@"
