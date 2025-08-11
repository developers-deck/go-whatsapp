# Deployment & Testing Guide

## 🚀 Complete WhatsApp Web Multi-Device System

### 📋 Pre-Deployment Checklist

#### 1. System Requirements
- [ ] Go 1.19+ installed
- [ ] PostgreSQL database configured
- [ ] Redis server (optional, for caching)
- [ ] Cloud storage credentials (S3/GCS) for backups
- [ ] Port 8080 available (or configured port)

#### 2. Environment Configuration
- [ ] Copy `.env.example` to `.env`
- [ ] Configure database connection
- [ ] Set Redis connection (if using)
- [ ] Configure cloud storage credentials
- [ ] Set webhook secrets and URLs

#### 3. Build & Dependencies
```bash
cd src
go mod tidy
go build -o whatsapp-api main.go
```

### 🧪 Testing Procedures

#### 1. Basic System Test
```bash
# Start the server
./whatsapp-api rest

# Check if server starts successfully
curl http://localhost:8080/
```

#### 2. Web Dashboard Access
1. Open browser to `http://localhost:8080`
2. Verify splash screen loads
3. Check all dashboard sections are visible:
   - [ ] System Overview
   - [ ] App (Login/Logout/Reconnect)
   - [ ] Send (Messages/Files/Media)
   - [ ] Message Management
   - [ ] Group Management
   - [ ] Newsletter
   - [ ] Account Management
   - [ ] Chat Management
   - [ ] Advanced Features

#### 3. Advanced Features Testing

##### System Dashboard
- [ ] System overview loads with metrics
- [ ] Auto-refresh functionality works
- [ ] Health indicators display correctly
- [ ] Recent activity shows mock data

##### Multi-Instance Manager
- [ ] Instance list loads
- [ ] Create instance modal opens
- [ ] Instance actions (start/stop/delete) work
- [ ] Status updates correctly

##### Process Monitor
- [ ] Process list displays
- [ ] System statistics show
- [ ] Auto-refresh toggles correctly
- [ ] Process actions available

##### Analytics Dashboard
- [ ] Date range picker works
- [ ] Statistics display correctly
- [ ] Report generation functions
- [ ] Download functionality works

##### Queue Manager
- [ ] Job list displays
- [ ] Queue statistics show
- [ ] Add job modal works
- [ ] Job actions (retry/cancel) function

##### Template Manager
- [ ] Template list loads
- [ ] Create/edit template works
- [ ] Template testing functions
- [ ] Variable management works

##### Webhook Manager
- [ ] Webhook list displays
- [ ] Create webhook modal works
- [ ] Test webhook functionality
- [ ] Delivery tracking shows

##### Backup Manager
- [ ] Backup list loads
- [ ] Configuration modal works
- [ ] Backup creation functions
- [ ] Download/restore options work

##### Cache Manager
- [ ] Cache statistics display
- [ ] Key management works
- [ ] Add/delete keys function
- [ ] Flush cache works

##### File Manager
- [ ] File list displays
- [ ] Upload functionality works
- [ ] File preview functions
- [ ] Download/delete works

### 🔧 API Endpoint Testing

#### Core WhatsApp Endpoints
```bash
# Test device status
curl http://localhost:8080/app/devices

# Test login (will return QR code)
curl http://localhost:8080/app/login

# Test logout
curl -X POST http://localhost:8080/app/logout
```

#### Advanced Feature Endpoints
```bash
# System overview
curl http://localhost:8080/system/overview

# Multi-instance
curl http://localhost:8080/multiinstance/list

# Process monitoring
curl http://localhost:8080/monitor/processes

# Analytics
curl http://localhost:8080/analytics/stats

# Queue management
curl http://localhost:8080/queue/jobs

# Templates
curl http://localhost:8080/templates/list

# Webhooks
curl http://localhost:8080/webhook/list

# Backups
curl http://localhost:8080/backup/list

# Cache
curl http://localhost:8080/cache/stats

# File management
curl http://localhost:8080/filemanager/list
```

### 🌐 WebSocket Testing

#### WebSocket Connection
```javascript
// Test WebSocket connection
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = function() {
    console.log('WebSocket connected');
    // Test device fetch
    ws.send(JSON.stringify({
        "code": "FETCH_DEVICES",
        "message": "List device"
    }));
};

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('Received:', message);
};
```

### 🐛 Troubleshooting

#### Common Issues

1. **Server Won't Start**
   - Check port availability
   - Verify Go version compatibility
   - Check environment variables

2. **Database Connection Issues**
   - Verify PostgreSQL is running
   - Check connection string format
   - Ensure database exists

3. **Web Dashboard Not Loading**
   - Check browser console for errors
   - Verify static file serving
   - Check network connectivity

4. **WebSocket Connection Failed**
   - Check firewall settings
   - Verify WebSocket endpoint
   - Check browser WebSocket support

5. **API Endpoints Return 404**
   - Verify REST handler initialization
   - Check route registration
   - Confirm base path configuration

#### Debug Mode
```bash
# Enable debug logging
export APP_DEBUG=true
./whatsapp-api rest
```

### 📊 Performance Testing

#### Load Testing
```bash
# Install hey (HTTP load testing tool)
go install github.com/rakyll/hey@latest

# Test system overview endpoint
hey -n 100 -c 10 http://localhost:8080/system/overview

# Test device list endpoint
hey -n 100 -c 10 http://localhost:8080/app/devices
```

#### Memory Usage Monitoring
```bash
# Monitor memory usage
top -p $(pgrep whatsapp-api)

# Or use htop for better visualization
htop -p $(pgrep whatsapp-api)
```

### 🔒 Security Testing

#### Authentication Testing
```bash
# Test without authentication (should fail if enabled)
curl http://localhost:8080/system/overview

# Test with basic auth (if configured)
curl -u username:password http://localhost:8080/system/overview
```

#### CORS Testing
```bash
# Test CORS headers
curl -H "Origin: http://example.com" \
     -H "Access-Control-Request-Method: GET" \
     -H "Access-Control-Request-Headers: X-Requested-With" \
     -X OPTIONS \
     http://localhost:8080/system/overview
```

### 📈 Production Deployment

#### 1. Build for Production
```bash
# Build optimized binary
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o whatsapp-api main.go
```

#### 2. Docker Deployment (Optional)
```dockerfile
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY whatsapp-api .
COPY .env .
EXPOSE 8080
CMD ["./whatsapp-api", "rest"]
```

#### 3. Systemd Service (Linux)
```ini
[Unit]
Description=WhatsApp API Service
After=network.target

[Service]
Type=simple
User=whatsapp
WorkingDirectory=/opt/whatsapp-api
ExecStart=/opt/whatsapp-api/whatsapp-api rest
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

#### 4. Nginx Reverse Proxy
```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### ✅ Final Verification

After deployment, verify:
- [ ] All 18 features are accessible
- [ ] Web dashboard loads completely
- [ ] All API endpoints respond correctly
- [ ] WebSocket connections work
- [ ] File uploads/downloads function
- [ ] Database operations succeed
- [ ] Cache operations work (if Redis configured)
- [ ] Backup operations function (if cloud storage configured)
- [ ] System monitoring displays real data
- [ ] All advanced features are operational

### 🎉 Success Criteria

The deployment is successful when:
1. ✅ Server starts without errors
2. ✅ Web dashboard loads and displays all sections
3. ✅ All 18 advanced features are accessible
4. ✅ API endpoints return expected responses
5. ✅ WebSocket connections establish successfully
6. ✅ File operations work correctly
7. ✅ System monitoring shows real metrics
8. ✅ All management interfaces function properly

## 🚀 Your WhatsApp Web Multi-Device System is Ready for Production!

The system now provides a complete enterprise-grade WhatsApp API platform with comprehensive web-based management capabilities.