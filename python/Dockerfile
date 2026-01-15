# Multi-stage build for Python MCP server container
# Accept Python version as build argument
ARG PYTHON_VERSION=3.12.12

# Stage 1: Builder stage for installing dependencies
FROM python:${PYTHON_VERSION}-slim AS builder

# Install system dependencies needed for building Python packages
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        build-essential \
        curl \
        git && \
    rm -rf /var/lib/apt/lists/*

# Install uv package manager
RUN pip install --no-cache-dir uv

# Create app directory and set up virtual environment
WORKDIR /app
RUN uv venv /app/venv

# Install dependencies using uv
RUN uv pip install --no-cache-dir --python /app/venv/bin/python mcp>=1.0.0 uvloop>=0.19.0 httpx>=0.25.0 pydantic>=2.0.0 typing-extensions>=4.8.0

# Stage 2: Runtime stage
FROM python:${PYTHON_VERSION}-slim AS runtime

# Install runtime dependencies and security updates
RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y --no-install-recommends \
        curl \
        git \
        dumb-init && \
    rm -rf /var/lib/apt/lists/* && \
    apt-get clean && \
    rm -rf /tmp/* && \
    rm -rf /var/tmp/*

# Remove unnecessary packages to minimize attack surface
RUN apt-get autoremove -y && \
    apt-get autoclean

# Create directories, set up user permissions, and install uv
RUN mkdir -p /app /data /home/mcp && \
    groupadd -g 1000 mcp && \
    useradd -u 1000 -g 1000 -d /home/mcp -s /bin/bash mcp && \
    chown -R 1000:1000 /app /data /home/mcp && \
    chmod 755 /app /data /home/mcp && \
    pip install --no-cache-dir uv

# Copy virtual environment from builder stage with secure permissions
COPY --from=builder --chown=1000:1000 /app/venv /app/venv

# Add virtual environment to PATH
ENV PATH="/app/venv/bin:$PATH"
ENV VIRTUAL_ENV="/app/venv"
ENV HOME="/home/mcp"

# Copy entrypoint script with secure permissions
COPY --chown=1000:1000 entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod 755 /usr/local/bin/entrypoint.sh && \
    chown 1000:1000 /usr/local/bin/entrypoint.sh

# Remove write permissions from system directories for security
RUN chmod -R go-w /usr/local/bin/ /usr/bin/ /bin/ /sbin/ || true

# Set working directory
WORKDIR /app

# Switch to non-root user
USER 1000:1000

# Set environment variables for unbuffered output
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1

# Add security labels and metadata
LABEL security.non-root="true" \
      security.uid="1000" \
      security.gid="1000" \
      security.no-new-privileges="true"

# Default entrypoint with dumb-init for proper signal handling
ENTRYPOINT ["/usr/bin/dumb-init", "--", "/usr/local/bin/entrypoint.sh"]
CMD ["python", "--version"]