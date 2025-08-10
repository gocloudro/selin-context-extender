#!/bin/bash

# Selin Project - Setup Verification Script
# This script verifies that all required tools are installed and ready

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Selin Project Setup Verification ===${NC}"
echo ""

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check version requirements
check_version() {
    local tool=$1
    local current=$2
    local required=$3
    
    if [ "$current" = "$required" ] || [ "$(printf '%s\n' "$required" "$current" | sort -V | head -n1)" = "$required" ]; then
        echo -e "${GREEN}✓${NC} $tool: $current (>= $required required)"
        return 0
    else
        echo -e "${RED}✗${NC} $tool: $current (>= $required required)"
        return 1
    fi
}

ERRORS=0

echo -e "${YELLOW}Checking required tools...${NC}"

# Check Go
if command_exists go; then
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    check_version "Go" "$GO_VERSION" "1.22.0" || ERRORS=$((ERRORS + 1))
else
    echo -e "${RED}✗${NC} Go: Not installed"
    ERRORS=$((ERRORS + 1))
fi

# Check kubectl
if command_exists kubectl; then
    KUBECTL_VERSION=$(kubectl version --client --short 2>/dev/null | grep 'Client Version' | awk '{print $3}' | sed 's/v//' || kubectl version --client 2>/dev/null | grep 'Client Version' | awk '{print $3}' | sed 's/v//')
    check_version "kubectl" "v$KUBECTL_VERSION" "v1.26.3" || ERRORS=$((ERRORS + 1))
else
    echo -e "${RED}✗${NC} kubectl: Not installed"
    ERRORS=$((ERRORS + 1))
fi

# Check Helm
if command_exists helm; then
    HELM_VERSION=$(helm version --short 2>/dev/null | awk '{print $1}' | sed 's/v//' | cut -d'+' -f1 || helm version 2>/dev/null | grep 'Version' | awk '{print $2}' | sed 's/v//' | cut -d'+' -f1)
    check_version "Helm" "v$HELM_VERSION" "v3.12.0" || ERRORS=$((ERRORS + 1))
else
    echo -e "${RED}✗${NC} Helm: Not installed"
    ERRORS=$((ERRORS + 1))
fi

# Check ArgoCD CLI
if command_exists argocd; then
    ARGOCD_VERSION=$(argocd version --client --short 2>/dev/null | awk '{print $2}' | sed 's/v//' || argocd version --client 2>/dev/null | grep 'argocd:' | awk '{print $2}' | sed 's/v//')
    check_version "ArgoCD CLI" "v$ARGOCD_VERSION" "v2.7.3" || ERRORS=$((ERRORS + 1))
else
    echo -e "${RED}✗${NC} ArgoCD CLI: Not installed"
    ERRORS=$((ERRORS + 1))
fi

echo ""
echo -e "${YELLOW}Checking project structure...${NC}"

# Check project directories
REQUIRED_DIRS=("services" "infra" "user" "config" "credentials" "templates" ".cursor/rules")
for dir in "${REQUIRED_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        echo -e "${GREEN}✓${NC} Directory: $dir"
    else
        echo -e "${RED}✗${NC} Directory: $dir (missing)"
        ERRORS=$((ERRORS + 1))
    fi
done

# Check key files
REQUIRED_FILES=("README.md" "user/sources.yaml" "user/preferences.yaml" "user/schedules.yaml")
for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} File: $file"
    else
        echo -e "${RED}✗${NC} File: $file (missing)"
        ERRORS=$((ERRORS + 1))
    fi
done

echo ""
echo -e "${YELLOW}Checking Go services...${NC}"

# Check API Gateway
if [ -d "services/api-gateway" ] && [ -f "services/api-gateway/go.mod" ]; then
    echo -e "${GREEN}✓${NC} API Gateway service structure"
    cd services/api-gateway
    if go build -o /tmp/api-gateway-test . >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} API Gateway builds successfully"
        rm -f /tmp/api-gateway-test
    else
        echo -e "${RED}✗${NC} API Gateway build failed"
        ERRORS=$((ERRORS + 1))
    fi
    cd ../..
else
    echo -e "${RED}✗${NC} API Gateway service structure"
    ERRORS=$((ERRORS + 1))
fi

# Check WebSocket service
if [ -d "services/ws" ] && [ -f "services/ws/go.mod" ]; then
    echo -e "${GREEN}✓${NC} WebSocket service structure"
    cd services/ws
    if go build -o /tmp/ws-test . >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} WebSocket service builds successfully"
        rm -f /tmp/ws-test
    else
        echo -e "${RED}✗${NC} WebSocket service build failed"
        ERRORS=$((ERRORS + 1))
    fi
    cd ../..
else
    echo -e "${RED}✗${NC} WebSocket service structure"
    ERRORS=$((ERRORS + 1))
fi

echo ""
echo -e "${YELLOW}Running tests...${NC}"

# Test API Gateway
cd services/api-gateway
if go test . >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} API Gateway tests pass"
else
    echo -e "${RED}✗${NC} API Gateway tests fail"
    ERRORS=$((ERRORS + 1))
fi
cd ../..

# Test WebSocket service
cd services/ws
if go test . >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} WebSocket service tests pass"
else
    echo -e "${RED}✗${NC} WebSocket service tests fail"
    ERRORS=$((ERRORS + 1))
fi
cd ../..

echo ""
echo -e "${YELLOW}Validating Kubernetes manifests...${NC}"

# Validate Kubernetes manifests
if kubectl apply --dry-run=client -f infra/ -R >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} Kubernetes manifests are valid"
else
    echo -e "${RED}✗${NC} Kubernetes manifests validation failed"
    ERRORS=$((ERRORS + 1))
fi

echo ""
echo "================================"

if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}🎉 All checks passed! Your Selin setup is ready.${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Configure your secrets in infra/secrets.yaml"
    echo "2. Update NFS configuration in infra/nfs/pv.yaml"
    echo "3. Deploy to your Kubernetes cluster"
    exit 0
else
    echo -e "${RED}❌ Found $ERRORS issues that need to be resolved.${NC}"
    echo ""
    echo "To fix missing tools, run:"
    echo "./scripts/install-tools.sh"
    exit 1
fi
