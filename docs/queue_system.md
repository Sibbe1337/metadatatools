# Queue System Documentation

## Overview

The Queue System is a Redis-based message processing system that implements a reliable pub/sub pattern with support for message retries, dead letter queues, and comprehensive monitoring. It follows clean architecture principles and provides a robust foundation for asynchronous task processing.

## Components

### Core Interfaces

#### Message
```go
type Message struct {
    ID            string
    Topic         string
    Data          []byte
    Status        MessageStatus
    RetryCount    int
    CreatedAt     time.Time
    UpdatedAt     time.Time
    DeadLetterAt  *time.Time
}
```

#### Publisher
```go
type Publisher interface {
    Publish(ctx context.Context, topic string, data []byte) error
    PublishBatch(ctx context.Context, topic string, messages [][]byte) error
}
```

#### Subscriber
```go
type Subscriber interface {
    Subscribe(ctx context.Context, topic string, handler MessageHandler) error
    Unsubscribe(ctx context.Context, topic string) error
}
```

### Message Lifecycle

1. **Publishing**: Messages are published to specific topics and stored in Redis with a "pending" status
2. **Processing**: Subscribers pick up messages and process them using registered handlers
3. **Completion**: Successfully processed messages are acknowledged and removed
4. **Retry**: Failed messages are retried based on configured policies
5. **Dead Letter**: Messages that exceed retry limits are moved to dead letter queues

## Configuration

### Queue Configuration
```go
type QueueConfig struct {
    RetryDelays        []time.Duration
    ProcessingTimeout  time.Duration
    BatchSize         int
    CleanupInterval   time.Duration
}
```

### Default Settings
- Retry Delays: [1m, 5m, 15m, 30m, 1h]
- Processing Timeout: 5 minutes
- Batch Size: 100
- Cleanup Interval: 1 hour

## Usage Examples

### Publishing Messages
```go
queue := queue.NewRedisQueue(redisClient, queueConfig)

// Single message
err := queue.Publish(ctx, "user.created", userData)

// Batch of messages
err := queue.PublishBatch(ctx, "user.created", userDataBatch)
```

### Subscribing to Topics
```go
handler := func(ctx context.Context, msg *domain.Message) error {
    // Process message
    return nil
}

err := queue.Subscribe(ctx, "user.created", handler)
```

### Dead Letter Queue Operations
```go
// List dead letter messages
messages, err := queue.ListDeadLetterMessages(ctx, "user.created")

// Replay dead letter message
err := queue.ReplayDeadLetterMessage(ctx, messageID)

// Purge dead letter queue
err := queue.PurgeDeadLetterQueue(ctx, "user.created")
```

## Monitoring & Metrics

### Prometheus Metrics

1. **Queue Operations**
   - `queue_operations_total{operation="publish|subscribe|ack|nack",status="success|failure"}`
   - Tracks total operations by type and status

2. **Message Processing**
   - `message_processing_duration_seconds{topic="..."}`
   - Histogram of message processing durations

3. **Message Status**
   - `message_status{status="pending|processing|completed|failed|retrying|dead_letter",topic="..."}`
   - Current count of messages by status

4. **Queue Size**
   - `queue_size{topic="..."}`
   - Current size of each queue

5. **Processing Errors**
   - `processing_errors_total{topic="...",error_type="timeout|validation|processing"}`
   - Total count of processing errors by type

### Grafana Dashboard

The queue system includes a pre-configured Grafana dashboard with panels for:
- Message throughput and latency
- Queue sizes and processing rates
- Error rates and types
- Dead letter queue metrics
- Resource utilization

## Error Handling

### Retry Policy
- Configurable retry delays with exponential backoff
- Maximum retry count per message
- Custom error types for different failure scenarios

### Error Types
1. `ErrMessageNotFound`: Message doesn't exist
2. `ErrTopicNotFound`: Topic doesn't exist
3. `ErrProcessingTimeout`: Message processing exceeded timeout
4. `ErrInvalidMessage`: Message validation failed
5. `ErrHandlerNotFound`: No handler registered for topic

## Best Practices

1. **Message Design**
   - Keep messages small and focused
   - Include necessary metadata for tracking
   - Use structured data formats (JSON/Protocol Buffers)

2. **Topic Naming**
   - Use hierarchical naming (e.g., "user.created", "order.updated")
   - Keep names consistent and descriptive
   - Document topic purposes and message formats

3. **Error Handling**
   - Implement idempotent message handlers
   - Log detailed error information
   - Monitor retry counts and dead letter queues

4. **Performance**
   - Use batch operations when possible
   - Configure appropriate timeouts
   - Monitor queue sizes and processing rates

5. **Monitoring**
   - Set up alerts for critical metrics
   - Review dead letter queues regularly
   - Monitor processing latency

## Testing

The queue system includes comprehensive tests:

1. **Unit Tests**
   - Message lifecycle
   - Retry mechanism
   - Dead letter handling
   - Configuration validation

2. **Integration Tests**
   - End-to-end message flow
   - Redis interaction
   - Concurrent processing
   - Error scenarios

3. **Performance Tests**
   - High throughput scenarios
   - Batch processing
   - Resource utilization

## Security Considerations

1. **Data Protection**
   - Messages are stored encrypted at rest
   - Sensitive data should be encrypted before publishing
   - Access control through Redis ACLs

2. **Resource Protection**
   - Rate limiting on publishing
   - Maximum message size limits
   - Queue size limits

3. **Monitoring**
   - Alerts for unusual activity
   - Audit logging of critical operations
   - Regular security reviews

## Troubleshooting

Common issues and solutions:

1. **Messages Not Being Processed**
   - Check subscriber health
   - Verify topic names
   - Review handler errors

2. **High Error Rates**
   - Monitor retry counts
   - Check handler timeouts
   - Review error patterns

3. **Performance Issues**
   - Monitor Redis metrics
   - Review batch sizes
   - Check network latency

4. **Dead Letter Queue Growth**
   - Review failed message patterns
   - Adjust retry policies
   - Implement manual review process 