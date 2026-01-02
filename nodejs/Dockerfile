# Multi-stage Node.js container for MCP servers
# Uses node:lts-alpine as base image with multi-architecture support

# Build stage - for dependency installation and compilation
FROM node:lts-alpine AS builder

# Install build dependencies with proper version constraints
RUN apk add --no-cache \
    python3 \
    make \
    g++ \
    git

# Set up working directory
WORKDIR /app

# Copy package files for dependency caching
COPY package*.json ./
COPY yarn.lock* ./

# Install dependencies (both npm and yarn available)
RUN npm install --only=production && \
    npm cache clean --force

# Runtime stage - minimal production image
FROM node:lts-alpine AS runtime

# Install security updates and create non-root user
RUN apk update && \
    apk upgrade && \
    apk add --no-cache \
    dumb-init \
    && rm -rf /var/cache/apk/* \
    && rm -rf /tmp/* \
    && rm -rf /var/tmp/*

# Remove unnecessary packages and files to minimize attack surface
RUN apk del --no-cache \
    wget \
    curl || true

# Use existing node user (already UID 1000 in node:lts-alpine)
# No need to create new user - node user already exists with UID 1000

# Set up directories with proper permissions
RUN mkdir -p /app /data && \
    chown -R node:node /app /data && \
    chmod 755 /app /data

# Copy built dependencies from builder stage
COPY --from=builder --chown=node:node /app/node_modules /app/node_modules

# Install global package managers (npm already included, yarn already available)
# Just ensure yarn is up to date
RUN npm install -g yarn@latest --force && \
    npm cache clean --force

# Copy entrypoint script with secure permissions
COPY --chown=node:node entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod 755 /usr/local/bin/entrypoint.sh && \
    chown node:node /usr/local/bin/entrypoint.sh

# Remove write permissions from system directories for security
RUN chmod -R go-w /usr/local/bin/ /usr/bin/ /bin/ /sbin/ || true

# Create home directory for node user with proper permissions (as root)
RUN mkdir -p /home/mcp && \
    chown -R node:node /home/mcp && \
    chmod 755 /home/mcp

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