# Docker Setup Guide for WhatsApp Web Multidevice API

This guide explains how to set up and run the WhatsApp Web Multidevice API using Docker and Docker Compose.

## 🚀 Quick Start

### Production Setup
```bash
# Clone the repository
git clone <repository-url>
cd go-whatsapp-web-multidevice-main

# Start all services
docker-compose up -d

# View logs
docker-compose logs -f whatsapp_api
```

### Development Setup
```bash
# Start with development overrides
docker-compose -f docker-compose.yml -f docker-compose.override.yml --profile dev up -d

# Start with hot reloading (using Air)
docker-compose -f docker-compose.yml -f docker-compose.override.yml --profile hot-reload up -d

# Start with development tools
docker-compose -f docker-compose.yml -f docker-compose.override.yml --profile dev --profile tools up -d
```

## 📋 Services Overview

### Core Services
- **PostgreSQL** (port 5432) - Main database and chat storage
- **Redis** (port 6379) - Caching and session management
- **WhatsApp API** (port 3000) - Main application server
- **Nginx** (ports 80, 443) - Reverse proxy and load balancer

### Optional Services (Monitoring Profile)
- **Prometheus** (port 9090) - Metrics collection
- **Grafana** (port 3001) - Monitoring dashboard

### Development Services (Dev Profile)
- **Air** - Hot reloading for Go development
- **Adminer** (port 8081) - Database management interface
- **Redis Commander** (port 8082) - Redis management interface

## 🔧 Configuration

### Environment Variables

The main configuration is handled through environment variables in the `docker-compose.yml` file:

#### Application Settings
```yaml
APP_PORT=3000
APP_DEBUG=false
APP_OS=Chrome
APP_BASIC_AUTH=admin:admin,user:password
APP_BASE_PATH=/app
```

#### Database Configuration
```yaml
DB_URI=postgres://whatsapp_user:whatsapp_secure_2024@postgres:5432/whatsapp_main?sslmode=disable
DB_KEYS_URI=postgres://whatsapp_user:whatsapp_secure_2024@postgres:5432/whatsapp_keys?sslmode=disable
CHAT_STORAGE_TYPE=postgres
CHAT_STORAGE_URI=postgres://whatsapp_user:whatsapp_secure_2024@postgres:5432/whatsapp_chat?sslmode=disable
```

#### Redis Configuration
```yaml
REDIS_ENABLED=true
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=redis_secure_2024
```

### Custom Configuration

To override default settings, create a `.env` file in the `src/` directory:

```bash
# Copy the example file
cp src/env.example src/.env

# Edit the configuration
nano src/.env
```

## 🗄️ Database Setup

### Initial Database Creation
```bash
# Start only the database services
docker-compose up -d postgres redis

# Wait for services to be healthy
docker-compose ps

# Run database initialization
docker-compose --profile setup up db_init
```

### Database Schemas
The application automatically creates the following databases:
- `whatsapp_main` - Main application data
- `whatsapp_keys` - WhatsApp session keys
- `whatsapp_chat` - Chat storage and messages

### Manual Database Access
```bash
# Connect to PostgreSQL
docker exec -it whatsapp_postgres psql -U whatsapp_user -d whatsapp_main

# Connect to Redis
docker exec -it whatsapp_redis redis-cli -a redis_secure_2024
```

## 🔍 Monitoring and Health Checks

### Health Check Endpoints
- **WhatsApp API**: `http://localhost:3000/app/devices`
- **PostgreSQL**: Internal health check using `pg_isready`
- **Redis**: Internal health check using `redis-cli incr ping`
- **Nginx**: `http://localhost/health` (requires health endpoint configuration)

### Monitoring Stack
```bash
# Start monitoring services
docker-compose --profile monitoring up -d

# Access Grafana
# URL: http://localhost:3001
# Username: admin
# Password: grafana_admin_2024
```

## 🛠️ Development Workflow

### Hot Reloading with Air
```bash
# Start development environment with hot reloading
docker-compose -f docker-compose.yml -f docker-compose.override.yml --profile hot-reload up -d

# View Air logs
docker-compose logs -f air
```

### Code Changes
The development setup mounts the source code directory, so changes are reflected immediately when using Air.

### Database Migrations
```bash
# Access the database
docker exec -it whatsapp_postgres psql -U whatsapp_user -d whatsapp_main

# Run migrations manually if needed
\i /docker-entrypoint-initdb.d/init-db.sql
```

## 📊 Performance Tuning

### Resource Limits
The WhatsApp API service has resource limits configured:
```yaml
deploy:
  resources:
    limits:
      memory: 1G
      cpus: '1.0'
    reservations:
      memory: 512M
      cpus: '0.5'
```

### Redis Optimization
```yaml
command: redis-server --requirepass redis_secure_2024 --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
```

## 🔒 Security Considerations

### Default Credentials
**⚠️ Change these in production!**

- **PostgreSQL**: `whatsapp_user` / `whatsapp_secure_2024`
- **Redis**: `redis_secure_2024`
- **Grafana**: `admin` / `grafana_admin_2024`
- **Basic Auth**: `admin:admin,user:password`

### Network Isolation
All services run on a dedicated Docker network (`whatsapp_network`) with subnet `172.20.0.0/16`.

### SSL/TLS
Configure SSL certificates in the `docker/ssl/` directory and update the Nginx configuration.

## 🚨 Troubleshooting

### Common Issues

#### Service Won't Start
```bash
# Check service status
docker-compose ps

# View service logs
docker-compose logs <service-name>

# Check health status
docker-compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Health}}"
```

#### Database Connection Issues
```bash
# Test database connectivity
docker exec -it whatsapp_postgres pg_isready -U whatsapp_user -d whatsapp_main

# Check database logs
docker-compose logs postgres
```

#### WebSocket Issues
The application now includes improved WebSocket handling:
- Automatic reconnection on abnormal closures
- Proper ping/pong mechanism
- Connection timeout management
- Better error logging

### Log Analysis
```bash
# View all logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f whatsapp_api

# Search logs for errors
docker-compose logs whatsapp_api | grep ERROR
```

## 📚 Additional Resources

### Useful Commands
```bash
# Restart a specific service
docker-compose restart whatsapp_api

# Scale services
docker-compose up -d --scale whatsapp_api=2

# View resource usage
docker stats

# Clean up volumes
docker-compose down -v
```

### File Locations
- **Application logs**: `./logs/`
- **WhatsApp storage**: `./storages/`
- **Static files**: `./statics/`
- **Database data**: Docker volumes
- **SSL certificates**: `./docker/ssl/`

### Support
For issues related to:
- **Docker setup**: Check this guide and Docker logs
- **Application**: Check application logs and WebSocket status
- **Database**: Verify connection strings and health checks
- **WebSocket**: Check the improved connection handling we've implemented

## 🔄 Updates and Maintenance

### Updating Services
```bash
# Pull latest images
docker-compose pull

# Restart services
docker-compose up -d

# Update specific service
docker-compose pull whatsapp_api
docker-compose up -d whatsapp_api
```

### Backup and Restore
```bash
# Backup database
docker exec whatsapp_postgres pg_dump -U whatsapp_user whatsapp_main > backup.sql

# Restore database
docker exec -i whatsapp_postgres psql -U whatsapp_user whatsapp_main < backup.sql
```

---

**Note**: This Docker setup has been updated with the latest improvements including:
- ✅ Fixed WebSocket connection issues
- ✅ Improved database connectivity
- ✅ Better health checks and monitoring
- ✅ Development-friendly overrides
- ✅ Resource management and optimization
- ✅ Enhanced security and network isolation
