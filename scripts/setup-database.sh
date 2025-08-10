#!/bin/bash

# Selin Database Setup Script
# This script sets up the Selin database schema on an existing PostgreSQL instance

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Selin Database Setup ===${NC}"
echo ""

# Default connection parameters
POSTGRES_HOST=${POSTGRES_HOST:-localhost}
POSTGRES_PORT=${POSTGRES_PORT:-5432}
POSTGRES_USER=${POSTGRES_USER:-postgres}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-changmeplease}
POSTGRES_DB=${POSTGRES_DB:-selin}

echo -e "${YELLOW}Attempting to connect to PostgreSQL...${NC}"
echo "Host: $POSTGRES_HOST"
echo "Port: $POSTGRES_PORT"
echo "User: $POSTGRES_USER"
echo "Database: $POSTGRES_DB"
echo ""

# Function to run SQL command
run_sql() {
    PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "$1" 2>/dev/null
}

# Function to run SQL file
run_sql_file() {
    PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f "$1" 2>/dev/null
}

# Test connection
if ! PGPASSWORD="$POSTGRES_PASSWORD" pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" >/dev/null 2>&1; then
    echo -e "${RED}✗ Cannot connect to PostgreSQL${NC}"
    echo ""
    echo "Please check:"
    echo "1. PostgreSQL is running on $POSTGRES_HOST:$POSTGRES_PORT"
    echo "2. User '$POSTGRES_USER' exists and has proper permissions"
    echo "3. Password is correct"
    echo ""
    echo "You can also set environment variables:"
    echo "export POSTGRES_HOST=your_host"
    echo "export POSTGRES_PORT=your_port"
    echo "export POSTGRES_USER=your_user"
    echo "export POSTGRES_PASSWORD=your_password"
    echo "export POSTGRES_DB=your_database"
    exit 1
fi

echo -e "${GREEN}✓ Connected to PostgreSQL${NC}"

# Check if database exists, if not create it
if ! PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -lqt | cut -d \| -f 1 | grep -qw "$POSTGRES_DB"; then
    echo "Creating database '$POSTGRES_DB'..."
    PGPASSWORD="$POSTGRES_PASSWORD" createdb -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" "$POSTGRES_DB"
    echo -e "${GREEN}✓ Database '$POSTGRES_DB' created${NC}"
else
    echo -e "${GREEN}✓ Database '$POSTGRES_DB' already exists${NC}"
fi

# Run the initialization script
echo ""
echo -e "${YELLOW}Setting up database schema...${NC}"

if [ -f "scripts/init-database.sql" ]; then
    if run_sql_file "scripts/init-database.sql"; then
        echo -e "${GREEN}✓ Database schema initialized successfully${NC}"
    else
        echo -e "${YELLOW}⚠️  Some schema initialization may have failed (might be normal if tables already exist)${NC}"
    fi
else
    echo -e "${RED}✗ Database initialization file not found: scripts/init-database.sql${NC}"
    exit 1
fi

# Verify tables were created
echo ""
echo -e "${YELLOW}Verifying database setup...${NC}"

TABLES=$(run_sql "SELECT tablename FROM pg_tables WHERE schemaname = 'public';" | grep -E "(content_metadata|learning_progress|query_history|data_sources)" | wc -l)

if [ "$TABLES" -ge 4 ]; then
    echo -e "${GREEN}✓ All required tables are present${NC}"
else
    echo -e "${YELLOW}⚠️  Expected 4 tables, found $TABLES${NC}"
fi

# Show table information
echo ""
echo -e "${BLUE}=== Database Information ===${NC}"
echo ""
echo "Tables created:"
run_sql "\dt" 2>/dev/null || echo "Could not list tables"

echo ""
echo "Sample data sources:"
run_sql "SELECT source_type, source_name, enabled FROM data_sources LIMIT 5;" 2>/dev/null || echo "Could not query data_sources"

echo ""
echo "Learning progress tracking:"
run_sql "SELECT topic, skill_level, progress_score FROM learning_progress;" 2>/dev/null || echo "Could not query learning_progress"

echo ""
echo -e "${GREEN}🎉 Selin database is ready!${NC}"
echo ""
echo -e "${BLUE}=== Connection Information ===${NC}"
echo "Host: $POSTGRES_HOST"
echo "Port: $POSTGRES_PORT"
echo "Database: $POSTGRES_DB"
echo "User: $POSTGRES_USER"
echo ""
echo "To connect manually:"
echo "PGPASSWORD='$POSTGRES_PASSWORD' psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB"
echo ""
echo "To query recent content:"
echo "SELECT * FROM recent_content LIMIT 5;"
echo ""
echo "To view learning analytics:"
echo "SELECT * FROM learning_analytics;"
