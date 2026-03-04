package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rennaisance-jomt/axon/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds telemetry configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	OtlpEndpoint   string
	Enabled        bool
	SampleRate     float64
}

// Span represents a tracing span
type Span struct {
	Name        string
	TraceID     string
	SpanID      string
	ParentID    string
	StartTime   time.Time
	EndTime     time.Time
	Attributes  map[string]string
	Status      string
	Error       error
}

// Tracer provides basic distributed tracing with OpenTelemetry support
type Tracer struct {
	mu              sync.RWMutex
	serviceName     string
	enabled         bool
	spans           []*Span
	maxSpans        int
	tracerProvider  *sdktrace.TracerProvider
	tracer          trace.Tracer
	propagator      propagation.TextMapPropagator
	otlpEndpoint    string
}

// Metrics holds basic telemetry metrics
type Metrics struct {
	mu             sync.RWMutex
	requestCount   int64
	errorCount     int64
	activeSessions int64
	durations       []time.Duration
}

// NewTracer creates a new telemetry tracer with OpenTelemetry
func NewTracer(cfg *Config) (*Tracer, error) {
	if cfg == nil {
		cfg = &Config{
			Enabled:     true,
			ServiceName: "axon",
			SampleRate:  1.0,
		}
	}

	t := &Tracer{
		serviceName:  cfg.ServiceName,
		enabled:      cfg.Enabled,
		spans:        make([]*Span, 0),
		maxSpans:     1000,
		otlpEndpoint: cfg.OtlpEndpoint,
		propagator:   propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	}

	// Initialize OpenTelemetry if endpoint is configured
	if cfg.OtlpEndpoint != "" && cfg.Enabled {
		if err := t.initOpenTelemetry(cfg); err != nil {
			logger.Warn("Failed to initialize OpenTelemetry: %v, falling back to simple tracing", err)
		}
	}

	return t, nil
}

// initOpenTelemetry initializes the OpenTelemetry tracer provider
func (t *Tracer) initOpenTelemetry(cfg *Config) error {
	ctx := context.Background()

	// Create OTLP exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OtlpEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service name
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider with sampling
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
	)

	t.tracerProvider = tp
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(t.propagator)

	// Create tracer
	t.tracer = tp.Tracer(cfg.ServiceName)

	logger.Success("OpenTelemetry initialized with endpoint: %s", cfg.OtlpEndpoint)
	return nil
}

// StartSpan starts a new span (simple mode)
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	if !t.enabled {
		return ctx, nil
	}

	// Use OpenTelemetry tracer if available
	if t.tracer != nil {
		var span trace.Span
		ctx, span = t.tracer.Start(ctx, name)
		return ctx, &Span{
			Name:      name,
			TraceID:   span.SpanContext().TraceID().String(),
			SpanID:    span.SpanContext().SpanID().String(),
			StartTime: time.Now(),
			Attributes: make(map[string]string),
		}
	}

	// Fallback to simple tracing
	span := &Span{
		Name:      name,
		StartTime: time.Now(),
		Attributes: make(map[string]string),
	}
	span.SpanID = fmt.Sprintf("%016x", time.Now().UnixNano())

	return context.WithValue(ctx, "span", span), span
}

// EndSpan ends a span (simple mode)
func (t *Tracer) EndSpan(span *Span, err error) {
	if span == nil || !t.enabled {
		return
	}

	span.EndTime = time.Now()
	if err != nil {
		span.Status = "error"
		span.Error = err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.spans = append(t.spans, span)
	if len(t.spans) > t.maxSpans {
		t.spans = t.spans[len(t.spans)-t.maxSpans:]
	}
}

// AddAttribute adds an attribute to a span
func (t *Tracer) AddAttribute(span *Span, key, value string) {
	if span == nil {
		return
	}
	span.Attributes[key] = value
}

// RecordRequest records a request metric
func (t *Tracer) RecordRequest(ctx context.Context, name string, duration time.Duration, success bool) {
	// In production, this would send to OTLP metrics
}

// RecordSessionChange records session count change
func (t *Tracer) RecordSessionChange(ctx context.Context, delta int64) {
	// In production, this would send to OTLP metrics
}

// GetTraceID returns the current trace ID
func (t *Tracer) GetTraceID(ctx context.Context) string {
	span, ok := ctx.Value("span").(*Span)
	if !ok || span == nil {
		return ""
	}
	return span.TraceID
}

// GetRecentSpans returns recent spans
func (t *Tracer) GetRecentSpans() []*Span {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]*Span, len(t.spans))
	copy(result, t.spans)
	return result
}

// SpanExporter interface for exporting spans
type SpanExporter interface {
	ExportSpans(spans []*Span) error
	Shutdown() error
}

// ConsoleExporter writes spans to console
type ConsoleExporter struct{}

// ExportSpans writes spans to console
func (c *ConsoleExporter) ExportSpans(spans []*Span) error {
	for _, span := range spans {
		logger.Info("TRACE | %s | %s | %v | %s",
			span.Name,
			span.SpanID,
			span.EndTime.Sub(span.StartTime),
			span.Status,
		)
	}
	return nil
}

// Shutdown closes the exporter
func (c *ConsoleExporter) Shutdown() error {
	return nil
}

// WithContext is a no-op for compatibility
func (t *Tracer) WithContext(ctx context.Context) context.Context {
	return ctx
}

// Shutdown shuts down the tracer provider
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.tracerProvider != nil {
		return t.tracerProvider.Shutdown(ctx)
	}
	return nil
}

// StartSessionSpan starts a span for a session operation
func (t *Tracer) StartSessionSpan(ctx context.Context, sessionID string) (context.Context, *Span) {
	ctx, span := t.StartSpan(ctx, "session."+sessionID)
	if span != nil {
		span.Attributes["session.id"] = sessionID
		span.Attributes["span.type"] = "session"
	}
	return ctx, span
}

// StartSnapshotSpan starts a span for a snapshot operation
func (t *Tracer) StartSnapshotSpan(ctx context.Context, sessionID string) (context.Context, *Span) {
	ctx, span := t.StartSpan(ctx, "snapshot")
	if span != nil {
		span.Attributes["session.id"] = sessionID
		span.Attributes["span.type"] = "snapshot"
	}
	return ctx, span
}

// StartActionSpan starts a span for an action operation
func (t *Tracer) StartActionSpan(ctx context.Context, sessionID, action, ref string) (context.Context, *Span) {
	ctx, span := t.StartSpan(ctx, "action."+action)
	if span != nil {
		span.Attributes["session.id"] = sessionID
		span.Attributes["action.type"] = action
		span.Attributes["action.ref"] = ref
		span.Attributes["span.type"] = "action"
	}
	return ctx, span
}
