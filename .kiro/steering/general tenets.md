---
inclusion: always
---
<!------------------------------------------------------------------------------------
   Add rules to this file or a short description and have Kiro refine them for you.
   
   Learn about inclusion modes: https://kiro.dev/docs/steering/#inclusion-modes
-------------------------------------------------------------------------------------> 
- Always ensure there are make targets for all added tests so we can be sure all tests are used in CI/CD.
- You MUST us the emakefile to run tests and build run-mcp
- A go code changeis done only when the `make test-run-mcp` tests are all passing.
- A change to container files is done only when `make test` tests are all passing.
