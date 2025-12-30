#!/bin/sh
# Python MCP Server Entrypoint
# Simple passthrough entrypoint for stdio MCP servers
# Ensures unbuffered I/O and proper signal forwarding

set -e

# Set unbuffered output for Python (critical for MCP protocol)
export PYTHONUNBUFFERED=1
export PYTHONDONTWRITEBYTECODE=1

# Set up signal forwarding for graceful shutdown
# Forward SIGTERM and SIGINT to child process
trap 'kill -TERM $PID' TERM INT

# Log startup information for debugging
echo "Starting Python MCP server container..." >&2
echo "User: $(id)" >&2
echo "Working directory: $(pwd)" >&2
echo "Python version: $(python --version)" >&2
echo "uv version: $(uv --version)" >&2
echo "Virtual environment: $VIRTUAL_ENV" >&2
echo "Command: $*" >&2
echo "---" >&2

# Execute the provided command with all arguments
# Use exec to replace the shell process (PID 1) with the command
# This ensures proper signal handling and exit code propagation
exec "$@"