# Security Best Practices

This document outlines the security best practices implemented in the MCP container images.

## Container Security Hardening

### Non-Root User Execution
- **Node.js Container**: Runs as user `node` (UID 1000, GID 1000)
- **Python Container**: Runs as user `mcp` (UID 1000, GID 1000)
- Both containers never execute processes as root user

### Minimal Attack Surface
- **Multi-stage builds**: Only production dependencies are included in final images
- **Minimal base images**: Using `node:lts-alpine` and `python:3.12-slim`
- **Unnecessary packages removed**: Automated cleanup of build tools and temporary files
- **No exposed ports**: Containers only use stdio transport (no network exposure)

### File System Security
- **Proper permissions**: All files and directories have restrictive permissions (755 for directories, 644 for files)
- **Read-only system directories**: System binaries are protected from modification
- **Secure entrypoints**: Entrypoint scripts have secure permissions and ownership
- **Home directory isolation**: Each container has its own isolated home directory

### Signal Handling and Process Management
- **dumb-init**: Proper PID 1 process for signal forwarding and zombie reaping
- **exec usage**: Entrypoint scripts use `exec` to replace shell process
- **Signal forwarding**: SIGTERM/SIGINT properly forwarded to child processes

### Environment Security
- **No sensitive defaults**: No hardcoded credentials or sensitive information
- **Minimal environment**: Only necessary environment variables are set
- **Security labels**: Containers are labeled with security metadata

### Runtime Security
- **No new privileges**: Containers cannot escalate privileges
- **Resource isolation**: Proper UID/GID mapping for volume permissions
- **Unbuffered I/O**: Secure stdio transport for MCP protocol

## Security Testing

### Automated Security Checks
- **Vulnerability scanning**: Trivy scans for known vulnerabilities
- **Dockerfile linting**: Hadolint enforces security best practices
- **User verification**: Automated tests verify non-root execution
- **Permission testing**: File system permissions are validated

### Property-Based Security Tests
- **Property 12**: Docker Security Best Practices validation
- **Property 14**: Language-specific package manager security
- **Property 17**: stdio transport integrity and security

## Security Monitoring

### Build-Time Security
- **Base image updates**: Automated security updates for base images
- **Dependency scanning**: Package vulnerabilities are detected and reported
- **Build failure on critical issues**: High/critical vulnerabilities fail the build

### Runtime Security
- **No privileged execution**: Containers never require privileged mode
- **Volume mount security**: Proper permission handling for mounted volumes
- **Signal handling**: Secure process termination and cleanup

## Reporting Security Issues

If you discover a security vulnerability, please report it to the maintainers through:
- GitHub Security Advisories (preferred)
- Private issue reporting
- Direct contact with maintainers

Do not report security vulnerabilities through public issues.

## Security Compliance

These containers follow:
- **CIS Docker Benchmark** security recommendations
- **NIST Container Security** guidelines
- **OWASP Container Security** best practices
- **Docker Security Best Practices** from Docker Inc.

## Regular Security Updates

- **Weekly upstream checks**: Automated detection of new runtime versions
- **Security patch automation**: Critical security updates trigger rebuilds
- **Dependency updates**: Regular updates of MCP SDK and dependencies
- **Base image updates**: Tracking of upstream security patches