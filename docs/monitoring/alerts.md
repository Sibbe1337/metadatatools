# Alert Rules Documentation

## Overview
This document details the alert rules configured in our monitoring stack. These rules are defined in Prometheus and handled by AlertManager.

## Alert Severity Levels

### Critical
- Immediate action required
- Service is down or severely impacted
- Business impact is significant
- Page on-call engineer 24/7

### Warning
- Action required within business hours
- Service degradation
- Some business impact
- Notify during business hours

### Info
- No immediate action required
- Potential issues to watch
- No direct business impact
- Collect for reporting

## Alert Rules Configuration

### Application Health

1. **High Error Rate**
```yaml
alert: HighErrorRate
expr: |
  sum(rate(http_requests_total{status=~"5.."}[5m])) 
  / 
  sum(rate(http_requests_total[5m])) > 0.05
for: 5m
labels:
  severity: critical
annotations:
  summary: High HTTP error rate (> 5%)
  description: "Error rate is {{ $value | humanizePercentage }} over last 5m"
```

2. **Slow Response Time**
```yaml
alert: SlowResponseTime
expr: |
  histogram_quantile(0.95, 
    rate(http_request_duration_seconds_bucket[5m])
  ) > 1.0
for: 5m
labels:
  severity: warning
annotations:
  summary: Slow response time (95th percentile > 1s)
  description: "95th percentile latency is {{ $value }}s"
```

### Database Performance

1. **High Connection Usage**
```yaml
alert: HighDBConnections
expr: database_connections{state="active"} > 80
for: 5m
labels:
  severity: warning
annotations:
  summary: High database connection count
  description: "{{ $value }} active connections"
```

2. **Slow Queries**
```yaml
alert: SlowDatabaseQueries
expr: |
  rate(database_query_duration_seconds_sum[5m]) 
  / 
  rate(database_query_duration_seconds_count[5m]) > 1
for: 5m
labels:
  severity: warning
annotations:
  summary: Slow database queries
  description: "Average query time is {{ $value }}s"
```

### Cache Performance

1. **Low Cache Hit Rate**
```yaml
alert: LowCacheHitRate
expr: |
  sum(rate(cache_hits_total[5m])) 
  / 
  (sum(rate(cache_hits_total[5m])) + sum(rate(cache_misses_total[5m]))) 
  < 0.7
for: 10m
labels:
  severity: warning
annotations:
  summary: Low cache hit rate
  description: "Cache hit rate is {{ $value | humanizePercentage }}"
```

### AI Service Performance

1. **High AI Latency**
```yaml
alert: HighAILatency
expr: |
  rate(ai_request_duration_seconds_sum[5m]) 
  / 
  rate(ai_request_duration_seconds_count[5m]) > 5
for: 5m
labels:
  severity: warning
annotations:
  summary: High AI service latency
  description: "Average AI request time is {{ $value }}s"
```

2. **Low AI Confidence**
```yaml
alert: LowAIConfidence
expr: avg(ai_confidence) < 0.6
for: 15m
labels:
  severity: warning
annotations:
  summary: Low AI confidence scores
  description: "Average confidence is {{ $value | humanizePercentage }}"
```

### Track Processing

1. **High Processing Failure Rate**
```yaml
alert: HighProcessingFailureRate
expr: |
  sum(rate(tracks_processed_total{status="error"}[5m])) 
  / 
  sum(rate(tracks_processed_total[5m])) > 0.1
for: 5m
labels:
  severity: critical
annotations:
  summary: High track processing failure rate
  description: "Failure rate is {{ $value | humanizePercentage }}"
```

2. **Low Processing Rate**
```yaml
alert: LowProcessingRate
expr: rate(tracks_processed_total[5m]) < 1
for: 15m
labels:
  severity: warning
annotations:
  summary: Low track processing rate
  description: "Processing {{ $value }} tracks per second"
```

### System Resources

1. **High Memory Usage**
```yaml
alert: HighMemoryUsage
expr: |
  (node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes)
  /
  node_memory_MemTotal_bytes > 0.85
for: 5m
labels:
  severity: warning
annotations:
  summary: High memory usage
  description: "Memory usage is {{ $value | humanizePercentage }}"
```

2. **High CPU Usage**
```yaml
alert: HighCPUUsage
expr: |
  100 - (avg by(instance)(rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
for: 5m
labels:
  severity: warning
annotations:
  summary: High CPU usage
  description: "CPU usage is {{ $value }}%"
```

## Alert Routing

### Slack Integration
```yaml
receivers:
- name: 'slack-critical'
  slack_configs:
  - channel: '#alerts-critical'
    username: 'Prometheus'
    icon_emoji: ':fire:'
    title: '{{ .GroupLabels.alertname }}'
    text: "{{ range .Alerts }}{{ .Annotations.description }}\n{{ end }}"

- name: 'slack-warnings'
  slack_configs:
  - channel: '#alerts-warnings'
    username: 'Prometheus'
    icon_emoji: ':warning:'
    title: '{{ .GroupLabels.alertname }}'
    text: "{{ range .Alerts }}{{ .Annotations.description }}\n{{ end }}"
```

## Alert Grouping
```yaml
route:
  group_by: ['alertname', 'severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  receiver: 'slack-warnings'
  routes:
  - match:
      severity: critical
    receiver: 'slack-critical'
    repeat_interval: 1h
```

## Maintenance Windows
```yaml
time_intervals:
- name: business-hours
  time_intervals:
  - weekdays: ['monday:friday']
    times:
    - start_time: 09:00
      end_time: 17:00
```

## Best Practices

1. **Alert Design**
   - Clear, actionable alerts
   - Appropriate thresholds
   - Meaningful descriptions
   - Proper severity levels

2. **Alert Management**
   - Regular review of alert effectiveness
   - Tuning of thresholds based on patterns
   - Documentation of resolution steps
   - Runbooks for common issues

3. **Notification Channels**
   - Multiple notification methods
   - Escalation paths
   - On-call rotation integration
   - Alert acknowledgment tracking 