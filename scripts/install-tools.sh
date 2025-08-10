#!/bin/bash

# Selin Project - Tool Installation Script
# This script installs kubectl, Helm, and ArgoCD CLI as specified in the implementation plan

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect architecture and OS
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

# Map architecture names
case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    armv7l)
        ARCH="arm"
        ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo -e "${GREEN}Installing tools for $OS/$ARCH${NC}"

# Create bin directory if it doesn't exist
mkdir -p ~/bin
export PATH="$HOME/bin:$PATH"

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to get latest release tag from GitHub
get_latest_release() {
    curl --silent "https://api.github.com/repos/$1/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

echo -e "${YELLOW}=== Installing kubectl v1.26.3 ===${NC}"
if command_exists kubectl; then
    KUBECTL_VERSION=$(kubectl version --client --short 2>/dev/null | grep 'Client Version' | awk '{print $3}' || echo "unknown")
    echo "kubectl is already installed (version: $KUBECTL_VERSION)"
    echo "Continuing with installation to ensure v1.26.3..."
fi

# Install kubectl v1.26.3
KUBECTL_VERSION="v1.26.3"
KUBECTL_URL="https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/${OS}/${ARCH}/kubectl"

echo "Downloading kubectl ${KUBECTL_VERSION} for ${OS}/${ARCH}..."
curl -LO "$KUBECTL_URL"
chmod +x kubectl

# Move to bin directory
if [ -w "/usr/local/bin" ]; then
    sudo mv kubectl /usr/local/bin/
    echo -e "${GREEN}kubectl installed to /usr/local/bin/${NC}"
else
    mv kubectl ~/bin/
    echo -e "${GREEN}kubectl installed to ~/bin/${NC}"
    echo -e "${YELLOW}Make sure ~/bin is in your PATH${NC}"
fi

echo -e "${YELLOW}=== Installing Helm v3.12.0 ===${NC}"
if command_exists helm; then
    HELM_VERSION=$(helm version --short 2>/dev/null | awk '{print $1}' | cut -d'+' -f1 || echo "unknown")
    echo "Helm is already installed (version: $HELM_VERSION)"
    echo "Continuing with installation to ensure v3.12.0..."
fi

# Install Helm v3.12.0
HELM_VERSION="v3.12.0"
HELM_URL="https://get.helm.sh/helm-${HELM_VERSION}-${OS}-${ARCH}.tar.gz"

echo "Downloading Helm ${HELM_VERSION} for ${OS}/${ARCH}..."
curl -LO "$HELM_URL"
tar -xzf "helm-${HELM_VERSION}-${OS}-${ARCH}.tar.gz"

# Move to bin directory
if [ -w "/usr/local/bin" ]; then
    sudo mv "${OS}-${ARCH}/helm" /usr/local/bin/
    echo -e "${GREEN}Helm installed to /usr/local/bin/${NC}"
else
    mv "${OS}-${ARCH}/helm" ~/bin/
    echo -e "${GREEN}Helm installed to ~/bin/${NC}"
fi

# Cleanup
rm -rf "${OS}-${ARCH}" "helm-${HELM_VERSION}-${OS}-${ARCH}.tar.gz"

echo -e "${YELLOW}=== Installing ArgoCD CLI v2.7.3 ===${NC}"
if command_exists argocd; then
    ARGOCD_VERSION=$(argocd version --client --short 2>/dev/null || echo "unknown")
    echo "ArgoCD CLI is already installed (version: $ARGOCD_VERSION)"
    echo "Continuing with installation to ensure v2.7.3..."
fi

# Install ArgoCD CLI v2.7.3
ARGOCD_VERSION="v2.7.3"
ARGOCD_URL="https://github.com/argoproj/argo-cd/releases/download/${ARGOCD_VERSION}/argocd-${OS}-${ARCH}"

echo "Downloading ArgoCD CLI ${ARGOCD_VERSION} for ${OS}/${ARCH}..."
curl -sSL -o argocd "$ARGOCD_URL"
chmod +x argocd

# Move to bin directory
if [ -w "/usr/local/bin" ]; then
    sudo mv argocd /usr/local/bin/
    echo -e "${GREEN}ArgoCD CLI installed to /usr/local/bin/${NC}"
else
    mv argocd ~/bin/
    echo -e "${GREEN}ArgoCD CLI installed to ~/bin/${NC}"
fi

echo -e "${YELLOW}=== Verifying Installations ===${NC}"

# Verify installations
echo "Verifying kubectl..."
if command_exists kubectl; then
    kubectl version --client --short 2>/dev/null || kubectl version --client 2>/dev/null
    echo -e "${GREEN}✓ kubectl is working${NC}"
else
    echo -e "${RED}✗ kubectl installation failed${NC}"
    exit 1
fi

echo "Verifying Helm..."
if command_exists helm; then
    helm version --short 2>/dev/null || helm version 2>/dev/null
    echo -e "${GREEN}✓ Helm is working${NC}"
else
    echo -e "${RED}✗ Helm installation failed${NC}"
    exit 1
fi

echo "Verifying ArgoCD CLI..."
if command_exists argocd; then
    argocd version --client --short 2>/dev/null || argocd version --client 2>/dev/null
    echo -e "${GREEN}✓ ArgoCD CLI is working${NC}"
else
    echo -e "${RED}✗ ArgoCD CLI installation failed${NC}"
    exit 1
fi

echo -e "${GREEN}=== All tools installed successfully! ===${NC}"
echo ""
echo "Installed versions:"
echo "- kubectl: $(kubectl version --client --short 2>/dev/null | grep 'Client Version' | awk '{print $3}' || kubectl version --client 2>/dev/null | head -1)"
echo "- Helm: $(helm version --short 2>/dev/null | awk '{print $1}' || helm version 2>/dev/null | head -1)"
echo "- ArgoCD: $(argocd version --client --short 2>/dev/null || argocd version --client 2>/dev/null | head -1)"
echo ""
echo -e "${YELLOW}Note: If tools were installed to ~/bin, make sure it's in your PATH:${NC}"
echo "export PATH=\"\$HOME/bin:\$PATH\""
echo ""
echo -e "${GREEN}You can now proceed with the Selin deployment!${NC}"
