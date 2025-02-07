# Monitoring Stack Setup Guide

## Overview
The monitoring stack consists of:
- **Prometheus**: Time-series database and monitoring system
- **Grafana**: Visualization and dashboarding
- **AlertManager**: Alert handling and routing
- **Node Exporter**: System metrics collection
- **Redis Exporter**: Redis metrics collection
- **Postgres Exporter**: PostgreSQL metrics collection

## Prerequisites
- Docker and Docker Compose installed
- Access to the application network
- Sufficient disk space for persistent storage

## Directory Structure
```
monitoring/
├── prometheus/
│   ├── prometheus.yml
│   └── rules/
│       └── alerts.yml
├── alertmanager/
│   ├── alertmanager.yml
│   └── templates/
│       └── slack.tmpl
├── grafana/
│   ├── provisioning/
│   │   ├── datasources/
│   │   │   └── prometheus.yml
│   │   └── dashboards/
│   │       └── provider.yml
│   └── dashboards/
│       └── metadatatool.json
```

## Installation Steps

1. **Create Required Directories**
```bash
mkdir -p monitoring/{prometheus,alertmanager,grafana/provisioning/{datasources,dashboards}}
```

2. **Configure Docker Compose**
File: `docker-compose.monitoring.yml`
```yaml
version: '3.8'
services:
  prometheus:
    image: prom/prometheus:v2.45.0
    volumes:
      - ./monitoring/prometheus:/etc/prometheus
      - prometheus_data:/prometheus
    ports:
      - "9090:9090"

  alertmanager:
    image: prom/alertmanager:v0.26.0
    volumes:
      - ./monitoring/alertmanager:/etc/alertmanager
    ports:
      - "9093:9093"

  grafana:
    image: grafana/grafana:10.1.0
    volumes:
      - ./monitoring/grafana:/etc/grafana
      - grafana_data:/var/lib/grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false

  node-exporter:
    image: prom/node-exporter:v1.6.1
    ports:
      - "9100:9100"

  redis-exporter:
    image: oliver006/redis_exporter:v1.54.0
    ports:
      - "9121:9121"

  postgres-exporter:
    image: prometheuscommunity/postgres-exporter:v0.13.2
    ports:
      - "9187:9187"
```

3. **Start the Stack**
```bash
docker-compose -f docker-compose.monitoring.yml up -d
```

## Component Configuration

### Prometheus
File: `prometheus.yml`
- Scrape interval: 15s
- Evaluation interval: 15s
- Targets:
  - Application metrics (`:8080/metrics`)
  - Node Exporter (`:9100`)
  - Redis Exporter (`:9121`)
  - Postgres Exporter (`:9187`)

### AlertManager
File: `alertmanager.yml`
- Route configuration for different severity levels
- Slack integration for notifications
- Grouping and inhibition rules
- Templates for alert formatting

### Grafana
- Default admin credentials: admin/admin
- Auto-provisioned datasources and dashboards
- Disabled sign-up
- Minimum refresh interval: 5s

## Security Considerations
1. **Network Security**
   - Services exposed only on necessary ports
   - External access restricted through network policies
   - TLS configuration for production

2. **Authentication**
   - Change default passwords
   - Configure SSO for production
   - API key rotation policy

3. **Authorization**
   - Role-based access control
   - Viewer/Editor/Admin roles
   - Dashboard permissions

## Maintenance

### Backup
1. **Volume Backup**
```bash
docker run --rm --volumes-from prometheus -v $(pwd):/backup alpine tar cvf /backup/prometheus-backup.tar /prometheus
docker run --rm --volumes-from grafana -v $(pwd):/backup alpine tar cvf /backup/grafana-backup.tar /var/lib/grafana
```

2. **Configuration Backup**
- Keep all configuration in version control
- Document changes in git commits
- Include dashboard JSON exports

### Scaling
1. **Storage**
   - Monitor disk usage
   - Configure retention policies
   - Consider remote storage for long-term metrics

2. **Performance**
   - Adjust scrape intervals
   - Configure recording rules
   - Optimize queries

### Troubleshooting
1. **Common Issues**
   - Check container logs: `docker-compose logs <service>`
   - Verify network connectivity
   - Validate configuration files

2. **Monitoring the Monitors**
   - Set up alerts for monitoring stack
   - Monitor resource usage
   - Check scrape targets in Prometheus

## Access URLs
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000
- AlertManager: http://localhost:9093

## Best Practices
1. **Configuration**
   - Use environment variables
   - Version control all configs
   - Document all changes

2. **Alerting**
   - Define clear severity levels
   - Set appropriate thresholds
   - Avoid alert fatigue

3. **Dashboard Design**
   - Consistent naming
   - Clear visualization
   - Useful legends and documentation 