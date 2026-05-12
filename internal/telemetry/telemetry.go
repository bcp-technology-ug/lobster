// Package telemetry sets up and tears down OpenTelemetry tracing for a lobster
// run. When an OTLP endpoint is configured a TracerProvider is initialised and
// the global OTel tracer is pointed at it; otherwise a no-op provider is used
// so all trace calls throughout the code path are safe no-ops.
package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const defaultShutdownTimeout = 5 * time.Second

// Config holds OTel configuration.
type Config struct {
	// Endpoint is the OTLP HTTP collector endpoint, e.g. "http://localhost:4318".
	// When empty, a no-op tracer is used and no spans are emitted.
	Endpoint string

	// ServiceName is the service.name resource attribute value.
	// Defaults to "lobster" when empty.
	ServiceName string
}

// Provider wraps an OTel TracerProvider and exposes a Shutdown method.
type Provider struct {
	tp       *sdktrace.TracerProvider
	shutdown func(context.Context) error
}

// Setup initialises a TracerProvider according to cfg and sets it as the global
// OTel provider. Call Shutdown when the run completes.
//
// When cfg.Endpoint is empty, a no-op provider is returned; no OTel calls are
// made and no network connections are opened.
func Setup(ctx context.Context, cfg Config) (*Provider, error) {
	if cfg.Endpoint == "" {
		nopTP := noop.NewTracerProvider()
		otel.SetTracerProvider(nopTP)
		return &Provider{shutdown: func(context.Context) error { return nil }}, nil
	}

	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "lobster"
	}

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(cfg.Endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
		resource.WithProcess(),
		resource.WithOS(),
	)
	if err != nil {
		// Non-fatal — fall back to empty resource.
		res = resource.Empty()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return &Provider{
		tp:       tp,
		shutdown: tp.Shutdown,
	}, nil
}

// Shutdown flushes pending spans and shuts down the provider gracefully.
// It is safe to call Shutdown on a no-op Provider.
func (p *Provider) Shutdown(ctx context.Context) error {
	shutCtx, cancel := context.WithTimeout(ctx, defaultShutdownTimeout)
	defer cancel()
	return p.shutdown(shutCtx)
}

// Tracer returns a named tracer from the global OTel provider.
func Tracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}
