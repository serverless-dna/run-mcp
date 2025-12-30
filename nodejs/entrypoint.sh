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
echo "[MCP-CONTAINER] Starting Node.js MCP server container" >&2
echo "[MCP-CONTAINER] User: $(id)" >&2
echo "[MCP-CONTAINER] Working directory: $(pwd)" >&2
echo "[MCP-CONTAINER] Node.js version: $(node --version)" >&2
echo "[MCP-CONTAINER] NPM version: $(npm --version)" >&2
echo "[MCP-CONTAINER] Yarn version: $(yarn --version)" >&2
echo "[MCP-CONTAINER] Command: $*" >&2
echo "[MCP-CONTAINER] Starting MCP server..." >&2

# Execute the provided command with all arguments
# Use exec to replace the shell process (PID 1) with the command
# This ensures proper signal handling and exit code propagation
# dumb-init (from Dockerfile) handles signal forwarding automatically
exec "$@"