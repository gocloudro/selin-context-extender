#!/usr/bin/env python3
"""
Selin MCP Server for Claude Desktop
Provides access to Selin's knowledge base via MCP (Model Context Protocol)
"""

import asyncio
import json
import logging
import os
import sys
from typing import Any, Dict, List, Optional

import httpx
from mcp.server import Server
from mcp.server.models import InitializationOptions
from mcp.server.stdio import stdio_server
from mcp.types import (
    CallToolRequest,
    CallToolResult,
    ListToolsRequest,
    ListToolsResult,
    TextContent,
    Tool,
)

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("selin-mcp")

# Selin API configuration
SELIN_API_BASE = os.getenv("SELIN_API_BASE", "http://localhost:8084")

class SelinMCPServer:
    def __init__(self):
        self.server = Server("selin")
        self.setup_handlers()
    
    def setup_handlers(self):
        @self.server.list_tools()
        async def list_tools() -> List[Tool]:
            """List available Selin tools for Claude"""
            return [
                Tool(
                    name="search_selin_content",
                    description="Search Selin's knowledge base for content related to Go, blockchain, or cryptography",
                    inputSchema={
                        "type": "object",
                        "properties": {
                            "query": {
                                "type": "string",
                                "description": "Search query (e.g., 'golang concurrency', 'cosmos blockchain')"
                            },
                            "limit": {
                                "type": "number",
                                "description": "Maximum number of results to return",
                                "default": 10
                            },
                            "platform": {
                                "type": "string", 
                                "description": "Filter by source platform",
                                "enum": ["reddit", "slack", "file_upload", "all"],
                                "default": "all"
                            }
                        },
                        "required": ["query"]
                    }
                ),
                Tool(
                    name="get_learning_progress",
                    description="Get the user's learning progress for specific topics",
                    inputSchema={
                        "type": "object",
                        "properties": {
                            "topic": {
                                "type": "string",
                                "description": "Learning topic (e.g., 'golang', 'blockchain', 'cryptography')"
                            }
                        },
                        "required": ["topic"]
                    }
                ),
                Tool(
                    name="get_recent_content", 
                    description="Get recently collected content from Selin's knowledge base",
                    inputSchema={
                        "type": "object",
                        "properties": {
                            "hours": {
                                "type": "number",
                                "description": "Number of hours back to look",
                                "default": 24
                            },
                            "platform": {
                                "type": "string",
                                "description": "Filter by source platform", 
                                "enum": ["reddit", "slack", "file_upload", "all"],
                                "default": "all"
                            }
                        }
                    }
                ),
                Tool(
                    name="analyze_content_trends",
                    description="Analyze trends in collected content and learning topics",
                    inputSchema={
                        "type": "object", 
                        "properties": {
                            "days": {
                                "type": "number",
                                "description": "Number of days to analyze",
                                "default": 7
                            },
                            "topic": {
                                "type": "string",
                                "description": "Focus on specific topic"
                            }
                        }
                    }
                )
            ]

        @self.server.call_tool()
        async def call_tool(name: str, arguments: Dict[str, Any]) -> CallToolResult:
            """Handle tool calls from Claude"""
            logger.info(f"Tool called: {name} with arguments: {arguments}")
            
            try:
                if name == "search_selin_content":
                    result = await self.search_content(arguments)
                elif name == "get_learning_progress":
                    result = await self.get_learning_progress(arguments)
                elif name == "get_recent_content":
                    result = await self.get_recent_content(arguments)
                elif name == "analyze_content_trends":
                    result = await self.analyze_trends(arguments)
                else:
                    raise ValueError(f"Unknown tool: {name}")
                
                return CallToolResult(content=[TextContent(type="text", text=result)])
                
            except Exception as e:
                logger.error(f"Tool call failed: {e}")
                return CallToolResult(
                    content=[TextContent(type="text", text=f"Error: {str(e)}")],
                    isError=True
                )

    async def search_content(self, args: Dict[str, Any]) -> str:
        """Search Selin's content database"""
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{SELIN_API_BASE}/mcp/call",
                json={"name": "search_content", "arguments": args},
                timeout=30.0
            )
            response.raise_for_status()
            data = response.json()
            
            if data.get("isError"):
                raise Exception(data["content"][0]["text"])
            
            return data["content"][0]["text"]

    async def get_learning_progress(self, args: Dict[str, Any]) -> str:
        """Get learning progress from Selin"""
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{SELIN_API_BASE}/mcp/call",
                json={"name": "get_learning_progress", "arguments": args},
                timeout=30.0
            )
            response.raise_for_status()
            data = response.json()
            
            if data.get("isError"):
                raise Exception(data["content"][0]["text"])
            
            return data["content"][0]["text"]

    async def get_recent_content(self, args: Dict[str, Any]) -> str:
        """Get recent content from Selin"""
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{SELIN_API_BASE}/mcp/call", 
                json={"name": "get_recent_content", "arguments": args},
                timeout=30.0
            )
            response.raise_for_status()
            data = response.json()
            
            if data.get("isError"):
                raise Exception(data["content"][0]["text"])
            
            return data["content"][0]["text"]

    async def analyze_trends(self, args: Dict[str, Any]) -> str:
        """Analyze content trends in Selin"""
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{SELIN_API_BASE}/mcp/call",
                json={"name": "analyze_content_trends", "arguments": args},
                timeout=30.0
            )
            response.raise_for_status()
            data = response.json()
            
            if data.get("isError"):
                raise Exception(data["content"][0]["text"])
            
            return data["content"][0]["text"]

    async def run(self):
        """Run the MCP server"""
        async with stdio_server() as (read_stream, write_stream):
            await self.server.run(
                read_stream,
                write_stream, 
                InitializationOptions(
                    server_name="selin",
                    server_version="1.0.0",
                    capabilities=self.server.get_capabilities(
                        notification_options=None,
                        experimental_capabilities=None
                    )
                )
            )

async def main():
    """Main entry point"""
    logger.info("Starting Selin MCP Server...")
    server = SelinMCPServer()
    await server.run()

if __name__ == "__main__":
    asyncio.run(main())
