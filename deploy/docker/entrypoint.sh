#!/bin/sh
set -e

DATA_DIR="${STORE_PATH:-/data}"
mkdir -p "$DATA_DIR"
chown -R appuser:appuser "$DATA_DIR"

exec su-exec appuser /app/server "$@"
