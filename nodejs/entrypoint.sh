#!/bin/sh
# Node.js MCP Server Entrypoint
# Standardized passthrough entrypoint for stdio MCP servers
# Ensures unbuffered I/O and proper signal forwarding

set -e

# Ensure unbuffered stdio for MCP protocol communication
# This is critical for JSON-RPC over stdio transport
export NODE_OPTIONS="${NODE_OPTIONS} --no-warnings"
export FORCE_COLOR=0

# Log startup information for debugging (standardized format)
echo "[MCP-CONTAINER] ========================================" >&2
echo "[MCP-CONTAINER] Starting Node.js MCP server container" >&2
echo "[MCP-CONTAINER] ========================================" >&2
echo "[MCP-CONTAINER] Timestamp: $(date -u '+%Y-%m-%d %H:%M:%S UTC')" >&2
echo "[MCP-CONTAINER] Container: Node.js MCP Server" >&2
echo "[MCP-CONTAINER] User: $(id)" >&2
echo "[MCP-CONTAINER] Working directory: $(pwd)" >&2
echo "[MCP-CONTAINER] Environment:" >&2
echo "[MCP-CONTAINER]   NODE_OPTIONS: ${NODE_OPTIONS}" >&2
echo "[MCP-CONTAINER]   FORCE_COLOR: ${FORCE_COLOR}" >&2
echo "[MCP-CONTAINER]   PATH: ${PATH}" >&2
echo "[MCP-CONTAINER] Runtime versions:" >&2
echo "[MCP-CONTAINER]   Node.js: $(node --version)" >&2
echo "[MCP-CONTAINER]   NPM: $(npm --version)" >&2
echo "[MCP-CONTAINER]   Yarn: $(yarn --version)" >&2
echo "[MCP-CONTAINER] Volume mounts:" >&2
echo "[MCP-CONTAINER]   /data: $(ls -la /data 2>/dev/null | wc -l) items" >&2
echo "[MCP-CONTAINER]   /app: $(ls -la /app 2>/dev/null | wc -l) items" >&2
echo "[MCP-CONTAINER] Command to execute: $*" >&2
echo "[MCP-CONTAINER] ========================================" >&2
echo "[MCP-CONTAINER] Starting MCP server process..." >&2

# Execute the provided command with all arguments
# Use exec to replace the shell process (PID 1) with the command
# This ensures proper signal handling and exit code propagation
# dumb-init (from Dockerfile) handles signal forwarding automatically
exec "$@"