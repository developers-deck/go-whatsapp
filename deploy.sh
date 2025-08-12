#!/bin/bash

# WhatsApp Web Multidevice API - Docker Deployment Script
# This script provides easy commands for managing the Docker setup

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
}

# Function to check if Docker Compose is available
check_docker_compose() {
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed. Please install Docker Compose and try again."
        exit 1
    fi
}

# Function to start services
start_services() {
    local profile=${1:-""}
    
    if [ -n "$profile" ]; then
        print_status "Starting services with profile: $profile"
        docker-compose --profile "$profile" up -d
    else
        print_status "Starting all services..."
        docker-compose up -d
    fi
    
    print_success "Services started successfully!"
}

# Function to stop services
stop_services() {
    print_status "Stopping all services..."
    docker-compose down
    print_success "Services stopped successfully!"
}

# Function to restart services
restart_services() {
    print_status "Restarting services..."
    docker-compose restart
    print_success "Services restarted successfully!"
}

# Function to view logs
view_logs() {
    local service=${1:-""}
    
    if [ -n "$service" ]; then
        print_status "Viewing logs for service: $service"
        docker-compose logs -f "$service"
    else
        print_status "Viewing all logs..."
        docker-compose logs -f
    fi
}

# Function to check service status
check_status() {
    print_status "Checking service status..."
    docker-compose ps
}

# Function to check service health
check_health() {
    print_status "Checking service health..."
    docker-compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Health}}"
}

# Function to scale services
scale_services() {
    local service=${1:-"whatsapp_api"}
    local replicas=${2:-"2"}
    
    print_status "Scaling $service to $replicas replicas..."
    docker-compose up -d --scale "$service=$replicas"
    print_success "Service scaled successfully!"
}

# Function to backup database
backup_database() {
    local backup_file="backup_$(date +%Y%m%d_%H%M%S).sql"
    
    print_status "Creating database backup: $backup_file"
    docker exec whatsapp_postgres pg_dump -U postgres whatsapp_db > "$backup_file"
    print_success "Database backup created: $backup_file"
}

# Function to restore database
restore_database() {
    local backup_file=${1:-""}
    
    if [ -z "$backup_file" ]; then
        print_error "Please specify a backup file to restore from."
        echo "Usage: $0 restore <backup_file>"
        exit 1
    fi
    
    if [ ! -f "$backup_file" ]; then
        print_error "Backup file not found: $backup_file"
        exit 1
    fi
    
    print_warning "This will overwrite the current database. Are you sure? (y/N)"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        print_status "Restoring database from: $backup_file"
        docker exec -i whatsapp_postgres psql -U postgres whatsapp_db < "$backup_file"
        print_success "Database restored successfully!"
    else
        print_status "Database restore cancelled."
    fi
}

# Function to clean up
cleanup() {
    print_warning "This will remove all containers, networks, and volumes. Are you sure? (y/N)"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        print_status "Cleaning up Docker environment..."
        docker-compose down -v --remove-orphans
        docker system prune -f
        print_success "Cleanup completed successfully!"
    else
        print_status "Cleanup cancelled."
    fi
}

# Function to show help
show_help() {
    echo "WhatsApp Web Multidevice API - Docker Deployment Script"
    echo ""
    echo "Usage: $0 <command> [options]"
    echo ""
    echo "Commands:"
    echo "  start [profile]     Start services (optionally with profile: dev, monitoring, setup)"
    echo "  stop                Stop all services"
    echo "  restart             Restart all services"
    echo "  logs [service]      View logs (all services or specific service)"
    echo "  status              Check service status"
    echo "  health              Check service health"
    echo "  scale <service> <n> Scale service to n replicas"
    echo "  backup              Create database backup"
    echo "  restore <file>      Restore database from backup file"
    echo "  cleanup             Remove all containers, networks, and volumes"
    echo "  help                Show this help message"
    echo ""
    echo "Profiles:"
    echo "  dev                 Development environment with hot reloading"
    echo "  monitoring          Monitoring stack (Prometheus + Grafana)"
    echo "  setup               Database initialization"
    echo ""
    echo "Examples:"
    echo "  $0 start                    # Start all services"
    echo "  $0 start dev               # Start development environment"
    echo "  $0 start monitoring        # Start monitoring stack"
    echo "  $0 logs whatsapp_api       # View API logs"
    echo "  $0 scale whatsapp_api 3    # Scale API to 3 replicas"
    echo "  $0 backup                  # Create database backup"
    echo "  $0 restore backup.sql      # Restore from backup"
}

# Main script logic
main() {
    # Check prerequisites
    check_docker
    check_docker_compose
    
    case "${1:-help}" in
        start)
            start_services "$2"
            ;;
        stop)
            stop_services
            ;;
        restart)
            restart_services
            ;;
        logs)
            view_logs "$2"
            ;;
        status)
            check_status
            ;;
        health)
            check_health
            ;;
        scale)
            scale_services "$2" "$3"
            ;;
        backup)
            backup_database
            ;;
        restore)
            restore_database "$2"
            ;;
        cleanup)
            cleanup
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "Unknown command: $1"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"