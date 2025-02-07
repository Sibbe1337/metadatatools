# Metadata Tool Monitoring Dashboard

## Overview
The Metadata Tool monitoring dashboard provides real-time visibility into the system's performance, health, and operational metrics. The dashboard is accessible at `http://localhost:3000` with default credentials (admin/admin).

## Dashboard Organization
The dashboard is organized into several logical sections, each focusing on specific aspects of the system:

### 1. Application Performance (Row 1)
- **HTTP Request Rate**
  - Metric: `rate(http_requests_total[5m])`
  - Description: Shows the rate of incoming HTTP requests over 5-minute windows
  - Purpose: Monitor traffic patterns and identify unusual spikes or drops

- **Active Users**
  - Metric: `active_users`
  - Description: Gauge showing current number of active users
  - Thresholds: Warning at 80 users (red)
  - Purpose: Real-time user activity monitoring

### 2. Response Times & Caching (Row 2)
- **Average Response Time**
  - Metric: `rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])`
  - Unit: Seconds
  - Description: Average response time for HTTP requests
  - Purpose: Track application responsiveness

- **Cache Hit Rate**
  - Metric: `sum(rate(cache_hits_total[5m])) / (sum(rate(cache_hits_total[5m])) + sum(rate(cache_misses_total[5m])))`
  - Unit: Percentage (0-1)
  - Description: Ratio of cache hits to total cache operations
  - Purpose: Monitor caching efficiency

### 3. Database Performance (Row 3)
- **Database Connections**
  - Metric: `database_connections{state='active'}`
  - Description: Number of active database connections
  - Purpose: Monitor connection pool utilization

- **Database Query Latency**
  - Metric: `rate(database_query_duration_seconds_sum[5m]) / rate(database_query_duration_seconds_count[5m])`
  - Unit: Seconds
  - Description: Average database query execution time
  - Purpose: Identify slow queries and performance issues

### 4. AI Service Performance (Row 4)
- **AI Service Latency**
  - Metric: `rate(ai_request_duration_seconds_sum[5m]) / rate(ai_request_duration_seconds_count[5m])`
  - Unit: Seconds
  - Description: Average response time for AI service requests
  - Purpose: Monitor AI service responsiveness

- **AI Confidence Score**
  - Metric: `avg(ai_confidence)`
  - Unit: Percentage (0-1)
  - Description: Average confidence score of AI predictions
  - Purpose: Monitor AI prediction quality

### 5. System Resources (Row 5)
- **System Resource Usage**
  - Metrics:
    - Memory: `100 * (node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes`
    - CPU: `100 * (1 - avg by (instance)(irate(node_cpu_seconds_total{mode='idle'}[5m])))`
  - Unit: Percentage
  - Description: System-level resource utilization
  - Purpose: Monitor infrastructure health

- **HTTP Error Rate**
  - Metric: `100 * sum(rate(http_requests_total{status=~'5..'}[5m])) / sum(rate(http_requests_total[5m]))`
  - Unit: Percentage
  - Description: Percentage of HTTP requests resulting in 5xx errors
  - Purpose: Track error rates and service stability

### 6. Track Processing (Row 6)
- **Track Processing Rate**
  - Metric: `rate(tracks_processed_total[5m])`
  - Description: Number of tracks processed per minute
  - Purpose: Monitor processing throughput

- **Track Processing Success Rate**
  - Metric: `sum(rate(tracks_processed_total{status='success'}[5m])) / sum(rate(tracks_processed_total[5m]))`
  - Unit: Percentage (0-1)
  - Description: Ratio of successfully processed tracks
  - Purpose: Monitor processing reliability

## Dashboard Settings
- **Refresh Rate**: 10 seconds
- **Time Range**: Last 1 hour by default
- **Style**: Dark theme
- **Tags**: metadatatool

## Panel Features
Each panel includes:
- Table-style legends with statistical calculations (mean, max, min where applicable)
- Tooltips for detailed point-in-time information
- Consistent color schemes
- Appropriate units and formatting
- 5-minute rate calculations for smoother graphs

## Using the Dashboard
1. **Time Range**: Adjust the time range using the picker in the top right
2. **Legends**: Click legend items to toggle visibility
3. **Tooltips**: Hover over graphs for detailed metrics
4. **Tables**: Sort legend tables by clicking column headers

## Alert Integration
The dashboard integrates with the following alert rules:
- High error rate alerts (>5% for 5 minutes)
- Slow response time alerts (>1s 95th percentile)
- Database connection alerts (>80 active connections)
- Cache performance alerts (<70% hit rate)
- AI service latency alerts (>5s 95th percentile)
- Track processing failure alerts (>10% failure rate)
- Low active user alerts (<10 during business hours)

## Maintenance
- Dashboard JSON is version controlled
- Panels can be exported/imported individually
- Dashboard is provisioned automatically via configuration
- Changes should be made through version control, not UI 