# Selin: AI-Powered Learning System for Golang & Blockchain

Selin is a personal AI-powered knowledge management and learning assistant focused on Golang, blockchain development (Cosmos/Celestia), and cryptographic mathematics. It continuously ingests content from Reddit, Twitter, GitHub, and local files, transforms it into semantic vectors, and provides contextualized answers through Claude AI via REST/WebSocket APIs.

## 🏗️ Architecture

Selin uses a hybrid architecture with:
- **Dedicated Server**: Hosts Weaviate (vector DB), PostgreSQL (metadata), Redis (cache), and NFS storage
- **K3s Cluster**: 20× Raspberry Pi 4 nodes running ARM64-optimized Go microservices
- **GitOps Deployment**: ArgoCD + Kustomize for declarative, version-controlled deployments

## 🚀 Features

- **Multi-Source Ingestion**: Reddit, Twitter, GitHub, local Markdown/PDF files
- **Semantic Search**: OpenAI embeddings + Weaviate for intelligent content discovery
- **Claude AI Integration**: MCP gateway for natural language Q&A and learning paths
- **ARM64 Optimized**: Efficient resource usage on Raspberry Pi hardware
- **Configurable**: All customization via `user/` directory YAML files
- **Observable**: Prometheus metrics, Grafana dashboards, Loki logs, Alertmanager

## 📁 Project Structure

```
selin-context-extender/
├── services/                 # Go microservices
│   ├── api-gateway/         # REST API with rate limiting
│   ├── ws/                  # WebSocket streaming service
│   ├── reddit-collector/    # Reddit content ingestion
│   ├── twitter-collector/   # Twitter content ingestion
│   ├── github-collector/    # GitHub repository tracking
│   ├── content-processor/   # Text normalization & cleaning
│   ├── vector-generator/    # OpenAI embedding generation
│   ├── concept-mapper/      # Go/blockchain concept extraction
│   └── mcp-server/         # Claude AI integration
├── infra/                   # Kubernetes manifests
│   ├── weaviate/           # Vector database deployment
│   ├── postgresql/         # SQL database deployment
│   ├── redis/              # Cache deployment
│   ├── monitoring/         # Prometheus & Grafana
│   └── cronjobs/           # Batch processing jobs
├── user/                    # User configuration (customizable)
│   ├── sources.yaml        # Data source configuration
│   ├── preferences.yaml    # Learning preferences
│   └── schedules.yaml      # Collection schedules
├── config/                  # Application configuration
├── credentials/             # API keys and secrets
└── templates/              # Kubernetes templates
```

## 🛠️ Prerequisites

### Required Software
- **Go 1.22+** (ARM64 optimized) - Should be pre-installed
- **kubectl v1.26.3+** - Installed via `./scripts/install-tools.sh`
- **Helm v3.12.0+** - Installed via `./scripts/install-tools.sh`
- **ArgoCD CLI v2.7.3+** - Installed via `./scripts/install-tools.sh`
- **Cursor** (AI-powered IDE) - For development

### Required Infrastructure
- **Kubernetes Cluster** (K3s recommended for Raspberry Pi)
- **NFS Server** for shared storage
- **External APIs**: OpenAI, Claude (via MCP), Reddit, Twitter, GitHub

## 🔧 Quick Start

### 1. Environment Setup

```bash
# Clone and setup project
git clone <your-repo-url> selin-context-extender
cd selin-context-extender

# Install required tools (kubectl v1.26.3, Helm v3.12.0, ArgoCD CLI v2.7.3)
./scripts/install-tools.sh

# Verify installation
./scripts/verify-setup.sh
```

## 🏃‍♂️ Running Selin

You have two options to run Selin:

### Option 1: Local Development (Recommended for Testing)
```bash
# Run services locally for development and testing
./scripts/run-local.sh
```
This will:
- Start Redis in Docker (or prompt for local Redis)
- Build and test both Go services
- Run API Gateway on port 8080
- Run WebSocket service on port 8081
- Provide testing commands and endpoints

### Option 2: Full Kubernetes Deployment
```bash
# Deploy to Kubernetes cluster (requires running cluster)
./scripts/run-selin.sh
```
This will:
- Deploy all infrastructure (PostgreSQL, Redis, Weaviate)
- Set up monitoring (Prometheus)
- Test services end-to-end
- Provide access information

## ⚙️ Configuration (For Production Deployment)

### Configure Secrets
Edit `infra/secrets.yaml` and update the base64-encoded values:

```bash
# Example: Encode your actual API keys
echo -n "your-actual-openai-key" | base64
echo -n "your-actual-claude-key" | base64
echo -n "your-actual-postgres-password" | base64
```

### Update NFS Configuration
Edit `infra/nfs/pv.yaml` and update:
- `YOUR_NFS_SERVER_IP` → Your actual NFS server IP
- `/path/to/selin/data` → Your actual NFS path

## 📊 API Usage

### REST API

```bash
# Health check
curl http://api-gateway:8080/health

# Query the AI
curl -X POST http://api-gateway:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -H "X-User-ID: your-user-id" \
  -d '{"prompt": "Explain Cosmos SDK validators"}'
```

### WebSocket

```javascript
const ws = new WebSocket('ws://websocket-service:8081/ws');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('AI Response:', data);
};
```

## ⚙️ Configuration

### Data Sources (`user/sources.yaml`)

Configure which Reddit subreddits, Twitter accounts, and GitHub repositories to monitor:

```yaml
reddit:
  subreddits: ["golang", "cosmosdev", "cryptography"]
  collection_interval: "5m"
  
twitter:
  hashtags: ["#cosmos", "#golang"]
  collection_interval: "10m"
  
github:
  repositories: ["cosmos/cosmos-sdk", "golang/go"]
  collection_interval: "30m"
```

### Learning Preferences (`user/preferences.yaml`)

Customize learning focus and content filtering:

```yaml
learning_focus:
  primary_topics: ["golang", "blockchain", "cryptography"]
  skill_level:
    golang: "intermediate"
    blockchain: "beginner"
```

### Schedules (`user/schedules.yaml`)

Control when data collection and processing jobs run:

```yaml
data_collection:
  reddit_collector:
    schedule: "*/5 * * * *"  # Every 5 minutes
    enabled: true
```

## 📈 Monitoring

Access monitoring dashboards:

- **Grafana**: `http://grafana:3000` - Metrics dashboards
- **Prometheus**: `http://prometheus:9090` - Raw metrics
- **AlertManager**: `http://alertmanager:9093` - Alert management

Key metrics tracked:
- API request latency and throughput
- Data ingestion rates
- Vector generation performance
- Resource usage on Raspberry Pi nodes

## 🔐 Security

- **TLS Everywhere**: All service-to-service communication encrypted
- **Rate Limiting**: 60 requests/minute per user, 120 for collectors
- **Secrets Management**: Kubernetes Secrets + Sealed Secrets
- **RBAC**: Role-based access control for cluster resources

## 🛠️ Installation Scripts

The project includes helpful scripts in the `scripts/` directory:

### `install-tools.sh`
Automatically installs the required tools with the correct versions:
```bash
./scripts/install-tools.sh
```
- Installs kubectl v1.26.3
- Installs Helm v3.12.0  
- Installs ArgoCD CLI v2.7.3
- Detects your OS and architecture automatically
- Installs to `/usr/local/bin` (with sudo) or `~/bin`

### `verify-setup.sh`
Comprehensive verification of your environment:
```bash
./scripts/verify-setup.sh
```
- Checks all required tools and versions
- Validates project structure
- Builds and tests Go services
- Validates Kubernetes manifests
- Provides detailed success/failure report

## 🧪 Development

### Running Tests

```bash
# Test API Gateway
cd services/api-gateway
go test -v .

# Test WebSocket service
cd ../ws
go test -v .
```

### Local Development

```bash
# Run API Gateway locally
cd services/api-gateway
PORT=8080 REDIS_URL=localhost:6379 go run .

# Run WebSocket service locally
cd ../ws
PORT=8081 go run .
```

## 📋 Implementation Status

### ✅ Completed
- [x] Project structure and directory setup
- [x] API Gateway with health/metrics endpoints
- [x] WebSocket service with real-time streaming
- [x] Redis rate limiting (60 requests/minute)
- [x] Infrastructure manifests (Weaviate, PostgreSQL, Redis)
- [x] Monitoring setup (Prometheus configuration)
- [x] Batch job configurations (CronJobs)
- [x] User configuration templates

### 🚧 In Progress
- [ ] Individual data collector implementations
- [ ] MCP Server integration with Claude AI
- [ ] Content processing pipeline
- [ ] Vector generation and storage
- [ ] ArgoCD deployment setup

### 📅 Planned
- [ ] Production deployment scripts
- [ ] Advanced monitoring dashboards
- [ ] Learning path generation
- [ ] Mobile/web interface

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📝 License

[Add your license here]

## 🆘 Support

For questions and support:
1. Check the implementation plan in `.cursor/rules/implementation_plan.mdc`
2. Review other documentation in `.cursor/rules/`
3. Open an issue on GitHub

---

**Note**: This is a personal learning system optimized for single-user deployment on ARM64 Raspberry Pi clusters. For multi-tenant or enterprise use cases, additional modifications would be required.
