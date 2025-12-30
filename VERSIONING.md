# Container Image Tagging Strategy for MCP Containers

## Overview

MCP container images are tagged by **language runtime version**, following the runtime's own semver. This lets you choose your stability/freshness tradeoff:

- `node22` — Always latest Node 22.x.x (patches automatically)
- `node22.11` — Latest Node 22.11.x (patch updates only)  
- `node22.11.0` — Exact Node 22.11.0 (never changes)

## Tag Types

### LTS Tags (Follow Current LTS)

| Tag | Description |
|-----|-------------|
| `latest` | Current LTS runtime |
| `node-lts` | Current Node.js LTS major version |
| `python-lts` | Current Python LTS major version |

These move when LTS designation changes (e.g., Node 24 becomes LTS).

### Major Version Tags

Tracks latest minor/patch within a major version.

| Tag | Tracks | Updates When |
|-----|--------|--------------|
| `node22` | 22.x.x | Any Node 22 release |
| `node20` | 20.x.x | Any Node 20 release |
| `python3.12` | 3.12.x | Any Python 3.12 release |
| `python3.11` | 3.11.x | Any Python 3.11 release |

### Minor Version Tags

Tracks latest patch within a minor version.

| Tag | Tracks | Updates When |
|-----|--------|--------------|
| `node22.11` | 22.11.x | Node 22.11.x patch release |
| `node22.12` | 22.12.x | Node 22.12.x patch release |

**Note:** Python versioning uses `3.12.x` format where `x` is the patch version. There are no "minor version tags" for Python — use major version tags (`python3.12`) or exact version tags (`python3.12.8`).

### Exact Version Tags (Immutable)

Never change after creation.

| Tag | Example | Description |
|-----|---------|-------------|
| `node{major}.{minor}.{patch}` | `node22.11.0` | Exact Node.js version |
| `python{major}.{minor}.{patch}` | `python3.12.8` | Exact Python version |
| `{version}-{YYYYMMDD}` | `node22.11.0-20250115` | Exact version + container build date |

### Development Tags

| Tag | Description |
|-----|-------------|
| `main` | Latest build from main branch |
| `pr-{number}` | Pull request preview builds |

## Choosing a Tag

| Need | Tag | Example | What Moves |
|------|-----|---------|------------|
| Always current LTS | `node-lts` | `node-lts` | Major version changes |
| Latest patches for a major | `node22` | `node22` | Minor + patch |
| Latest patches for a minor | `node22.11` | `node22.11` | Patch only |
| Exact runtime version | `node22.11.0` | `node22.11.0` | Nothing |
| Exact container build | `node22.11.0-20250115` | `node22.11.0-20250115` | Nothing |

**Recommendation:** Use major version tags (`node22`) for most use cases. Pin to exact versions (`node22.11.0`) only when reproducibility is critical.

## Tag Lifecycle Example

When Node.js 22.12.0 releases:
```
┌─────────────────────────────────────────────────────────┐
│ node22      │ 22.11.0 ──────────────> 22.12.0 │ MOVES   │
│ node22.12   │ (new)                   22.12.0 │ CREATED │
│ node22.12.0 │ (new)                   22.12.0 │ CREATED │
│ node22.11   │ 22.11.0                 22.11.0 │ UNCHANGED │
│ node22.11.0 │ 22.11.0                 22.11.0 │ UNCHANGED │
└─────────────────────────────────────────────────────────┘
```

When Node.js 22.11.1 releases (patch):
```
┌─────────────────────────────────────────────────────────┐
│ node22      │ 22.11.0 ──────────────> 22.11.1 │ MOVES   │
│ node22.11   │ 22.11.0 ──────────────> 22.11.1 │ MOVES   │
│ node22.11.0 │ 22.11.0                 22.11.0 │ UNCHANGED │
│ node22.11.1 │ (new)                   22.11.1 │ CREATED │
└─────────────────────────────────────────────────────────┘
```

## How Tags Are Updated

### Runtime Tags (node22, python3.12)
Updated automatically when:
- Base image receives security patches
- Dockerfile improvements are made
- Dependencies are updated

**Guarantee:** Same runtime version, latest security patches.

### Pinned Tags (node22-20250115)
Never updated. Created on each successful build.

**Guarantee:** Byte-for-byte identical pulls.

### latest Tag
Points to the current recommended runtime:
- `mcp-nodejs:latest` → `mcp-nodejs:node22`
- `mcp-python:latest` → `mcp-python:python3.12`

Updated when default runtime changes (major event, announced in advance).

## Choosing a Tag

| Scenario | Recommended Tag |
|----------|-----------------|
| Local development | `latest` or runtime tag |
| CI/CD pipelines | Runtime tag (`node22`) |
| Production with reproducibility needs | Pinned tag (`node22-20250115`) |
| Testing new features | `main` or `pr-{n}` |

## Listing Available Tags

To see all available tags for an image:

```bash
# Using GitHub CLI
gh api /users/{owner}/packages/container/mcp-nodejs/versions \
  --jq '.[].metadata.container.tags[]'

# Using Docker (after pulling once)
docker image ls ghcr.io/{owner}/mcp-nodejs --format "{{.Tag}}"
```

Or browse: https://github.com/{owner}/mcp-containers/pkgs/container/mcp-nodejs

## Version Support Limits

We actively build and maintain a limited set of versions to manage storage and build resources.

### Build Limits

| Dimension | Limit | Notes |
|-----------|-------|-------|
| Major versions per runtime | 2 | Current LTS + previous LTS |
| Minor versions per major | 3 | Latest 3 minor releases |
| Patch versions per minor | 5 | Latest 5 patch releases |
| Date-stamped tags per version | 10 | Latest 10 container builds |

### Currently Supported Versions

#### Node.js
| Major | Status | Minor Versions |
|-------|--------|----------------|
| 22 | Active LTS | 22.11, 22.10, 22.9 |
| 20 | Maintenance LTS | 20.18, 20.17, 20.16 |

#### Python
| Major | Status | Minor Versions |
|-------|--------|----------------|
| 3.12 | Active | 3.12.8, 3.12.7, 3.12.6 |
| 3.11 | Security | 3.11.11, 3.11.10, 3.11.9 |

### What Happens When Limits Are Exceeded

**New minor version released (e.g., Node 22.12.0):**
- `node22.12` created
- `node22.9` dropped (oldest of 3)
- All `node22.9.x` tags removed

**New major LTS released (e.g., Node 24 becomes LTS):**
- `node24` created
- `node20` dropped (oldest of 2)
- All `node20.x.x` tags removed after 90-day deprecation notice

**New patch released (e.g., Node 22.11.1):**
- `node22.11.1` created
- If >5 patches exist for 22.11, oldest is removed

### Deprecation Process

1. **Announcement** — 90 days before removal for major versions, 30 days for minor
2. **Warning tags** — Deprecated images tagged with `-deprecated` suffix
3. **Removal** — Tags deleted from registry
4. **Documentation** — Migration guide published

### Requesting Extended Support

If you need a version outside these limits retained:
1. Open an issue explaining the use case
2. We may retain specific versions on a case-by-case basis
3. No guarantees on security patches for extended versions

### Storage Budget

Target: < 50GB total registry storage

| Component | Estimated Size |
|-----------|---------------|
| Per image (compressed) | ~80MB Node, ~120MB Python |
| Total tags | ~680 |
| Multi-arch overhead | 2× (AMD64 + ARM64) |
| **Total estimate** | ~40GB |

### Summary Table

| What | Limit | When Exceeded |
|------|-------|---------------|
| Major versions | 2 | Oldest dropped (90-day notice) |
| Minors per major | 3 | Oldest dropped (30-day notice) |
| Patches per minor | 5 | Oldest dropped (immediate) |
| Date tags per version | 10 | Oldest dropped (immediate) |

This keeps storage bounded while giving users reasonable rollback options.

## Tag Retention

Based on the version support limits above:

| Tag Type | Retention |
|----------|-----------|
| LTS tags | Always available |
| Major version tags | 2 per runtime (current + previous LTS) |
| Minor version tags | 3 per major version |
| Exact version tags | 5 per minor version |
| Date-stamped tags | 10 per exact version |
| Development tags | 30 days |

**Total tag count per runtime:** ~340 tags
**Total for both runtimes:** ~680 tags

## Runtime Support Lifecycle

| Runtime | Tag | Status | Support Until |
|---------|-----|--------|---------------|
| Node.js 22 | `node22` | Active LTS | April 2027 |
| Node.js 20 | `node20` | Maintenance | April 2026 |
| Python 3.12 | `python3.12` | Active | October 2028 |
| Python 3.11 | `python3.11` | Security | October 2027 |

Tags for EOL runtimes are removed 90 days after the runtime's end-of-life date.

## Breaking Changes

Breaking changes (new default runtime, deprecated runtime removal) are:
- Announced 90 days in advance
- Documented in release notes
- Never affect pinned tags

## Multi-Architecture

All tags include both AMD64 and ARM64 in a single manifest. Docker automatically pulls the correct architecture.

## Claude Desktop Examples

### Major Version (Recommended)
```json
{
  "mcpServers": {
    "filesystem": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "/Users/me/Documents:/data",
        "ghcr.io/owner/mcp-nodejs:node22",
        "npx", "@modelcontextprotocol/server-filesystem", "/data"
      ]
    }
  }
}
```

### Exact Version (Production)
```json
{
  "mcpServers": {
    "sqlite": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "/Users/me/data:/data",
        "ghcr.io/owner/mcp-python:python3.12.8",
        "uvx", "mcp-server-sqlite", "--db-path", "/data/mydb.sqlite"
      ]
    }
  }
}
```

## Summary

| Level | Tag Example | Mutability | Use Case |
|-------|-------------|------------|----------|
| LTS | `node-lts` | Moves with LTS | Always current |
| Major | `node22` | Minor+patch updates | Stable major, auto-patched |
| Minor | `node22.11` | Patch updates | Tighter control |
| Exact | `node22.11.0` | Immutable | Reproducibility |
| Build | `node22.11.0-20250115` | Immutable | Debug container issues |

This gives users full semver granularity matching Node.js/Python releases while maintaining clear stability guarantees at each level.