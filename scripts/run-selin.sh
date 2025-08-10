#!/bin/bash

# Selin Project - Deployment and Running Script
# This script helps you deploy and run the Selin system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Selin AI Learning System Deployment ===${NC}"
echo ""

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to wait for deployment
wait_for_deployment() {
    local namespace=$1
    local deployment=$2
    local timeout=${3:-300}
    
    echo "Waiting for $deployment to be ready..."
    kubectl wait --for=condition=available --timeout=${timeout}s deployment/$deployment -n $namespace
}

# Function to wait for pod
wait_for_pod() {
    local namespace=$1
    local selector=$2
    local timeout=${3:-300}
    
    echo "Waiting for pod with selector $selector to be ready..."
    kubectl wait --for=condition=ready --timeout=${timeout}s pod -l $selector -n $namespace
}

echo -e "${YELLOW}Step 1: Pre-deployment checks${NC}"

# Check if tools are installed
if ! command_exists kubectl; then
    echo -e "${RED}kubectl not found. Run ./scripts/install-tools.sh first${NC}"
    exit 1
fi

if ! command_exists helm; then
    echo -e "${RED}Helm not found. Run ./scripts/install-tools.sh first${NC}"
    exit 1
fi

# Check if we have a Kubernetes cluster
if ! kubectl cluster-info >/dev/null 2>&1; then
    echo -e "${RED}No Kubernetes cluster found. Please ensure you have:${NC}"
    echo "1. A running Kubernetes cluster (K3s recommended for Pi)"
    echo "2. kubectl configured to connect to your cluster"
    echo ""
    echo "For local testing, you can use:"
    echo "- Docker Desktop with Kubernetes enabled"
    echo "- minikube"
    echo "- kind (Kubernetes in Docker)"
    exit 1
fi

echo -e "${GREEN}✓ Kubernetes cluster is accessible${NC}"

echo ""
echo -e "${YELLOW}Step 2: Configuration Setup${NC}"

# Check if secrets are configured
if grep -q "changmeplease\|your-" infra/secrets.yaml; then
    echo -e "${YELLOW}⚠️  Default secrets detected in infra/secrets.yaml${NC}"
    echo "Please update the following secrets with real values:"
    echo "- postgres-password (currently: changmeplease)"
    echo "- openai-api-key (currently: your-openai-api-key)"
    echo "- claude-api-key (currently: your-claude-api-key)"
    echo ""
    read -p "Continue with default secrets for testing? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Please update infra/secrets.yaml and run this script again."
        exit 1
    fi
    echo -e "${YELLOW}Continuing with default secrets for testing...${NC}"
fi

# Check NFS configuration
if grep -q "YOUR_NFS_SERVER_IP" infra/nfs/pv.yaml; then
    echo -e "${YELLOW}⚠️  NFS configuration needs updating in infra/nfs/pv.yaml${NC}"
    echo "For local testing, we'll skip NFS and use local storage."
    
    # Create a local storage version
    cat > /tmp/local-storage-class.yaml << EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: nfs-storage
provisioner: kubernetes.io/no-provisioner
reclaimPolicy: Retain
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
EOF
    
    echo -e "${YELLOW}Created temporary local storage class for testing${NC}"
fi

echo ""
echo -e "${YELLOW}Step 3: Infrastructure Deployment${NC}"

# Create namespace
echo "Creating namespace..."
kubectl apply -f infra/namespace.yaml

# Apply storage configuration
if [ -f /tmp/local-storage-class.yaml ]; then
    kubectl apply -f /tmp/local-storage-class.yaml
else
    kubectl apply -f infra/nfs/
fi

# Apply secrets and config
echo "Applying secrets and configuration..."
kubectl apply -f infra/secrets.yaml

# Deploy infrastructure services
echo "Deploying infrastructure services..."
kubectl apply -f infra/postgresql/
kubectl apply -f infra/redis/
kubectl apply -f infra/weaviate/

# Wait for infrastructure to be ready
echo ""
echo -e "${YELLOW}Step 4: Waiting for infrastructure to be ready${NC}"

wait_for_deployment "selin" "postgresql" 180
wait_for_deployment "selin" "redis" 120
wait_for_deployment "selin" "weaviate" 240

echo -e "${GREEN}✓ Infrastructure services are ready${NC}"

echo ""
echo -e "${YELLOW}Step 5: Local Service Testing${NC}"

# Test API Gateway locally first
echo "Testing API Gateway locally..."
cd services/api-gateway

# Set environment variables for local testing
export PORT=8080
export REDIS_URL="localhost:6379"
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5432"
export POSTGRES_DB="selin"
export POSTGRES_USER="postgres"
export POSTGRES_PASSWORD="changmeplease"

# Port forward Redis for local testing
echo "Setting up port forwarding for Redis..."
kubectl port-forward svc/redis 6379:6379 -n selin &
REDIS_PID=$!

# Wait a moment for port forward to establish
sleep 3

echo "Starting API Gateway locally..."
go run . &
API_GATEWAY_PID=$!

# Wait for API Gateway to start
sleep 5

# Test the API Gateway
echo "Testing API Gateway endpoints..."
if curl -s http://localhost:8080/health | grep -q "OK"; then
    echo -e "${GREEN}✓ API Gateway health check passed${NC}"
else
    echo -e "${RED}✗ API Gateway health check failed${NC}"
    kill $API_GATEWAY_PID $REDIS_PID 2>/dev/null
    exit 1
fi

# Test query endpoint
echo "Testing query endpoint..."
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Hello, Selin!"}')

if echo "$RESPONSE" | grep -q "response"; then
    echo -e "${GREEN}✓ Query endpoint is working${NC}"
    echo "Response: $RESPONSE"
else
    echo -e "${RED}✗ Query endpoint failed${NC}"
    echo "Response: $RESPONSE"
fi

# Test WebSocket service
cd ../ws
echo "Testing WebSocket service..."
export PORT=8081
go run . &
WS_PID=$!

sleep 3

if curl -s http://localhost:8081/health | grep -q "OK"; then
    echo -e "${GREEN}✓ WebSocket service health check passed${NC}"
else
    echo -e "${RED}✗ WebSocket service health check failed${NC}"
fi

# Cleanup local processes
echo ""
echo "Cleaning up local test processes..."
kill $API_GATEWAY_PID $WS_PID $REDIS_PID 2>/dev/null || true

cd ../..

echo ""
echo -e "${YELLOW}Step 6: Monitoring Setup${NC}"

# Deploy monitoring
echo "Deploying monitoring services..."
kubectl apply -f infra/monitoring/

wait_for_deployment "selin" "prometheus" 180

echo -e "${GREEN}✓ Monitoring services deployed${NC}"

echo ""
echo -e "${GREEN}🎉 Selin system is running successfully!${NC}"
echo ""
echo -e "${BLUE}=== Access Information ===${NC}"
echo ""
echo "Services running in Kubernetes cluster:"

# Get service information
kubectl get services -n selin

echo ""
echo "To access services locally, use port forwarding:"
echo ""
echo -e "${YELLOW}# PostgreSQL${NC}"
echo "kubectl port-forward svc/postgresql 5432:5432 -n selin"
echo ""
echo -e "${YELLOW}# Redis${NC}"
echo "kubectl port-forward svc/redis 6379:6379 -n selin"
echo ""
echo -e "${YELLOW}# Weaviate${NC}"
echo "kubectl port-forward svc/weaviate 8080:8080 -n selin"
echo ""
echo -e "${YELLOW}# Prometheus${NC}"
echo "kubectl port-forward svc/prometheus 9090:9090 -n selin"
echo ""
echo -e "${BLUE}=== Next Steps ===${NC}"
echo ""
echo "1. Configure your data sources in user/sources.yaml"
echo "2. Update API keys in infra/secrets.yaml for production use"
echo "3. Implement the data collector services"
echo "4. Set up ArgoCD for GitOps deployment"
echo ""
echo "For development:"
echo "- Run local services: cd services/api-gateway && go run ."
echo "- Run tests: go test ./services/..."
echo "- Monitor logs: kubectl logs -f deployment/postgresql -n selin"
echo ""
echo -e "${GREEN}Happy learning with Selin! 🚀${NC}"
