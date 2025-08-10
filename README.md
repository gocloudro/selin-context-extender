# Selin - Personal AI Learning System

A microservices-based personal knowledge management system that ingests content from multiple sources and provides intelligent insights through Claude AI integration. Designed for learning Golang, blockchain development (Cosmos/Celestia ecosystem), and cryptographic mathematics.

## Features

### **Multi-source ingestion**
* Reddit, Twitter, GitHub data collection
* File watching for Markdown, PDFs, and other documents
* Configurable sources and collection frequencies

### **Content processing**
* Cleaning, classification, and metadata extraction
* Concept mapping and relationship building
* Go code analysis with AST parsing
* Cross-repository implementation comparison

### **Vector embedding**
* Weaviate-powered semantic search
* OpenAI Embeddings API integration
* Intelligent content relationships

### **Claude integration**
* Search, reasoning, and learning path generation via MCP
* Natural language queries for code explanation
* Progress tracking and personalized recommendations

### **Scalable architecture**
* Runs in K3s Kubernetes clusters
* Lightweight for Raspberry Pi 4 deployments
* 22 microservices with clear separation of concerns

### **User-centric configuration**
* All settings customizable through `user/` directory
* No code changes required for personalization
* Hot configuration reloading support

## Architecture

### Hybrid Deployment Model
- **Dedicated Server**: Weaviate vector database, PostgreSQL, Redis, NFS storage
- **K3s Cluster**: 20x Raspberry Pi 4 running lightweight Go microservices

### System Components (22 Services)

#### Infrastructure Services (Dedicated Server)
1. **Weaviate Vector Database** - Semantic search and embeddings
2. **PostgreSQL Database** - Metadata and relationships  
3. **Redis Cache** - Rate limiting and caching
4. **NFS File Server** - Shared content storage

#### Data Collection Microservices (K3s)
5. **Reddit Collector** - Subreddit content ingestion
6. **Twitter Collector** - Hashtag and account monitoring
7. **GitHub Collector** - Repository code, issues, PRs
8. **File Watcher** - Markdown and PDF monitoring
9. **PDF Processor** - Research paper extraction

#### Processing & Intelligence (K3s)
10. **Content Processor** - Data normalization and enrichment
11. **Vector Generator** - OpenAI embeddings generation
12. **Concept Mapper** - Knowledge relationship building
13. **Code Analyzer** - Go AST parsing and analysis

#### API & Interface (K3s)
14. **MCP Server** - Claude AI integration
15. **REST API Gateway** - External API access
16. **WebSocket Service** - Real-time updates

#### Batch Processing (K3s CronJobs)
17. **Daily Processor** - Aggregation and optimization
18. **Weekly Analyzer** - Progress analysis and reporting
19. **Backup Service** - Data backup and recovery

#### Monitoring & Operations (K3s)
20. **Health Monitor** - Service health monitoring
21. **Log Aggregator** - Centralized logging
22. **Metrics Collector** - Performance metrics

## Configuration

All system behavior is customizable through the `user/` directory structure:

```
user/
├── config/
│   ├── environment.env          # Environment variables
│   ├── sources.yaml            # Data source configurations  
│   ├── schedules.yaml          # Update frequencies and timing
│   └── preferences.yaml        # User learning preferences
├── credentials/
│   ├── reddit.env              # Reddit API credentials
│   ├── twitter.env             # Twitter API credentials
│   ├── github.env              # GitHub API credentials
│   └── openai.env              # OpenAI API key
└── templates/
    ├── learning-paths.yaml     # Custom learning path templates
    └── concepts.yaml           # Custom concept definitions
```

### Example Configurations

**Customize data sources** (`user/config/sources.yaml`):
```yaml
reddit:
  subreddits: ["golang", "cosmosdev", "cryptography"]
  max_posts_per_subreddit: 50

github:
  repositories:
    - "celestiaorg/celestia-core"
    - "cosmos/cosmos-sdk" 
    - "cometbft/cometbft"
```

**Adjust collection schedules** (`user/config/schedules.yaml`):
```yaml
collection_schedules:
  reddit_collector: "0 */2 * * *"    # Every 2 hours
  github_collector: "0 */4 * * *"    # Every 4 hours
```

**Set learning preferences** (`user/config/preferences.yaml`):
```yaml
learning_focus:
  primary_topics: ["golang", "cosmos-sdk", "celestia"]
  skill_levels:
    golang: "novice"
    blockchain: "intermediate"

weekly_stats:
  enabled: true
  delivery_day: "sunday"
  include_sections: ["learning_progress", "new_concepts"]
```

## Dependencies

### **Infrastructure Requirements**
* Dedicated server [2-4 cores, 8-16GB RAM] for heavy services
* K3s cluster [1-2 cores/node, 4-8GB RAM/node] for microservices
* Docker & Docker Compose
* K3s/Kubernetes

### **External Services**
* Weaviate vector database
* PostgreSQL database
* Redis cache
* OpenAI API (embeddings)

### **API Access**
* Reddit API credentials
* Twitter API v2 credentials  
* GitHub API token
* Claude API access (via MCP)

### **Development**
* Go 1.22+
* ARM64 support for Raspberry Pi deployment

## Quick Start

### 1. Infrastructure Setup

**Dedicated Server (Docker Compose):**
```bash
# Deploy core infrastructure
docker-compose up -d weaviate postgres redis
```

**K3s Cluster:**
```bash
# Deploy microservices
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmaps.yaml
kubectl apply -f k8s/secrets.yaml
kubectl apply -f k8s/services/
```

### 2. Configuration

**Copy and customize configuration templates:**
```bash
cp -r templates/user/ user/
# Edit user/config/*.yaml files
# Add API credentials to user/credentials/*.env
```

**Generate Kubernetes configs:**
```bash
./scripts/generate-configs.sh user/
```

### 3. Deployment

**Phase 1 - Foundation:**
```bash
# Deploy file watcher and MCP server
kubectl apply -f k8s/services/file-watcher.yaml
kubectl apply -f k8s/services/mcp-server.yaml
```

**Phase 2 - Data Collection:**
```bash
# Deploy API collectors
kubectl apply -f k8s/services/reddit-collector.yaml
kubectl apply -f k8s/services/github-collector.yaml
```

**Phase 3 - Intelligence:**
```bash
# Deploy processing services
kubectl apply -f k8s/services/content-processor.yaml
kubectl apply -f k8s/services/concept-mapper.yaml
```

## Usage Examples

### Claude Integration Queries

**Recent developments:**
```
"Show me recent developments in Go blockchain libraries"
```

**Code explanation:**
```  
"How does consensus work in Celestia vs Cosmos SDK?"
```

**Learning guidance:**
```
"What math concepts do I need to understand this crypto paper?"
```

**Progress tracking:**
```
"Generate my learning path for Cosmos development"
```

### API Examples

**Search for content:**
```bash
curl -X POST http://mcp-server/search \
  -d '{"query": "golang channels", "sources": ["github", "reddit"]}'
```

**Get learning recommendations:**
```bash
curl -X GET http://mcp-server/recommendations \
  -H "User-ID: your-user-id"
```

## Development

### Building Services

**ARM64 optimized builds:**
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o app
docker buildx build --platform linux/arm64 -t selin/service:arm64 .
```

### Implementation Phases

**Phase 1: Foundation (Weeks 1-3)**
- Weaviate setup and schema design
- File Watcher service (first Go microservice)
- Basic MCP Server for Claude integration

**Phase 2: Core Collection (Weeks 4-6)**  
- GitHub Collector with selective processing
- Content Processor for data normalization
- Daily batch processing pipeline

**Phase 3: API Integration (Weeks 7-10)**
- Reddit and Twitter collectors
- Code Analyzer with Go AST parsing
- Concept Mapper for knowledge relationships

**Phase 4: Intelligence & Monitoring (Weeks 11+)**
- Advanced MCP tools and analytics
- Monitoring and observability stack
- Performance optimization

### Project Structure

```
selin/
├── cmd/                    # Service entry points
├── internal/               # Private application code
├── pkg/                    # Public library code
├── api/                    # API definitions
├── deployments/            # Kubernetes manifests
├── scripts/                # Build and deployment scripts
├── user/                   # User configuration directory
└── docs/                   # Documentation
```

## Monitoring

### Health Checks
All services expose standard health endpoints:
- `GET /health` - Service health status
- `GET /ready` - Readiness for traffic
- `GET /metrics` - Prometheus metrics

### Observability Stack
- **Logging**: Centralized via Log Aggregator service
- **Metrics**: Prometheus metrics collection
- **Monitoring**: Service health and performance tracking
- **Alerting**: Configurable alerts for service issues

## Contributing

### Development Workflow
1. Fork the repository
2. Create feature branch
3. Implement changes with tests
4. Update documentation
5. Submit pull request

### Testing
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Integration tests
go test -tags=integration ./...
```

### Code Standards
- Follow standard Go project layout
- Implement comprehensive error handling
- Add unit tests for all business logic
- Use structured logging
- Document all public APIs

## License

[Your chosen license]

## Support

For issues and questions:
- Create GitHub issues for bugs and feature requests
- Check documentation in `docs/` directory  
- Review configuration examples in `templates/`

---

**Built for learning Go, blockchain development, and cryptographic mathematics through intelligent content aggregation and AI-powered insights.**
