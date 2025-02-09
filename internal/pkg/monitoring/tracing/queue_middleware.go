package tracing

import (
	"context"
	"metadatatool/internal/pkg/domain"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// QueueTracingMiddleware wraps a QueueService with OpenTelemetry tracing
type QueueTracingMiddleware struct {
	next   domain.QueueService
	tracer trace.Tracer
}

// NewQueueTracingMiddleware creates a new tracing middleware for queue operations
func NewQueueTracingMiddleware(next domain.QueueService, tracer trace.Tracer) *QueueTracingMiddleware {
	return &QueueTracingMiddleware{
		next:   next,
		tracer: tracer,
	}
}

// Publish traces the publish operation
func (m *QueueTracingMiddleware) Publish(ctx context.Context, topic string, message *domain.Message, priority domain.QueuePriority) error {
	ctx, span := m.tracer.Start(ctx, "QueueService.Publish",
		trace.WithAttributes(
			attribute.String("topic", topic),
			attribute.String("message.id", message.ID),
			attribute.String("message.type", message.Type),
			attribute.String("priority", strconv.Itoa(int(priority))),
		),
	)
	defer span.End()

	err := m.next.Publish(ctx, topic, message, priority)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return err
}

// Subscribe traces the subscribe operation
func (m *QueueTracingMiddleware) Subscribe(ctx context.Context, topic string, handler domain.MessageHandler) error {
	ctx, span := m.tracer.Start(ctx, "QueueService.Subscribe",
		trace.WithAttributes(
			attribute.String("topic", topic),
		),
	)
	defer span.End()

	// Wrap the handler to add tracing
	tracedHandler := func(ctx context.Context, msg *domain.Message) error {
		handlerCtx, handlerSpan := m.tracer.Start(ctx, "QueueService.MessageHandler",
			trace.WithAttributes(
				attribute.String("message.id", msg.ID),
				attribute.String("message.type", msg.Type),
			),
		)
		defer handlerSpan.End()

		err := handler(handlerCtx, msg)
		if err != nil {
			handlerSpan.SetStatus(codes.Error, err.Error())
			handlerSpan.RecordError(err)
		}
		return err
	}

	err := m.next.Subscribe(ctx, topic, tracedHandler)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return err
}

// HandleDeadLetter traces the dead letter handling operation
func (m *QueueTracingMiddleware) HandleDeadLetter(ctx context.Context, topic string, message *domain.Message) error {
	ctx, span := m.tracer.Start(ctx, "QueueService.HandleDeadLetter",
		trace.WithAttributes(
			attribute.String("topic", topic),
			attribute.String("message.id", message.ID),
			attribute.String("message.type", message.Type),
			attribute.Int("retry_count", message.RetryCount),
		),
	)
	defer span.End()

	err := m.next.HandleDeadLetter(ctx, topic, message)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return err
}

// GetMessage traces the get message operation
func (m *QueueTracingMiddleware) GetMessage(ctx context.Context, id string) (*domain.Message, error) {
	ctx, span := m.tracer.Start(ctx, "QueueService.GetMessage",
		trace.WithAttributes(
			attribute.String("message.id", id),
		),
	)
	defer span.End()

	msg, err := m.next.GetMessage(ctx, id)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return msg, err
}

// RetryMessage traces the retry message operation
func (m *QueueTracingMiddleware) RetryMessage(ctx context.Context, id string) error {
	ctx, span := m.tracer.Start(ctx, "QueueService.RetryMessage",
		trace.WithAttributes(
			attribute.String("message.id", id),
		),
	)
	defer span.End()

	err := m.next.RetryMessage(ctx, id)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return err
}

// AckMessage traces the acknowledge message operation
func (m *QueueTracingMiddleware) AckMessage(ctx context.Context, id string) error {
	ctx, span := m.tracer.Start(ctx, "QueueService.AckMessage",
		trace.WithAttributes(
			attribute.String("message.id", id),
		),
	)
	defer span.End()

	err := m.next.AckMessage(ctx, id)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return err
}

// NackMessage traces the negative acknowledge message operation
func (m *QueueTracingMiddleware) NackMessage(ctx context.Context, id string, nackErr error) error {
	ctx, span := m.tracer.Start(ctx, "QueueService.NackMessage",
		trace.WithAttributes(
			attribute.String("message.id", id),
			attribute.String("error", nackErr.Error()),
		),
	)
	defer span.End()

	err := m.next.NackMessage(ctx, id, nackErr)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return err
}

// ListDeadLetters traces the list dead letters operation
func (m *QueueTracingMiddleware) ListDeadLetters(ctx context.Context, topic string, offset, limit int) ([]*domain.Message, error) {
	ctx, span := m.tracer.Start(ctx, "QueueService.ListDeadLetters",
		trace.WithAttributes(
			attribute.String("topic", topic),
			attribute.Int("offset", offset),
			attribute.Int("limit", limit),
		),
	)
	defer span.End()

	msgs, err := m.next.ListDeadLetters(ctx, topic, offset, limit)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return msgs, err
}

// ReplayDeadLetter traces the replay dead letter operation
func (m *QueueTracingMiddleware) ReplayDeadLetter(ctx context.Context, id string) error {
	ctx, span := m.tracer.Start(ctx, "QueueService.ReplayDeadLetter",
		trace.WithAttributes(
			attribute.String("message.id", id),
		),
	)
	defer span.End()

	err := m.next.ReplayDeadLetter(ctx, id)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return err
}

// PurgeDeadLetters traces the purge dead letters operation
func (m *QueueTracingMiddleware) PurgeDeadLetters(ctx context.Context, topic string) error {
	ctx, span := m.tracer.Start(ctx, "QueueService.PurgeDeadLetters",
		trace.WithAttributes(
			attribute.String("topic", topic),
		),
	)
	defer span.End()

	err := m.next.PurgeDeadLetters(ctx, topic)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return err
}

// Close traces the close operation
func (m *QueueTracingMiddleware) Close() error {
	return m.next.Close()
}
