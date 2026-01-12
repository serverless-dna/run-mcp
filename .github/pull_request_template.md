# Pull Request

## Description

<!-- Provide a brief description of the changes in this PR -->

## Type of Change

<!-- Mark the relevant option with an "x" -->

- [ ] 🐛 Bug fix (non-breaking change which fixes an issue)
- [ ] ✨ New feature (non-breaking change which adds functionality)
- [ ] 💥 Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] 📚 Documentation update
- [ ] 🔧 Refactoring (no functional changes)
- [ ] ⚡ Performance improvement
- [ ] 🔒 Security enhancement
- [ ] 🧪 Test improvements
- [ ] 🏗️ Build/CI changes

## Related Issues

<!-- Link to any related issues -->
Fixes #(issue number)
Relates to #(issue number)

## Changes Made

<!-- Describe the specific changes made in this PR -->

- 
- 
- 

## Testing

<!-- Describe the tests you ran and how to reproduce them -->

### Test Commands Run

<!-- Check all that apply and confirm tests pass -->

- [ ] `make test` - All container tests pass
- [ ] `make test-run-mcp` - All Go tests pass (required for Go code changes)
- [ ] Manual testing performed (describe below)

### Manual Testing Details

<!-- If you performed manual testing, describe what you tested -->

```bash
# Example commands you ran to test
run-mcp --version
# Add your test commands here
```

### Test Configuration Used

<!-- If relevant, share the MCP configuration you used for testing (remove sensitive data) -->

```json
{
  "mcpServers": {
    "test": {
      "command": "run-mcp",
      "args": ["..."],
      "env": {
        "MCP_DATA_DIR": "..."
      }
    }
  }
}
```

## Security Considerations

<!-- Address any security implications of your changes -->

- [ ] No new security risks introduced
- [ ] Security implications have been considered and documented
- [ ] Changes improve security posture
- [ ] N/A - No security implications

### Security Details

<!-- If there are security implications, describe them -->

## Breaking Changes

<!-- If this is a breaking change, describe the impact and migration path -->

- [ ] No breaking changes
- [ ] Breaking changes documented below

### Migration Guide

<!-- If breaking changes exist, provide migration instructions -->

## Documentation

<!-- Ensure documentation is updated -->

- [ ] Code is self-documenting with clear comments
- [ ] README.md updated (if needed)
- [ ] Documentation in `docs/` updated (if needed)
- [ ] Examples updated (if needed)
- [ ] CHANGELOG.md updated

## Checklist

<!-- Ensure all items are completed before requesting review -->

### Code Quality

- [ ] Code follows the project's style guidelines
- [ ] Self-review of code completed
- [ ] Code is properly commented, particularly in hard-to-understand areas
- [ ] No debugging code or console logs left in

### Testing & CI

- [ ] Tests added for new functionality
- [ ] All existing tests still pass
- [ ] Make targets updated if new tests added
- [ ] CI/CD pipeline considerations addressed

### Dependencies

- [ ] No new dependencies added, or new dependencies are justified
- [ ] `go.mod` and `go.sum` updated appropriately (if Go changes)
- [ ] Container image dependencies updated appropriately (if container changes)

### Compatibility

- [ ] Changes are compatible with supported container runtimes (Docker, Podman)
- [ ] Changes are compatible with supported operating systems
- [ ] Backward compatibility maintained (or breaking changes documented)

## Additional Notes

<!-- Add any additional notes for reviewers -->

## Screenshots/Logs

<!-- If applicable, add screenshots or log output to help explain your changes -->

```
# Paste any relevant log output here
```

---

**For Reviewers:**

- [ ] Code review completed
- [ ] Security implications reviewed
- [ ] Test coverage is adequate
- [ ] Documentation is clear and complete
- [ ] Breaking changes are properly communicated