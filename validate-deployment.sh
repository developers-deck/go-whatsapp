#!/bin/bash

# WhatsApp Multi-Device Deployment Validation Script
set -e

echo "🔍 Validating WhatsApp Multi-Device Deployment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
WARNINGS=0

# Function to print colored output
print_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASSED++))
}

print_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAILED++))
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((WARNINGS++))
}

# Test functions
test_docker() {
    print_test "Checking Docker installation..."
    if command -v docker &> /dev/null; then
        print_pass "Docker is installed"
    else
        print_fail "Docker is not installed"
        return 1
    fi
}

test_docker_compose() {
    print_test "Checking Docker Compose installation..."
    if command -v docker-compose &> /dev/null; then
        print_pass "Docker Compose is installed"
    else
        print_fail "Docker Compose is not installed"
        return 1
    fi
}

test_env_file() {
    print_test "Checking environment configuration..."
    if [ -f "src/.env" ]; then
        print_pass ".env file exists"
        
        # Check for default/insecure values
        if grep -q "CHANGE_THIS" src/.env; then
            print_warn "Found default values in .env file - please update for production"
        fi
        
        if grep -q "your_" src/.env; then
            print_warn "Found placeholder values in .env file - please update for production"
        fi
        
        # Check required variables
        required_vars=("DB_URI" "REDIS_URL" "JWT_SECRET" "ENCRYPTION_KEY")
        for var in "${required_vars[@]}"; do
            if grep -q "^${var}=" src/.env; then
                print_pass "$var is configured"
            else
                print_fail "$var is missing from .env"
            fi
        done
    else
        print_fail ".env file not found"
        return 1
    fi
}

test_services_running() {
    print_test "Checking if services are running..."
    
    services=("postgres" "redis" "whatsapp_api")
    for service in "${services[@]}"; do
        if docker-compose ps | grep -q "${service}.*Up"; then
            print_pass "$service is running"
        else
            print_fail "$service is not running"
        fi
    done
}

test_database_connection() {
    print_test "Testing database connection..."
    if docker-compose exec -T postgres pg_isready -U whatsapp_user -d whatsapp_main &> /dev/null; then
        print_pass "PostgreSQL connection successful"
    else
        print_fail "PostgreSQL connection failed"
    fi
}

test_redis_connection() {
    print_test "Testing Redis connection..."
    if docker-compose exec -T redis redis-cli ping &> /dev/null; then
        print_pass "Redis connection successful"
    else
        print_fail "Redis connection failed"
    fi
}

test_api_health() {
    print_test "Testing API health..."
    
    # Wait a bit for API to be ready
    sleep 5
    
    if curl -f -s http://localhost:3000/app/devices &> /dev/null; then
        print_pass "API health check passed"
    else
        print_warn "API health check failed - service might still be starting"
        
        # Try system overview endpoint
        if curl -f -s http://localhost:3000/system/overview &> /dev/null; then
            print_pass "System overview endpoint is accessible"
        else
            print_fail "System overview endpoint is not accessible"
        fi
    fi
}

test_web_dashboard() {
    print_test "Testing web dashboard..."
    
    if curl -f -s http://localhost:3000/ | grep -q "WhatsApp API"; then
        print_pass "Web dashboard is accessible"
    else
        print_fail "Web dashboard is not accessible"
    fi
}

test_nginx_proxy() {
    print_test "Testing Nginx proxy..."
    
    if docker-compose ps | grep -q "nginx.*Up"; then
        if curl -f -s http://localhost:80/health &> /dev/null; then
            print_pass "Nginx proxy is working"
        else
            print_warn "Nginx is running but health check failed"
        fi
    else
        print_warn "Nginx is not running (optional service)"
    fi
}

test_volumes() {
    print_test "Checking Docker volumes..."
    
    volumes=("postgres_data" "redis_data" "whatsapp_storage")
    for volume in "${volumes[@]}"; do
        if docker volume ls | grep -q "$volume"; then
            print_pass "$volume volume exists"
        else
            print_fail "$volume volume not found"
        fi
    done
}

test_logs() {
    print_test "Checking service logs for errors..."
    
    # Check for critical errors in logs
    if docker-compose logs whatsapp_api 2>&1 | grep -i "fatal\|panic\|error" | grep -v "connection refused" | head -5; then
        print_warn "Found errors in application logs (check above)"
    else
        print_pass "No critical errors found in logs"
    fi
}

test_ports() {
    print_test "Checking port accessibility..."
    
    ports=("3000" "5432" "6379")
    for port in "${ports[@]}"; do
        if netstat -tuln 2>/dev/null | grep -q ":$port " || ss -tuln 2>/dev/null | grep -q ":$port "; then
            print_pass "Port $port is accessible"
        else
            print_fail "Port $port is not accessible"
        fi
    done
}

# Run all tests
echo "Starting deployment validation..."
echo "================================"

test_docker
test_docker_compose
test_env_file
test_services_running
test_database_connection
test_redis_connection
test_volumes
test_ports
test_api_health
test_web_dashboard
test_nginx_proxy
test_logs

echo ""
echo "================================"
echo "Validation Summary:"
echo "================================"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${YELLOW}Warnings: $WARNINGS${NC}"
echo -e "${RED}Failed: $FAILED${NC}"

if [ $FAILED -eq 0 ]; then
    echo ""
    echo -e "${GREEN}🎉 Deployment validation completed successfully!${NC}"
    echo ""
    echo "Your WhatsApp Multi-Device system is ready for use:"
    echo "• Web Dashboard: http://localhost:3000"
    echo "• API Documentation: http://localhost:3000/docs"
    echo "• System Overview: http://localhost:3000/system/overview"
    echo ""
    echo "Next steps:"
    echo "1. Access the web dashboard to manage WhatsApp instances"
    echo "2. Configure webhooks for your integrations"
    echo "3. Set up SSL certificates for production use"
    echo "4. Configure monitoring and alerting"
    
    if [ $WARNINGS -gt 0 ]; then
        echo ""
        echo -e "${YELLOW}⚠️  Please review the warnings above before going to production.${NC}"
    fi
else
    echo ""
    echo -e "${RED}❌ Deployment validation failed with $FAILED errors.${NC}"
    echo "Please fix the issues above and run the validation again."
    exit 1
fi