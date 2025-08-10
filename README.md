# Selin Context Extender

---

## Features

- **Multi-source ingestion**
  - Reddit, Twitter, GitHub data collection
  - File watching for Markdown, PDFs, and other documents
- **Content processing**
  - Cleaning, classification, and metadata extraction
  - Concept mapping and relationship building
- **Vector embedding**
  - Weaviate-powered semantic search
  - OpenAI Embeddings API integration
- **Claude integration**
  - Search, reasoning, and learning path generation via MCP
- **Scalable architecture**
  - Runs in K3s Kubernetes clusters
  - Lightweight for Raspberry Pi 4 deployments

---

## Dependencies

- Dedicated K3s server [1-2 cores/core, 4-8GB ram/core]
- Weaviate
- PostgreSQL
- Redis
- Runtime dependencies:
    - Docker & Docker Compose
    - K3s/Kubernetes
    - Go 1.22+
    - Access to:
        - Reddit API
        - Twitter API v2
        - Github API
