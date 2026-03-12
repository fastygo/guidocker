# Go PaaS Implementation Guide

## Overview

This guide outlines the implementation of a containerized Platform as a Service (PaaS) built with pure Go using Quicktemplate for server-side rendering. The PaaS provides a web-based GUI for managing Docker Compose applications while maintaining complete isolation - the GUI can be removed after application deployment without affecting running containers.

## Core Principles

### Complete GUI Isolation
- **No Runtime Dependencies**: Applications run independently of the PaaS GUI
- **Removable Interface**: Delete the PaaS binary and database at any time
- **File-Based Persistence**: All configurations stored in `/opt/stacks/` directory
- **Clean Removal**: No traces left after GUI deletion

### Application Lifecycle Management
- **Docker Compose Integration**: Manages `docker-compose.yml` files in isolated directories
- **Automatic Discovery**: Detects new applications and existing containers
- **Deployment Tracking**: Maintains history of deployments and container instances
- **State Persistence**: Application metadata stored in BoltDB (removable with GUI)

## Dependencies and Libraries

### Core Go Standard Library
- `encoding/json` - JSON marshaling/unmarshaling for API responses
- `log` - Standard logging functionality
- `os` - Operating system interface for file operations
- `path/filepath` - File path manipulation utilities
- `context` - Context handling for request cancellation and timeouts

### External Dependencies
- `github.com/fasthttp/router` - HTTP routing for fasthttp server
- `github.com/fasthttp/session/v2` - Session management for user authentication
- `github.com/valyala/fasthttp` - High-performance HTTP server implementation
- `github.com/valyala/quicktemplate` - Fast template engine for server-side rendering
- `go.etcd.io/bbolt` - Embedded key-value database for metadata storage
- `github.com/docker/docker/client` - Docker API client for container management

### Why fasthttp Router with Nginx?

**Yes, fasthttp router is absolutely relevant and necessary** even when using Nginx as a reverse proxy. Here's why:

- **Nginx acts as external proxy**: Handles SSL termination, load balancing, and static file serving
- **Go application runs internal HTTP server**: fasthttp provides the web server inside your PaaS application
- **Router handles internal routing**: Directs requests to appropriate handlers within the Go application
- **Separation of concerns**: Nginx manages external traffic, Go handles application logic

```
Internet → Nginx (SSL, routing) → Go PaaS (fasthttp + router) → Docker API
```

The fasthttp router is essential for:
- API endpoint routing (`/api/apps`, `/api/deploy`)
- Web page routing (`/dashboard`, `/apps/{id}`)
- Static file serving for CSS/JS (if needed)
- Session-based authentication routing

## Architecture Components

### 1. Application Management System
- **Stack Directory**: `/opt/stacks/{app-id}/` contains `docker-compose.yml`
- **Metadata Storage**: BoltDB stores app configurations, domains, and settings
- **Container Discovery**: Scans Docker API for running containers by project labels
- **Status Monitoring**: Real-time status updates for all managed applications

### 2. Domain and SSL Management
- **Wildcard SSL**: Automatic Let's Encrypt certificates for `*.example.com`
- **GUI Domain Flexibility**: PaaS interface accessible via any domain (e.g., `goserver.com`, `dash.example.com`)
- **Application Domains**: Each app gets subdomain routing (e.g., `myapp.example.com`)
- **Nginx Integration**: Dynamic nginx configuration generation and reloading

### 3. Deployment History and Cleanup
- **Version Tracking**: Maintains deployment timestamps and container identifiers
- **Instance Management**: Tracks multiple container instances per application
- **Automated Cleanup**: CRON-based removal of old deployments
- **Retention Policy**: Configurable retention (keep last 1-3 deployments)

## Key Features Implementation

### GUI Interface (Quicktemplate)
- **Dashboard**: Overview of all applications with status indicators
- **Application Creator**: Forms for new app deployment with stack selection
- **Configuration Editor**: Edit compose files and nginx settings
- **SSL Manager**: Certificate status and renewal controls
- **Security Monitor**: View fail2ban status, banned IPs, and security events
- **CRON Settings**: Configure automated cleanup schedules

### Application Discovery
- **New App Detection**: Scans `/opt/stacks/` for new compose files
- **Container Matching**: Associates running containers with stored configurations
- **Status Synchronization**: Updates app status based on Docker API responses
- **Orphan Detection**: Identifies containers without PaaS management

### CRON Automation
- **Cleanup Scheduler**: Configurable CRON jobs for old deployment removal
- **Retention Settings**: GUI controls for keeping recent deployments
- **Safe Deletion**: Only removes containers not referenced in recent deployments
- **Log Retention**: Optional log archiving before cleanup

### Domain Architecture
```
GUI Domain: goserver.com (or dash.example.com)
├── PaaS Interface: https://goserver.com
└── API Endpoints: https://goserver.com/api/*

App Domains: *.example.com (wildcard SSL)
├── app1.example.com → container port 3000
├── api.example.com → container port 8000
└── blog.example.com → container port 4000
```

## Implementation Phases

### Phase 1: Core Infrastructure
1. Set up BoltDB for metadata storage
2. Implement Docker API client for container management
3. Create Quicktemplate components for UI
4. Establish nginx configuration management

### Phase 2: Application Management
1. Build compose file generation system
2. Implement app creation/deployment workflow
3. Add container status monitoring
4. Create application discovery mechanisms

### Phase 3: Advanced Features
1. Integrate Let's Encrypt for automatic SSL
2. Implement wildcard domain routing
3. Add CRON-based cleanup system
4. Build deployment history tracking

### Phase 4: GUI Polish
1. Design responsive dashboard interface
2. Add configuration editing capabilities
3. Implement real-time status updates
4. Create cleanup scheduling interface

## Data Management

### BoltDB Schema
- **Applications**: ID, name, domain, stack type, ports, SSL settings
- **Deployments**: App ID, timestamp, container IDs, compose hash
- **Settings**: CRON schedules, retention policies, SSL configurations
- **Domains**: Wildcard certificates, nginx configurations

### File System Structure
```
/opt/stacks/              # Application compose files
├── app1/
│   ├── docker-compose.yml
│   └── nginx.conf
└── app2/
    ├── docker-compose.yml
    └── nginx.conf

/etc/nginx/sites-enabled/  # Generated nginx configs
├── app1.conf
└── app2.conf

/var/lib/go-paas/         # BoltDB and temp files (removable)
└── data.db
```

## Security Considerations

### Container Isolation
- Each application runs in separate Docker networks
- No shared volumes between applications
- Resource limits per container
- Regular security updates for base images

### GUI Security
- HTTPS enforcement for all interfaces
- API authentication and authorization
- Input validation and sanitization
- Rate limiting for management operations
- **Fail2Ban Integration**: Monitor banned IPs and jail status
- **Security Events**: View brute force attempts and blocks

### Host Security (Always Enabled)
- **Fail2Ban**: Automatic IP banning for brute force attacks on SSH, web services
- **UFW Firewall**: Default deny policy with explicit allow rules
- **Automatic Updates**: Security patches applied automatically
- **Rootkit Detection**: Regular scanning with rkhunter and chkrootkit
- **Log Monitoring**: System log analysis with logwatch

### SSL Management
- Automatic certificate renewal via Let's Encrypt
- Wildcard certificate support
- Certificate monitoring and alerts
- Secure certificate storage

## Deployment and Maintenance

### Installation
1. Deploy PaaS binary to server
2. Initialize BoltDB and directories
3. Configure nginx for wildcard routing
4. Set up SSL certificates
5. Start PaaS service

### Backup Strategy
- **Hosting Provider**: Server-level backups handled by hosting provider
- **Future S3 Integration**: Planned cloud storage backups in upcoming versions
- **Current Approach**: Rely on hosting provider's backup solutions

### Log Persistence Strategy
- Application logs stored in `/opt/stacks/{app-id}/logs/`
- Automatic log rotation and archival
- GUI automatically loads existing logs on startup (even after 6 months)
- Logs persist independently of GUI lifecycle

### Removal Process
1. Stop PaaS service
2. Remove BoltDB (`/var/lib/go-paas/`)
3. Optionally remove nginx configurations
4. Applications continue running independently

## Logging and Diagnostics

### Application Logging
- **Persistent Storage**: Logs saved in `/opt/stacks/{app-id}/logs/` directory
- **GUI Integration**: Automatic log loading when GUI is restarted (even after months)
- **Log Rotation**: Automatic cleanup of old log files
- **Real-time Streaming**: Live log viewing during container execution

### Security Monitoring
- **Fail2Ban Status**: Active jails, banned IPs, and unban management
- **Firewall Rules**: Current UFW configuration and active rules
- **Security Scans**: Results from rkhunter and chkrootkit scans
- **System Logs**: Security-related system events and alerts

### System Diagnostics
- **Simple Monitoring**: Basic container status and resource usage
- **Error Tracking**: Deployment and runtime error logging
- **Health Checks**: Basic container health verification
- **Manual Inspection**: Direct access to container logs via Docker commands

### Fail2Ban Configuration
Fail2Ban automatically monitors log files and bans IPs that show malicious signs (too many password failures, etc.).

**Default Jails (Active by Default):**
- **sshd**: SSH brute force protection
- **nginx-http-auth**: HTTP basic auth brute force
- **nginx-noscript**: Script kiddie protection
- **nginx-badbots**: Bad bot blocking
- **nginx-noproxy**: Proxy abuse prevention

**GUI Integration:**
- View currently banned IPs
- See jail status and configuration
- Manually unban IPs if needed
- Monitor ban/unban events in logs

### Systemd Integration
**Systemd in Ubuntu is written in C** (not Python). It's the core init system and service manager for modern Linux distributions. For your PaaS:

```ini
# /etc/systemd/system/paas-gui.service
[Unit]
Description=Go PaaS GUI Service
After=network.target docker.service

[Service]
Type=simple
User=your-user
WorkingDirectory=/opt/paas
ExecStart=/opt/paas/paas-gui
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**Commands:**
```bash
sudo systemctl enable paas-gui  # Auto-start on boot
sudo systemctl start paas-gui   # Start service
sudo systemctl status paas-gui  # Check status
sudo systemctl stop paas-gui    # Stop service
```

## Additional Features from Previous Discussions

### Stack Templates
- **Predefined Templates**: Ready-to-use templates for React, PHP, Go, Ruby, Node.js stacks
- **Custom Template Creation**: GUI for creating and saving custom application templates
- **Template Variables**: Dynamic substitution of app name, domain, ports in templates
- **Template Marketplace**: Community templates with ratings and reviews

### Environment Variables Management
- **GUI Editor**: Visual environment variable editor with validation
- **Secret Management**: Encrypted storage for sensitive environment variables
- **Variable Inheritance**: Global and per-application environment variables
- **Runtime Updates**: Hot-reload of environment variables without container restart

### Container Logs and Monitoring
- **Real-time Logs**: Live log streaming from containers via WebSocket
- **Log Filtering**: Search and filter logs by time, level, or content
- **Log Persistence**: Optional log storage and rotation
- **Health Checks**: Automated health monitoring and alerts

### REST API for Integration
- **Full REST API**: Complete API for all PaaS operations
- **API Documentation**: Auto-generated Swagger/OpenAPI documentation
- **Webhook Support**: Event-driven notifications for deployments and failures
- **CLI Tool**: Command-line interface using the REST API

### Authentication and Authorization
- **User Management**: Multi-user support with role-based access
- **OAuth Integration**: Support for GitHub, GitLab, Google OAuth
- **Team Collaboration**: Application sharing and permission management
- **Audit Logs**: Complete audit trail of all user actions

### Advanced Container Features
- **Resource Limits**: CPU, memory, and disk quota management per application
- **Network Configuration**: Custom network setup and firewall rules
- **Volume Management**: Persistent storage and backup configurations
- **Container Exec**: Execute commands inside running containers via GUI

### File Upload and Deployment
- **Archive Deployment**: Upload ZIP/tar.gz files for deployment
- **Git Integration**: Direct deployment from Git repositories
- **Build Hooks**: Custom build scripts and pre/post-deployment hooks
- **Rollback System**: One-click rollback to previous versions

### Troubleshooting and Support
- **Health Diagnostics**: Automated system health checks and recommendations
- **Error Reporting**: Detailed error messages with suggested solutions
- **Debug Mode**: Verbose logging and debugging tools
- **Community Support**: Integration with forums and knowledge base

### Development and Testing
- **Local Development**: Docker-based local development environment
- **Testing Framework**: Automated testing for templates and deployments
- **CI/CD Integration**: Webhooks for automated testing and deployment
- **Staging Environment**: Separate staging environment for testing

### Performance Optimization
- **Response Caching**: Smart caching of API responses and static content
- **Database Optimization**: Query optimization and connection pooling
- **Async Processing**: Background job processing for heavy operations
- **CDN Integration**: Content delivery network support for assets

## Future Enhancements

### Advanced Features
- Blue-green deployments
- Rollback capabilities
- Custom domain support
- API rate limiting per application

### Scaling Considerations
- Multi-server deployment support
- Load balancer integration
- Database migration tools
- Backup and restore automation

## Infrastructure Preparation Guide

### Host System Requirements

#### Minimal Ubuntu Server Setup (Bare Metal/VM)
```bash
# Essential packages on host system (including security)
sudo apt update && sudo apt install -y \
  docker.io \
  docker-compose \
  nginx \
  certbot \
  python3-certbot-nginx \
  git \
  curl \
  wget \
  htop \
  vim \
  ufw \
  fail2ban \
  unattended-upgrades \
  logwatch \
  rkhunter \
  chkrootkit
```

**Why on host:**
- **Docker**: Required for running application containers
- **Nginx**: High-performance reverse proxy, SSL termination, better than containerized
- **Certbot**: Direct access to host certificates and nginx configs
- **System tools**: For monitoring, security, and maintenance

**Not needed on host:**
- **Go runtime**: ❌ Your PaaS GUI is compiled to static binary
- **Node.js**: ❌ If using Quicktemplate (no frontend build process)
- **Databases**: ❌ BoltDB is embedded in your binary

### Optional Containerized Components

#### PaaS GUI in Container (Optional)
```yaml
# docker-compose.yml for PaaS GUI
version: '3.8'
services:
  paas-gui:
    image: your-paas-gui:latest  # Your compiled Go binary
    ports:
      - "3000:3000"
    environment:
      - PAAS_ADMIN_USER=admin
      - PAAS_ADMIN_PASSWORD=securepass
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock  # Docker API access
      - /opt/stacks:/opt/stacks                    # Compose files
      - /etc/nginx/sites-enabled:/etc/nginx/sites-enabled  # Nginx configs
      - /etc/letsencrypt:/etc/letsencrypt          # SSL certificates
      - paas-data:/app/data                        # BoltDB storage
    restart: unless-stopped

volumes:
  paas-data:
```

**Pros of containerizing GUI:**
- ✅ Isolated dependencies
- ✅ Easy updates/replacement
- ✅ Resource limits
- ✅ Backup entire container state

**Cons:**
- ❌ Mounts required for Docker/Nginx access
- ❌ More complex networking
- ❌ Slightly higher resource usage

#### Certbot in Container (Alternative)
```yaml
version: '3.8'
services:
  certbot:
    image: certbot/certbot:latest
    volumes:
      - /etc/letsencrypt:/etc/letsencrypt
      - /var/lib/letsencrypt:/var/lib/letsencrypt
      - /etc/nginx/sites-enabled:/etc/nginx/sites-enabled:ro
    command: certonly --webroot --webroot-path=/var/www/html --email admin@example.com -d example.com -d *.example.com
```

### Development vs Production Setup

#### Development Environment
```bash
# On development machine
sudo apt install -y golang-go git docker.io docker-compose

# For building PaaS GUI
go mod download
go build -o paas-gui ./cmd/server

# For testing with local Docker
# (same as production but with test domains)
```

#### Production Server Preparation
```bash
#!/bin/bash
# production-server-setup.sh

# 1. Base Ubuntu setup
sudo apt update && sudo apt upgrade -y

# 2. Install required packages
sudo apt install -y docker.io docker-compose nginx certbot python3-certbot-nginx git curl htop ufw fail2ban

# 3. Configure Docker
sudo systemctl enable docker
sudo systemctl start docker

# 4. Configure firewall
sudo ufw allow ssh
sudo ufw allow 80
sudo ufw allow 443
sudo ufw --force enable

# 5. Create directories
sudo mkdir -p /opt/stacks
sudo mkdir -p /var/lib/go-paas

# 6. Configure security (fail2ban, ufw, auto-updates)
sudo systemctl enable fail2ban
sudo systemctl start fail2ban

# Enable automatic security updates
sudo dpkg-reconfigure --priority=low unattended-upgrades

# 7. Set permissions (adjust for your user)
sudo chown -R $USER:$USER /opt/stacks
sudo chown -R $USER:$USER /var/lib/go-paas
```

### Missing Components Check

**You didn't miss anything critical, but here are optimizations:**

1. **Docker Compose V2**: Modern `docker compose` (without hyphen)
2. **Systemd Services**: For auto-starting PaaS GUI and nginx
3. **Log Rotation**: Configure logrotate for application logs
4. **Monitoring**: Prometheus + Grafana for system monitoring
5. **Logs**: Persistent application logs accessible via GUI

### Recommended Production Stack

```
Host System (Ubuntu)
├── Docker Engine          # Container runtime
├── Docker Compose         # Multi-container apps
├── Nginx                  # Reverse proxy + SSL
├── Certbot                # Let's Encrypt automation
├── systemd                # Service management
├── ufw/fail2ban           # Security
└── Your PaaS binary       # Static Go executable

Containerized (Optional)
├── PaaS GUI Container     # If you want isolation
├── Certbot Container      # For SSL automation
└── Monitoring Stack       # Prometheus/Grafana
```

This setup gives you maximum flexibility: bare-metal performance for critical components, containerization where it makes sense for isolation.

## Complete Deployment Workflow Summary

### Initial Server Setup
1. **Clean Ubuntu VPS** with Docker, Docker Compose, Nginx, and Certbot installed (Go NOT required)
2. **Optional**: Docker Swarm for multi-node deployments
3. **Environment ready** for PaaS GUI deployment

### GUI Deployment and Initial Access
1. **Deploy PaaS binary** to server
2. **Start GUI service** - initially accessible at `IP:3000`
3. **Set admin credentials** via environment variables during build:
   ```bash
   export PAAS_ADMIN_USER=admin
   export PAAS_ADMIN_PASSWORD=yourpassword
   ```

### Domain Configuration
1. **DNS Setup**: Point `dash.example.com` to server IP
2. **GUI Access**: Now available at `http://dash.example.com`
3. **SSL Certificate**: Configure Let's Encrypt certificate through GUI
   - Enter email and domain information
   - GUI triggers Certbot automatically
   - `IP:3000` optionally redirects to domain

### Application Deployment
1. **Add Wildcard Domain**: Configure `*.example.com` in GUI
2. **SSL for Applications**: Automatic wildcard certificate via Let's Encrypt
3. **Deploy Applications**: Create and deploy apps through GUI
4. **Domain Routing**: Apps accessible at `app1.example.com`, `app2.example.com`, etc.

### GUI Removal (Optional)
1. **Stop PaaS service** when satisfied with deployments
2. **Remove GUI binary and BoltDB** (`/var/lib/go-paas/`)
3. **Remove GUI domain** (`dash.example.com`) and certificate
4. **Applications continue running** with wildcard SSL at `*.example.com`
5. **All configurations persist** in `/opt/stacks/` directories

### Result
- **Applications**: Fully operational and independent
- **SSL**: Wildcard certificate maintained for `*.example.com`
- **Nginx**: Continues routing traffic to application containers
- **Docker**: All containers and networks remain active
- **GUI**: Completely removed with no traces

This workflow provides a **temporary management interface** that bootstraps your application infrastructure and can be safely discarded once everything is configured and running.

## Key Benefits

### For Development
- **Rapid prototyping**: Quick setup of complex multi-stack architectures
- **Visual management**: Intuitive GUI for container orchestration
- **Template system**: Pre-built stacks for common technologies

### For Production
- **Zero overhead**: GUI removable after initial setup
- **Security**: Minimal attack surface after GUI removal
- **Performance**: Direct Nginx → Docker routing without middleware
- **Reliability**: Applications isolated from management interface failures

### For Operations
- **Cost effective**: Single VPS can host GUI + applications during setup
- **Flexible**: Move GUI to separate server if needed for multi-environment management
- **Maintainable**: Standard Docker Compose workflows for ongoing operations

This implementation provides a lightweight, isolated PaaS solution where the GUI serves as a management tool that can be completely removed while applications continue operating independently.
