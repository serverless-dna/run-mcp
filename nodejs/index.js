#!/usr/bin/env node
/**
 * Sample MCP Server Entry Point
 * Demonstrates ES module support and MCP SDK integration
 */

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';

/**
 * Sample MCP server implementation
 * This is a minimal example - real MCP servers should implement
 * specific tools, resources, or prompts based on their functionality
 */
class SampleMCPServer {
  constructor() {
    this.server = new Server(
      {
        name: 'sample-mcp-server',
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
            description: 'Echo back the input',
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
              text: `Echo: ${request.params.arguments.message}`,
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
    console.error('Sample MCP server running on stdio');
  }
}

// Run the server if this file is executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  const server = new SampleMCPServer();
  server.run().catch((error) => {
    console.error('Server error:', error);
    process.exit(1);
  });
}

export { SampleMCPServer };