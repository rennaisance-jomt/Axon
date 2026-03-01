package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rennaisance-jomt/axon/pkg/logger"
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

// Tracer provides basic distributed tracing
type Tracer struct {
	mu          sync.RWMutex
	serviceName string
	enabled     bool
	spans       []*Span
	maxSpans    int
}

// Metrics holds basic telemetry metrics
type Metrics struct {
	mu             sync.RWMutex
	requestCount   int64
	errorCount     int64
	activeSessions int64
	durations       []time.Duration
}

// NewTracer creates a new telemetry tracer
func NewTracer(cfg *Config) (*Tracer, error) {
	if cfg == nil {
		cfg = &Config{
			Enabled:     true,
			ServiceName: "axon",
			SampleRate:  1.0,
		}
	}

	return &Tracer{
		serviceName: cfg.ServiceName,
		enabled:     cfg.Enabled,
		spans:       make([]*Span, 0),
		maxSpans:    1000,
	}, nil
}

// StartSpan starts a new span
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	if !t.enabled {
		return ctx, nil
	}

	span := &Span{
		Name:      name,
		StartTime: time.Now(),
		Attributes: make(map[string]string),
	}

	// Generate simple span ID
	span.SpanID = fmt.Sprintf("%016x", time.Now().UnixNano())
	
	return context.WithValue(ctx, "span", span), span
}

// EndSpan ends a span
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

// RecordRequest records a request metric (simplified)
func (t *Tracer) RecordRequest(ctx context.Context, name string, duration time.Duration, success bool) {
	// In a full implementation, this would send to OTLP
	// For now, we just track locally
}

// RecordSessionChange records session count change
func (t *Tracer) RecordSessionChange(ctx context.Context, delta int64) {
	// In a full implementation, this would send to OTLP
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
