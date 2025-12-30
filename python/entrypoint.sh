#!/bin/sh
# Python MCP Server Entrypoint
# Standardized passthrough entrypoint for stdio MCP servers
# Ensures unbuffered I/O and proper signal forwarding

set -e

# Set unbuffered output for Python (critical for MCP protocol)
export PYTHONUNBUFFERED=1
export PYTHONDONTWRITEBYTECODE=1

# Log startup information for debugging (standardized format)
echo "[MCP-CONTAINER] Starting Python MCP server container" >&2
echo "[MCP-CONTAINER] User: $(id)" >&2
echo "[MCP-CONTAINER] Working directory: $(pwd)" >&2
echo "[MCP-CONTAINER] Python version: $(python --version)" >&2
echo "[MCP-CONTAINER] uv version: $(uv --version)" >&2
echo "[MCP-CONTAINER] Virtual environment: $VIRTUAL_ENV" >&2
echo "[MCP-CONTAINER] Command: $*" >&2
echo "[MCP-CONTAINER] Starting MCP server..." >&2

# Execute the provided command with all arguments
# Use exec to replace the shell process (PID 1) with the command
# This ensures proper signal handling and exit code propagation
# Signal forwarding is handled by the container runtime
exec "$@"