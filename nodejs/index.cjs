#!/usr/bin/env node
/**
 * Sample MCP Server Entry Point (CommonJS version)
 * Demonstrates CommonJS support and MCP SDK integration
 */

const { Server } = require('@modelcontextprotocol/sdk/server/index.js');
const { StdioServerTransport } = require('@modelcontextprotocol/sdk/server/stdio.js');

/**
 * Sample MCP server implementation (CommonJS)
 */
class SampleMCPServer {
  constructor() {
    this.server = new Server(
      {
        name: 'sample-mcp-server-cjs',
        version: '1.0.0',
      },
      {
        capabilities: {
          tools: {},
          resources: {},
          prompts: {},
        },
      }
    );

    this.setupHandlers();
  }

  setupHandlers() {
    // Add sample tool
    this.server.setRequestHandler('tools/list', async () => {
      return {
        tools: [
          {
            name: 'echo',
            description: 'Echo back the input (CommonJS)',
            inputSchema: {
              type: 'object',
              properties: {
                message: {
                  type: 'string',
                  description: 'Message to echo back',
                },
              },
              required: ['message'],
            },
          },
        ],
      };
    });

    this.server.setRequestHandler('tools/call', async (request) => {
      if (request.params.name === 'echo') {
        return {
          content: [
            {
              type: 'text',
              text: `Echo (CommonJS): ${request.params.arguments.message}`,
            },
          ],
        };
      }
      throw new Error(`Unknown tool: ${request.params.name}`);
    });
  }

  async run() {
    const transport = new StdioServerTransport();
    await this.server.connect(transport);
    console.error('Sample MCP server (CommonJS) running on stdio');
  }
}

// Run the server if this file is executed directly
if (require.main === module) {
  const server = new SampleMCPServer();
  server.run().catch((error) => {
    console.error('Server error:', error);
    process.exit(1);
  });
}

module.exports = { SampleMCPServer };