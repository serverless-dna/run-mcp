#!/bin/sh
# Node.js MCP Server Entrypoint
# Simple passthrough entrypoint for stdio MCP servers
# Ensures unbuffered I/O and proper signal forwarding

set -e

# Ensure unbuffered stdio for MCP protocol communication
# This is critical for JSON-RPC over stdio transport
export NODE_OPTIONS="${NODE_OPTIONS} --no-warnings"
export FORCE_COLOR=0

# Set up signal forwarding for graceful shutdown
# Forward SIGTERM and SIGINT to child process
trap 'kill -TERM $PID' TERM INT

# Log startup information for debugging
echo "Starting Node.js MCP server container..." >&2
echo "User: $(id)" >&2
echo "Working directory: $(pwd)" >&2
echo "Node.js version: $(node --version)" >&2
echo "NPM version: $(npm --version)" >&2
echo "Yarn version: $(yarn --version)" >&2
echo "Command: $*" >&2
echo "---" >&2

# Execute the provided command with all arguments
# Use exec to replace the shell process (PID 1) with the command
# This ensures proper signal handling and exit code propagation
exec "$@"