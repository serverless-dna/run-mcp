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
    && rm -rf /var/cache/apk/*

# Use existing node user (already UID 1000 in node:lts-alpine)
# No need to create new user - node user already exists with UID 1000

# Set up directories with proper permissions
RUN mkdir -p /app /data && \
    chown -R node:node /app /data

# Copy built dependencies from builder stage
COPY --from=builder --chown=node:node /app/node_modules /app/node_modules

# Install global package managers (npm already included, yarn already available)
# Just ensure yarn is up to date
RUN npm install -g yarn@latest --force && \
    npm cache clean --force

# Copy entrypoint script
COPY --chown=node:node entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Switch to non-root user
USER node

# Set working directory
WORKDIR /app

# Set environment variables for unbuffered I/O (required for MCP stdio)
ENV NODE_ENV=production \
    NPM_CONFIG_UPDATE_NOTIFIER=false \
    NPM_CONFIG_FUND=false \
    FORCE_COLOR=0 \
    NODE_OPTIONS="--unhandled-rejections=strict"

# Expose no ports (stdio transport only)
# Volume mount points: /data for user data, /app for MCP server code

# Use dumb-init for proper signal handling and entrypoint for MCP protocol
ENTRYPOINT ["/usr/bin/dumb-init", "--", "/usr/local/bin/entrypoint.sh"]

# Default command (can be overridden)
CMD ["node", "--version"]