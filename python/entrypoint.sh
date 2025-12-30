#!/bin/sh
# Simple passthrough entrypoint for Python MCP servers
# Ensures unbuffered stdio for MCP protocol communication

# Set unbuffered output for Python
export PYTHONUNBUFFERED=1
export PYTHONDONTWRITEBYTECODE=1

# Execute the provided command directly
exec "$@"