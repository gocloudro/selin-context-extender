# 🤖 Adding Selin as an MCP to Claude Desktop

This guide shows you how to connect your Selin knowledge system to Claude Desktop using the Model Context Protocol (MCP).

## 🎯 What You'll Get

Once connected, Claude will be able to:
- 🔍 **Search your knowledge base** for Go, blockchain, and cryptography content
- 📊 **Check your learning progress** across different topics
- 📅 **Get recent content** from Reddit, Slack, and uploaded files
- 📈 **Analyze content trends** to guide your learning

## ⚡ Quick Setup

### 1. Install Python MCP Dependencies

```bash
# Install MCP SDK
pip install mcp httpx

# Or if using conda:
conda install -c conda-forge httpx
pip install mcp
```

### 2. Make the MCP Script Executable

```bash
chmod +x services/mcp-server/selin-mcp.py
```

### 3. Start Selin Services

```bash
# Start all Selin services (including MCP server)
./scripts/run-local.sh

# Or start just the MCP server:
cd services/mcp-server
POSTGRES_HOST="localhost" POSTGRES_PORT="5433" \
POSTGRES_USER="postgres" POSTGRES_PASSWORD="changmeplease" \
POSTGRES_DB="selin" PORT=8084 go run main.go
```

### 4. Configure Claude Desktop

#### Option A: Using Python MCP Client (Recommended)

1. **Locate your Claude Desktop config file:**
   - **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
   - **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
   - **Linux**: `~/.config/Claude/claude_desktop_config.json`

2. **Add Selin to your config:**

```json
{
  "mcpServers": {
    "selin": {
      "command": "python3",
      "args": ["/Users/sysrex/git/selin-context-extender/services/mcp-server/selin-mcp.py"],
      "env": {
        "SELIN_API_BASE": "http://localhost:8084"
      }
    }
  }
}
```

**⚠️ Important**: Replace `/Users/sysrex/git/selin-context-extender` with your actual project path!

#### Option B: Direct HTTP Integration

If the Python approach doesn't work, you can use direct HTTP calls:

```json
{
  "mcpServers": {
    "selin": {
      "command": "curl",
      "args": [
        "-X", "POST",
        "-H", "Content-Type: application/json",
        "http://localhost:8084/mcp/call"
      ]
    }
  }
}
```

### 5. Restart Claude Desktop

1. **Quit Claude Desktop completely**
2. **Restart it**
3. **Look for the 🔧 tools icon** in your chat interface

## 🧪 Test the Integration

Once Claude is restarted with Selin connected, try these commands:

### Search Your Knowledge Base
```
Hey Claude, can you search my Selin knowledge base for content about "golang concurrency"?
```

### Check Learning Progress
```
What's my learning progress on blockchain topics?
```

### Get Recent Content
```
Show me what content Selin has collected in the last 24 hours.
```

### Analyze Trends
```
Can you analyze the content trends in my knowledge base over the last week?
```

## 🔧 Available MCP Tools

When connected, Claude has access to these Selin tools:

| Tool | Purpose | Example Usage |
|------|---------|---------------|
| `search_content` | Search knowledge base | "Find posts about Go channels" |
| `get_learning_progress` | Check progress by topic | "How am I doing with cryptography?" |
| `get_recent_content` | Get latest collected content | "What's new today?" |
| `analyze_content_trends` | Analyze learning patterns | "Show me trends this week" |

## 🚀 What Claude Can Do With Your Data

### Smart Learning Assistant
```
"Based on my recent Reddit activity and learning progress, 
what Go topics should I focus on next?"
```

### Content Discovery
```
"Find the most relevant blockchain content from my knowledge base 
that relates to Cosmos SDK development."
```

### Progress Tracking
```
"Create a learning summary showing my progress across all topics 
and suggest areas for improvement."
```

### Personalized Recommendations
```
"Based on my current skill level and recent content, 
recommend specific GitHub repositories I should study."
```

## 🔍 Verification

### Check MCP Server Status
```bash
# Test health
curl http://localhost:8084/health

# List available tools
curl http://localhost:8084/mcp/tools

# Test a search
curl -X POST http://localhost:8084/mcp/call \
  -H "Content-Type: application/json" \
  -d '{"name": "search_content", "arguments": {"query": "golang", "limit": 3}}'
```

### Check Claude Integration
1. Look for 🔧 tools icon in Claude interface
2. Try asking Claude to search your knowledge base
3. Check Claude's responses include Selin data

## 🐛 Troubleshooting

### MCP Server Won't Start
```bash
# Check if port 8084 is free
lsof -i :8084

# Check database connection
docker exec -it selin-postgres psql -U postgres -d selin -c "SELECT COUNT(*) FROM content_metadata;"
```

### Claude Can't Connect
1. **Verify config file path** - make sure you edited the right file
2. **Check permissions** - ensure the Python script is executable
3. **Restart Claude completely** - quit and restart the app
4. **Check logs** - Claude Desktop logs are usually in the same config directory

### No Data in Responses
1. **Check if Reddit collector is running** and storing data
2. **Verify database has content**: `docker exec -it selin-postgres psql -U postgres -d selin -c "SELECT COUNT(*) FROM content_metadata;"`
3. **Upload some test files** to populate the database

## 🎉 Success!

Once working, you'll have Claude powered by your personal knowledge base! 

Ask Claude things like:
- "What are the key Go concepts I should master next?"
- "Summarize the latest blockchain developments from my feeds"
- "Help me create a study plan based on my learning progress"

Your Selin system becomes Claude's memory, making it a truly personalized AI assistant for your Go, blockchain, and cryptography learning journey! 🚀
