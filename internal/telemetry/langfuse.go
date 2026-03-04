package telemetry

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/rennaisance-jomt/axon/internal/config"
	"github.com/rennaisance-jomt/axon/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// AxonEvent types that can be tracked
type AxonEventType string

const (
	// Session events
	EventSessionCreated   AxonEventType = "session.created"
	EventSessionEnded     AxonEventType = "session.ended"
	EventSessionActive   AxonEventType = "session.active"

	// Snapshot events
	EventSnapshotTaken    AxonEventType = "snapshot.taken"
	EventSnapshotTokens  AxonEventType = "snapshot.tokens"
	EventSnapshotReduced AxonEventType = "snapshot.reduced"

	// Action events
	EventActionClick     AxonEventType = "action.click"
	EventActionFill     AxonEventType = "action.fill"
	EventActionNavigate AxonEventType = "action.navigate"
	EventActionSuccess  AxonEventType = "action.success"
	EventActionFailed   AxonEventType = "action.failed"

	// Security events
	EventSSRFBlocked    AxonEventType = "security.ssrf_blocked"
	EventPromptInject   AxonEventType = "security.prompt_injection"

	// Network events
	EventAdsBlocked     AxonEventType = "network.ads_blocked"
	EventTrackersBlocked AxonEventType = "network.trackers_blocked"
	EventNetworkLatency AxonEventType = "network.latency"

	// Performance events
	EventMemoryUsage    AxonEventType = "performance.memory"
	EventCPUUsage       AxonEventType = "performance.cpu"

	// LLM events
	EventLLMTokenUsage  AxonEventType = "llm.token_usage"
)

// AxonEvent represents a tracked Axon event
type AxonEvent struct {
	Type        AxonEventType
	SessionID   string
	Timestamp   time.Time
	Duration    time.Duration
	Success     bool
	Error       string
	Metadata    map[string]interface{}
}

// AxonTelemetry provides comprehensive tracing for Axon browser
type AxonTelemetry struct {
	mu            sync.RWMutex
	config        *TelemetryConfig
	tracer        trace.Tracer
	provider      *sdktrace.TracerProvider
	events        []*AxonEvent
	maxEvents     int
	enabled       bool
	environment   string
}

// TelemetryConfig holds telemetry configuration
type TelemetryConfig struct {
	Enabled      bool
	Provider     string // "langfuse", "datadog", "grafana", "jaeger", "zipkin"
	Endpoint     string // OTLP endpoint URL
	PublicKey    string
	SecretKey    string
	SampleRate  float64
	Environment string
}

// NewAxonTelemetry creates new Axon telemetry
func NewAxonTelemetry(cfg *TelemetryConfig) (*AxonTelemetry, error) {
	if cfg == nil {
		cfg = &TelemetryConfig{
			Enabled:    false,
			SampleRate: 1.0,
		}
	}

	t := &AxonTelemetry{
		config:      cfg,
		events:     make([]*AxonEvent, 0),
		maxEvents:  10000,
		enabled:    cfg.Enabled,
		environment: cfg.Environment,
	}

	// Initialize OpenTelemetry if enabled and endpoint is configured
	if cfg.Enabled && cfg.Endpoint != "" {
		if err := t.initOTLP(); err != nil {
			logger.Warn("Failed to initialize OTLP: %v, continuing without remote tracing", err)
			// Don't fail - continue with local-only tracing
		}
	}

	// Create tracer
	t.tracer = otel.Tracer("axon")

	logger.Info("Axon Telemetry initialized: enabled=%v, provider=%s, endpoint=%s",
		cfg.Enabled, cfg.Provider, cfg.Endpoint)

	return t, nil
}

// initOTLP initializes OTLP exporter
func (t *AxonTelemetry) initOTLP() error {
	ctx := context.Background()

	// Determine if this is Langfuse cloud
	isLangfuse := t.config.Provider == "langfuse"

	var endpoint string
	var urlPath string

	// Build endpoint URL based on provider
	if isLangfuse {
		endpoint = t.config.Endpoint
		// If empty or default cloud URL, use the known good cloud endpoint
		if endpoint == "" || endpoint == "https://cloud.langfuse.com" || endpoint == "cloud.langfuse.com" {
			endpoint = "cloud.langfuse.com"
		}

		// Strip protocol if present
		if len(endpoint) > 8 && endpoint[:8] == "https://" {
			endpoint = endpoint[8: ]
		} else if len(endpoint) > 7 && endpoint[:7] == "http://" {
			endpoint = endpoint[7: ]
		}

		// Langfuse OTLP path is /api/public/otel/v1/traces
		urlPath = "/api/public/otel/v1/traces"
		logger.Info("Using Langfuse OTLP HTTP endpoint: %s%s", endpoint, urlPath)
	} else {
		endpoint = t.config.Endpoint
		// Strip protocol for generic OTLP endpoint
		if len(endpoint) > 8 && endpoint[:8] == "https://" {
			endpoint = endpoint[8: ]
		} else if len(endpoint) > 7 && endpoint[:7] == "http://" {
			endpoint = endpoint[7: ]
		}
	}

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
	}

	if urlPath != "" {
		opts = append(opts, otlptracehttp.WithURLPath(urlPath))
	}

	// For Langfuse cloud, use Basic Authentication
	if isLangfuse && t.config.PublicKey != "" && t.config.SecretKey != "" {
		logger.Info("Configuring Langfuse Basic Auth with public key: %s...", t.config.PublicKey[:10])
		auth := base64.StdEncoding.EncodeToString([]byte(t.config.PublicKey + ":" + t.config.SecretKey))
		opts = append(opts, otlptracehttp.WithHeaders(map[string]string{
			"Authorization": "Basic " + auth,
		}))
	} else if t.config.SecretKey != "" {
		opts = append(opts, otlptracehttp.WithHeaders(map[string]string{
			"Authorization": "Bearer " + t.config.SecretKey,
		}))
	}

	// Insecure only if not using Langfuse cloud (for local development)
	if !isLangfuse {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	logger.Info("Creating OTLP HTTP exporter to: %s", endpoint)

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service name
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("axon-browser"),
			semconv.ServiceVersionKey.String("1.0.0"),
			semconv.DeploymentEnvironmentKey.String(t.environment),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(t.config.SampleRate)),
	)

	t.provider = tp
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Success("OTLP exporter initialized: %s", t.config.Endpoint)
	return nil
}

// StartSpan starts a new span with Axon attributes
func (t *AxonTelemetry) StartSpan(ctx context.Context, eventType AxonEventType, sessionID string, metadata map[string]interface{}) (context.Context, trace.Span) {
	if !t.enabled {
		return ctx, nil
	}

	// Create span name
	spanName := string(eventType)

	// Start OTLP span
	ctx, span := t.tracer.Start(ctx, spanName,
		trace.WithAttributes(
			attribute.String("axon.event_type", string(eventType)),
			attribute.String("axon.session_id", sessionID),
			attribute.String("axon.environment", t.environment),
		),
	)

	// Add custom metadata as attributes
	for key, value := range metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("axon.%s", key), fmt.Sprintf("%v", value)))
	}

	return ctx, span
}

// EndSpan ends a span
func (t *AxonTelemetry) EndSpan(ctx context.Context, span trace.Span, duration time.Duration, success bool, err error) {
	if span == nil || !t.enabled {
		return
	}

	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		span.SetAttributes(attribute.Bool("axon.success", false))
	} else {
		span.SetAttributes(attribute.Bool("axon.success", success))
	}

	span.SetAttributes(attribute.Int64("axon.duration_ms", duration.Milliseconds()))
	span.End()
}

// RecordEvent records an Axon event locally
func (t *AxonTelemetry) RecordEvent(event *AxonEvent) {
	if !t.enabled {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	event.Timestamp = time.Now()
	t.events = append(t.events, event)

	if len(t.events) > t.maxEvents {
		t.events = t.events[len(t.events)-t.maxEvents:]
	}
}

// GetEvents returns recorded events
func (t *AxonTelemetry) GetEvents() []*AxonEvent {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]*AxonEvent, len(t.events))
	copy(result, t.events)
	return result
}

// GetStats returns telemetry statistics
func (t *AxonTelemetry) GetStats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := map[string]interface{}{
		"total_events":   len(t.events),
		"enabled":        t.enabled,
		"provider":      t.config.Provider,
		"environment":   t.environment,
	}

	// Count by event type
	eventCounts := make(map[string]int)
	for _, e := range t.events {
		eventCounts[string(e.Type)]++
	}
	stats["event_counts"] = eventCounts

	return stats
}

// --- High-level tracking methods ---

// TrackSessionCreated tracks when a new session is created
func (t *AxonTelemetry) TrackSessionCreated(ctx context.Context, sessionID string) {
	if !t.enabled {
		return
	}

	ctx, span := t.StartSpan(ctx, EventSessionCreated, sessionID, nil)
	t.EndSpan(ctx, span, 0, true, nil)
}

// TrackSnapshot tracks snapshot operations with token savings
func (t *AxonTelemetry) TrackSnapshot(ctx context.Context, sessionID string, rawTokens, reducedTokens int, duration time.Duration) {
	if !t.enabled {
		return
	}

	metadata := map[string]interface{}{
		"raw_tokens":       rawTokens,
		"reduced_tokens":  reducedTokens,
		"token_savings":   rawTokens - reducedTokens,
		"savings_percent":  float64(rawTokens-reducedTokens) / float64(rawTokens) * 100,
	}

	ctx, span := t.StartSpan(ctx, EventSnapshotTaken, sessionID, metadata)
	span.SetAttributes(
		attribute.Int("axon.raw_tokens", rawTokens),
		attribute.Int("axon.reduced_tokens", reducedTokens),
		attribute.Int("axon.token_savings", rawTokens-reducedTokens),
	)
	t.EndSpan(ctx, span, duration, true, nil)
}

// TrackAction tracks action performance
func (t *AxonTelemetry) TrackAction(ctx context.Context, sessionID, actionType, ref string, success bool, duration time.Duration, err error) {
	if !t.enabled {
		return
	}

	eventType := EventActionClick
	if actionType == "fill" {
		eventType = EventActionFill
	} else if actionType == "navigate" {
		eventType = EventActionNavigate
	}

	if !success {
		eventType = EventActionFailed
	}

	metadata := map[string]interface{}{
		"action_type": actionType,
		"ref":         ref,
		"success":     success,
	}

	ctx, span := t.StartSpan(ctx, eventType, sessionID, metadata)
	t.EndSpan(ctx, span, duration, success, err)
}

// TrackSecurityEvent tracks security events (SSRF blocks, prompt injection)
func (t *AxonTelemetry) TrackSecurityEvent(ctx context.Context, eventType AxonEventType, sessionID, reason, url string) {
	if !t.enabled {
		return
	}

	metadata := map[string]interface{}{
		"reason": reason,
		"url":    url,
	}

	ctx, span := t.StartSpan(ctx, eventType, sessionID, metadata)
	span.SetAttributes(
		attribute.String("security.reason", reason),
		attribute.String("security.url", url),
	)
	t.EndSpan(ctx, span, 0, false, nil)
}

// TrackNetworkBlocked tracks network blocking (ads, trackers)
func (t *AxonTelemetry) TrackNetworkBlocked(ctx context.Context, sessionID string, blockedCount int, blockedType string) {
	if !t.enabled {
		return
	}

	eventType := EventAdsBlocked
	if blockedType == "tracker" {
		eventType = EventTrackersBlocked
	}

	metadata := map[string]interface{}{
		"blocked_count": blockedCount,
		"blocked_type":  blockedType,
	}

	ctx, span := t.StartSpan(ctx, eventType, sessionID, metadata)
	span.SetAttributes(attribute.Int("network.blocked_count", blockedCount))
	t.EndSpan(ctx, span, 0, true, nil)
}

// TrackPerformance tracks performance metrics
func (t *AxonTelemetry) TrackPerformance(ctx context.Context, sessionID string, memoryMB float64, cpuPercent float64) {
	if !t.enabled {
		return
	}

	metadata := map[string]interface{}{
		"memory_mb":    memoryMB,
		"cpu_percent":  cpuPercent,
	}

	ctx, span := t.StartSpan(ctx, EventMemoryUsage, sessionID, metadata)
	span.SetAttributes(
		attribute.Float64("performance.memory_mb", memoryMB),
		attribute.Float64("performance.cpu_percent", cpuPercent),
	)
	t.EndSpan(ctx, span, 0, true, nil)
}

// TrackLLMUsage tracks exact token usage from an external LLM
func (t *AxonTelemetry) TrackLLMUsage(ctx context.Context, sessionID string, promptTokens, completionTokens int, model string) {
	if !t.enabled {
		return
	}

	metadata := map[string]interface{}{
		"prompt_tokens":     promptTokens,
		"completion_tokens": completionTokens,
		"total_tokens":      promptTokens + completionTokens,
		"model":             model,
	}

	ctx, span := t.StartSpan(ctx, EventLLMTokenUsage, sessionID, metadata)
	span.SetAttributes(
		attribute.Int("llm.prompt_tokens", promptTokens),
		attribute.Int("llm.completion_tokens", completionTokens),
		attribute.Int("llm.total_tokens", promptTokens+completionTokens),
		attribute.String("llm.model", model),
	)
	t.EndSpan(ctx, span, 0, true, nil)
}

// Flush flushes any pending spans to the exporter
func (t *AxonTelemetry) Flush(ctx context.Context) error {
	if t.provider != nil {
		return t.provider.ForceFlush(ctx)
	}
	return nil
}

// Shutdown shuts down the telemetry provider
func (t *AxonTelemetry) Shutdown(ctx context.Context) error {
	if t.provider != nil {
		// Attempt to flush before shutdown
		_ = t.provider.ForceFlush(ctx)
		return t.provider.Shutdown(ctx)
	}
	return nil
}

// Global telemetry instance
var globalTelemetry *AxonTelemetry

// Init initializes the global telemetry provider
func Init(cfg *config.TelemetryConfig) error {
	// Convert config.TelemetryConfig to local TelemetryConfig
	localCfg := &TelemetryConfig{
		Enabled:    cfg.Enabled,
		Provider:  cfg.Provider,
		Endpoint:  cfg.Endpoint,
		PublicKey: cfg.PublicKey,
		SecretKey: cfg.SecretKey,
		Environment: cfg.Environment,
	}
	
	var err error
	globalTelemetry, err = NewAxonTelemetry(localCfg)
	return err
}

// GetGlobalTelemetry returns the global telemetry instance
func GetGlobalTelemetry() *AxonTelemetry {
	return globalTelemetry
}

// Flush flushes the global telemetry provider
func Flush(ctx context.Context) {
	if globalTelemetry != nil {
		_ = globalTelemetry.Flush(ctx)
	}
}

// Shutdown shuts down the global telemetry provider
func Shutdown() {
	if globalTelemetry != nil {
		// Use a local context with timeout for shutdown flush
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = globalTelemetry.Shutdown(ctx)
	}
}
