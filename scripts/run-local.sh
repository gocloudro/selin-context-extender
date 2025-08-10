#!/bin/bash

# Selin Project - Local Development Script
# This script runs the core services locally for development and testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Running Selin Locally for Development ===${NC}"
echo ""

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check if port is available
port_available() {
    ! lsof -i:$1 >/dev/null 2>&1
}

# Cleanup function
cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up processes...${NC}"
    
    # Kill background processes
    for pid in ${PIDS[@]}; do
        if kill -0 $pid 2>/dev/null; then
            kill $pid 2>/dev/null
            echo "Killed process $pid"
        fi
    done
    
    # Kill Docker containers if they were started
    if [ "$STARTED_POSTGRES" = "true" ]; then
        docker stop selin-postgres >/dev/null 2>&1 || true
        docker rm selin-postgres >/dev/null 2>&1 || true
        echo "Stopped PostgreSQL container"
    fi
    
    if [ "$STARTED_REDIS" = "true" ]; then
        docker stop selin-redis >/dev/null 2>&1 || true
        docker rm selin-redis >/dev/null 2>&1 || true
        echo "Stopped Redis container"
    fi
    
    echo -e "${GREEN}Cleanup complete${NC}"
    exit 0
}

# Set trap for cleanup
trap cleanup SIGINT SIGTERM EXIT

# Array to store PIDs
PIDS=()
STARTED_REDIS=false

echo -e "${YELLOW}Step 1: Pre-flight checks${NC}"

# Check if Go is available
if ! command_exists go; then
    echo -e "${RED}Go not found. Please install Go 1.22+${NC}"
    exit 1
fi

# Check if Docker is available for Redis
if ! command_exists docker; then
    echo -e "${YELLOW}Docker not found. Redis will need to be installed separately.${NC}"
    if ! command_exists redis-server; then
        echo -e "${RED}Neither Docker nor redis-server found. Please install one of them.${NC}"
        exit 1
    fi
fi

echo -e "${GREEN}✓ Prerequisites check passed${NC}"

echo ""
echo -e "${YELLOW}Step 2: Starting dependencies${NC}"

# Start PostgreSQL if not running on our dedicated port
STARTED_POSTGRES=false
POSTGRES_PORT=5433
if ! nc -z localhost $POSTGRES_PORT 2>/dev/null; then
    if command_exists docker; then
        echo "Starting PostgreSQL container on port $POSTGRES_PORT..."
        docker run -d --name selin-postgres \
            -e POSTGRES_DB=selin \
            -e POSTGRES_USER=postgres \
            -e POSTGRES_PASSWORD=changmeplease \
            -p $POSTGRES_PORT:5432 \
            postgres:15.3 >/dev/null
        STARTED_POSTGRES=true
        echo -e "${GREEN}✓ PostgreSQL container started on port $POSTGRES_PORT${NC}"
    else
        echo "Please start PostgreSQL manually on port $POSTGRES_PORT"
        read -p "Press Enter when PostgreSQL is running..."
    fi
else
    echo -e "${GREEN}✓ PostgreSQL is already running on port $POSTGRES_PORT${NC}"
fi

# Start Redis if not running
if ! nc -z localhost 6379 2>/dev/null; then
    if command_exists docker; then
        echo "Starting Redis container..."
        docker run -d --name selin-redis -p 6379:6379 redis:7.2 >/dev/null
        STARTED_REDIS=true
        echo -e "${GREEN}✓ Redis container started${NC}"
    else
        echo "Please start Redis manually: redis-server"
        read -p "Press Enter when Redis is running on port 6379..."
    fi
else
    echo -e "${GREEN}✓ Redis is already running${NC}"
fi

# Wait for PostgreSQL to be ready
if [ "$STARTED_POSTGRES" = "true" ]; then
    echo "Waiting for PostgreSQL to be ready..."
    timeout=60
    while ! nc -z localhost $POSTGRES_PORT 2>/dev/null && [ $timeout -gt 0 ]; do
        sleep 1
        timeout=$((timeout - 1))
    done

    if [ $timeout -eq 0 ]; then
        echo -e "${RED}✗ PostgreSQL failed to start${NC}"
        exit 1
    fi
    
    # Wait a bit more for PostgreSQL to fully initialize
    sleep 5
    
    # Initialize database schema
    echo "Initializing database schema..."
    if [ -f "scripts/init-database.sql" ]; then
        docker exec -i selin-postgres psql -U postgres -d selin < scripts/init-database.sql >/dev/null 2>&1 || {
            echo -e "${YELLOW}Database initialization failed, but continuing...${NC}"
        }
        echo -e "${GREEN}✓ Database schema initialized${NC}"
    fi
    
    echo -e "${GREEN}✓ PostgreSQL is ready${NC}"
fi

# Wait for Redis to be ready
echo "Waiting for Redis to be ready..."
timeout=30
while ! nc -z localhost 6379 2>/dev/null && [ $timeout -gt 0 ]; do
    sleep 1
    timeout=$((timeout - 1))
done

if [ $timeout -eq 0 ]; then
    echo -e "${RED}✗ Redis failed to start${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Redis is ready${NC}"

echo ""
echo -e "${YELLOW}Step 3: Building and testing services${NC}"

# Build and test API Gateway
echo "Building API Gateway..."
cd services/api-gateway
if ! go build -o /tmp/selin-api-gateway .; then
    echo -e "${RED}✗ API Gateway build failed${NC}"
    exit 1
fi

echo "Running API Gateway tests..."
if ! go test -v .; then
    echo -e "${RED}✗ API Gateway tests failed${NC}"
    exit 1
fi

echo -e "${GREEN}✓ API Gateway built and tested${NC}"

# Build and test WebSocket service
echo "Building WebSocket service..."
cd ../ws
if ! go build -o /tmp/selin-ws .; then
    echo -e "${RED}✗ WebSocket service build failed${NC}"
    exit 1
fi

echo "Running WebSocket service tests..."
if ! go test -v .; then
    echo -e "${RED}✗ WebSocket service tests failed${NC}"
    exit 1
fi

echo -e "${GREEN}✓ WebSocket service built and tested${NC}"

cd ../..

echo ""
echo -e "${YELLOW}Step 4: Starting services${NC}"

# Check if ports are available
if ! port_available 8080; then
    echo -e "${RED}Port 8080 is already in use${NC}"
    exit 1
fi

if ! port_available 8081; then
    echo -e "${RED}Port 8081 is already in use${NC}"
    exit 1
fi

# Start API Gateway
echo "Starting API Gateway on port 8080..."
cd services/api-gateway
export PORT=8080
export REDIS_URL="localhost:6379"
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="$POSTGRES_PORT"
export POSTGRES_DB="selin"
export POSTGRES_USER="postgres"
export POSTGRES_PASSWORD="changmeplease"
export LOG_LEVEL="info"
go run . &
API_PID=$!
PIDS+=($API_PID)
cd ../..

# Wait for API Gateway to start
sleep 3

# Test API Gateway
if curl -s http://localhost:8080/health >/dev/null; then
    echo -e "${GREEN}✓ API Gateway is running${NC}"
else
    echo -e "${RED}✗ API Gateway failed to start${NC}"
    exit 1
fi

# Start WebSocket service
echo "Starting WebSocket service on port 8081..."
cd services/ws
export PORT=8081
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="$POSTGRES_PORT"
export POSTGRES_DB="selin"
export POSTGRES_USER="postgres"
export POSTGRES_PASSWORD="changmeplease"
export LOG_LEVEL="info"
go run . &
WS_PID=$!
PIDS+=($WS_PID)
cd ../..

# Wait for WebSocket service to start
sleep 3

# Test WebSocket service
if curl -s http://localhost:8081/health >/dev/null; then
    echo -e "${GREEN}✓ WebSocket service is running${NC}"
else
    echo -e "${RED}✗ WebSocket service failed to start${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}🎉 Selin services are running locally!${NC}"
echo ""
echo -e "${BLUE}=== Service Information ===${NC}"
echo ""
echo -e "${YELLOW}API Gateway:${NC}"
echo "  Health: http://localhost:8080/health"
echo "  Ready:  http://localhost:8080/ready"
echo "  Metrics: http://localhost:8080/metrics"
echo "  Query:  POST http://localhost:8080/api/v1/query"
echo ""
echo -e "${YELLOW}WebSocket Service:${NC}"
echo "  Health: http://localhost:8081/health"
echo "  Ready:  http://localhost:8081/ready"
echo "  Metrics: http://localhost:8081/metrics"
echo "  WebSocket: ws://localhost:8081/ws"
echo ""
echo -e "${YELLOW}PostgreSQL:${NC}"
echo "  Host: localhost:$POSTGRES_PORT"
echo "  Database: selin"
echo "  User: postgres"
echo "  Password: changmeplease"
echo ""
echo -e "${YELLOW}Redis:${NC}"
echo "  Host: localhost:6379"
echo ""
echo -e "${BLUE}=== Testing the API ===${NC}"
echo ""

# Test the query endpoint
echo "Testing query endpoint..."
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Hello, Selin!"}')

echo "Query Response:"
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"
echo ""

echo -e "${BLUE}=== Development Commands ===${NC}"
echo ""
echo "Test query endpoint:"
echo 'curl -X POST http://localhost:8080/api/v1/query \'
echo '  -H "Content-Type: application/json" \'
echo '  -d '"'"'{"prompt": "How does Go handle concurrency?"}'"'"''
echo ""
echo "Test WebSocket (using wscat if installed):"
echo "wscat -c ws://localhost:8081/ws"
echo ""
echo "View metrics:"
echo "curl http://localhost:8080/metrics"
echo "curl http://localhost:8081/metrics"
echo ""
echo "Access PostgreSQL:"
echo "docker exec -it selin-postgres psql -U postgres -d selin"
echo "# OR: PGPASSWORD=changmeplease psql -h localhost -p $POSTGRES_PORT -U postgres -d selin"
echo ""
echo "Query recent content:"
echo "docker exec -it selin-postgres psql -U postgres -d selin -c 'SELECT * FROM recent_content LIMIT 5;'"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop all services${NC}"
echo ""

# Keep the script running
while true; do
    sleep 1
    
    # Check if processes are still running
    for pid in ${PIDS[@]}; do
        if ! kill -0 $pid 2>/dev/null; then
            echo -e "${RED}Process $pid stopped unexpectedly${NC}"
            exit 1
        fi
    done
done
