# Multi-stage Node.js container for MCP servers
# Accept Node.js version as build argument
ARG NODE_VERSION=22.11.0

# Runtime stage - minimal production image
FROM node:${NODE_VERSION}-alpine AS runtime

# Copy entrypoint script
COPY --chown=node:node entrypoint.sh /usr/local/bin/entrypoint.sh

# Install security updates, set up directories, and configure system
RUN apk update && \
    apk upgrade && \
    apk add --no-cache dumb-init && \
    apk del --no-cache wget curl || true && \
    rm -rf /var/cache/apk/* /tmp/* /var/tmp/* && \
    mkdir -p /app /data /home/mcp && \
    chown -R node:node /app /data /home/mcp && \
    chmod 755 /app /data /home/mcp && \
    npm install -g yarn@latest --force && \
    npm cache clean --force && \
    chmod 755 /usr/local/bin/entrypoint.sh && \
    chmod -R go-w /usr/local/bin/ /usr/bin/ /bin/ /sbin/ || true

# Switch to non-root user
USER node

# Set working directory
WORKDIR /app

# Set environment variables for unbuffered I/O (required for MCP stdio)
ENV NODE_ENV=production \
    NPM_CONFIG_UPDATE_NOTIFIER=false \
    NPM_CONFIG_FUND=false \
    FORCE_COLOR=0 \
    NODE_OPTIONS="--unhandled-rejections=strict" \
    HOME=/home/mcp

# Expose no ports (stdio transport only)
# Volume mount points: /data for user data, /app for MCP server code

# Add security labels and metadata
LABEL security.non-root="true" \
      security.uid="1000" \
      security.gid="1000" \
      security.no-new-privileges="true"

# Use dumb-init for proper signal handling and entrypoint for MCP protocol
ENTRYPOINT ["/usr/bin/dumb-init", "--", "/usr/local/bin/entrypoint.sh"]

# Default command (can be overridden)
CMD ["node", "--version"]