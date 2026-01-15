<!------------------------------------------------------------------------------------
   Developer Experience - Cross-Platform Support
   
   Learn about inclusion modes: https://kiro.dev/docs/steering/#inclusion-modes
------------------------------------------------------------------------------------>

## Cross-Platform Development

- Developer experience MUST be equally supported on both Linux and MacOS
- All testing workflows, scripts, and make targets must work identically on Linux and MacOS
- Development setup instructions and tooling must support both platforms without platform-specific workarounds
- CI/CD pipelines should validate functionality on both Linux and MacOS where applicable
- Avoid platform-specific dependencies unless absolutely necessary; when required, provide equivalent alternatives for both platforms and auto-detect the need for which alternative.
- Documentation should highlight any platform-specific considerations upfront

## Container Runtime Support

- Developer experience MUST support both Docker and Podman equally
- Never assume a specific container runtime is available
- Scripts and make targets should detect and work with either Docker or Podman
- Documentation and examples should be runtime-agnostic or provide instructions for both
- Use generic container commands that work across both runtimes where possible
