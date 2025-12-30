#!/bin/sh
# Python MCP Server Entrypoint
# Standardized passthrough entrypoint for stdio MCP servers
# Ensures unbuffered I/O and proper signal forwarding

set -e

# Set unbuffered output for Python (critical for MCP protocol)
export PYTHONUNBUFFERED=1
export PYTHONDONTWRITEBYTECODE=1

# Log startup information for debugging (standardized format)
echo "[MCP-CONTAINER] ========================================" >&2
echo "[MCP-CONTAINER] Starting Python MCP server container" >&2
echo "[MCP-CONTAINER] ========================================" >&2
echo "[MCP-CONTAINER] Timestamp: $(date -u '+%Y-%m-%d %H:%M:%S UTC')" >&2
echo "[MCP-CONTAINER] Container: Python MCP Server" >&2
echo "[MCP-CONTAINER] User: $(id)" >&2
echo "[MCP-CONTAINER] Working directory: $(pwd)" >&2
echo "[MCP-CONTAINER] Environment:" >&2
echo "[MCP-CONTAINER]   PYTHONUNBUFFERED: ${PYTHONUNBUFFERED}" >&2
echo "[MCP-CONTAINER]   PYTHONDONTWRITEBYTECODE: ${PYTHONDONTWRITEBYTECODE}" >&2
echo "[MCP-CONTAINER]   VIRTUAL_ENV: ${VIRTUAL_ENV:-'(not set)'}" >&2
echo "[MCP-CONTAINER]   PATH: ${PATH}" >&2
echo "[MCP-CONTAINER] Runtime versions:" >&2
echo "[MCP-CONTAINER]   Python: $(python --version)" >&2
echo "[MCP-CONTAINER]   uv: $(uv --version)" >&2
echo "[MCP-CONTAINER] Volume mounts:" >&2
echo "[MCP-CONTAINER]   /data: $(ls -la /data 2>/dev/null | wc -l) items" >&2
echo "[MCP-CONTAINER]   /app: $(ls -la /app 2>/dev/null | wc -l) items" >&2
echo "[MCP-CONTAINER] Virtual environment status:" >&2
if [ -n "${VIRTUAL_ENV}" ]; then
    echo "[MCP-CONTAINER]   Active virtual environment: ${VIRTUAL_ENV}" >&2
    echo "[MCP-CONTAINER]   Python executable: $(which python)" >&2
else
    echo "[MCP-CONTAINER]   No virtual environment active" >&2
fi
echo "[MCP-CONTAINER] Command to execute: $*" >&2
echo "[MCP-CONTAINER] ========================================" >&2
echo "[MCP-CONTAINER] Starting MCP server process..." >&2

# Execute the provided command with all arguments
# Use exec to replace the shell process (PID 1) with the command
# This ensures proper signal handling and exit code propagation
# Signal forwarding is handled by the container runtime
exec "$@"