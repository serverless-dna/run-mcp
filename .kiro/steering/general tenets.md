---
inclusion: always
---
<!------------------------------------------------------------------------------------
   Add rules to this file or a short description and have Kiro refine them for you.
   
   Learn about inclusion modes: https://kiro.dev/docs/steering/#inclusion-modes
-------------------------------------------------------------------------------------> 
- When needing to compile the run-mcp go command.  ALways use the `make build-run-mcp` action.  It places the binary into the correct location in build/run-mcp.  
- Always is make targets for testing so we can be sure all tests are included in the make targets.
- All project tests, linting, validations must be accessible through make targets.