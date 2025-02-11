#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Setting up development environment...${NC}"

# Check if Redis is running
redis-cli ping > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo -e "${RED}Redis is not running. Starting Redis...${NC}"
    brew services start redis
fi

# Start monitoring stack
echo -e "${GREEN}Starting monitoring stack...${NC}"
docker-compose -f docker-compose.monitoring.yml up -d

# Install frontend dependencies
echo -e "${GREEN}Setting up frontend...${NC}"
mkdir -p frontend
cd frontend
if [ ! -f package.json ]; then
    echo -e "${GREEN}Initializing new React project...${NC}"
    npx create-react-app . --template typescript
    npm install @apollo/client graphql @tanstack/react-query tailwindcss @headlessui/react
    npm install -D @storybook/react @testing-library/react @testing-library/jest-dom
fi

# Run tests
echo -e "${GREEN}Running backend tests...${NC}"
cd ..

# Run tests and capture output
TEST_OUTPUT=$(go test ./... -v 2>&1)
TEST_EXIT_CODE=$?

# Check for test failures
if [ $TEST_EXIT_CODE -ne 0 ]; then
    echo -e "${RED}Some tests failed. Here are the failures:${NC}"
    echo "$TEST_OUTPUT" | grep -A 1 "FAIL:"
    echo -e "\n${YELLOW}Please check TODO_DETAILED.md for instructions on fixing these tests.${NC}"
else
    echo -e "${GREEN}All tests passed!${NC}"
fi

echo -e "${GREEN}Setup complete! You can start implementing the TODO list.${NC}"
echo -e "${GREEN}Don't forget to:${NC}"
echo "1. Check TODO_DETAILED.md for today's tasks"
echo "2. Keep DEBUGGING.md updated"
echo "3. Run tests frequently"
echo "4. Commit small, focused changes"

# If tests failed, exit with error code
if [ $TEST_EXIT_CODE -ne 0 ]; then
    echo -e "${YELLOW}Warning: Some tests failed. Please fix them before proceeding with new features.${NC}"
fi 