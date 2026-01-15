# Multi-stage build for Python MCP server container
# Accept Python version as build argument
ARG PYTHON_VERSION=3.12.12

# Runtime stage
FROM python:${PYTHON_VERSION}-slim AS runtime

# Copy entrypoint script
COPY --chown=1000:1000 entrypoint.sh /usr/local/bin/entrypoint.sh

# Install runtime dependencies, create user, set up directories, and configure system
RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y --no-install-recommends curl git dumb-init && \
    apt-get autoremove -y && \
    apt-get autoclean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* && \
    mkdir -p /app /data /home/mcp && \
    groupadd -g 1000 mcp && \
    useradd -u 1000 -g 1000 -d /home/mcp -s /bin/bash mcp && \
    chown -R 1000:1000 /app /data /home/mcp && \
    chmod 755 /app /data /home/mcp && \
    pip install --no-cache-dir uv && \
    chmod 755 /usr/local/bin/entrypoint.sh && \
    chmod -R go-w /usr/local/bin/ /usr/bin/ /bin/ /sbin/ || true

# Set environment variables
ENV HOME="/home/mcp" \
    PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1

# Set working directory and switch to non-root user
WORKDIR /app
USER 1000:1000

# Add security labels and metadata
LABEL security.non-root="true" \
      security.uid="1000" \
      security.gid="1000" \
      security.no-new-privileges="true"

# Default entrypoint with dumb-init for proper signal handling
ENTRYPOINT ["/usr/bin/dumb-init", "--", "/usr/local/bin/entrypoint.sh"]
CMD ["python", "--version"]