package telemetry

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/blaisecz/sleep-tracker/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// InitTracer initializes the global OpenTelemetry tracer provider.
// If Langfuse is not configured, this function is a no-op.
func InitTracer(ctx context.Context, cfg *config.Config, serviceName string) (func(context.Context) error, error) {
	if cfg.LangfuseBaseURL == "" || cfg.LangfusePublicKey == "" || cfg.LangfuseSecretKey == "" {
		// Langfuse not configured; keep default noop tracer provider.
		return func(context.Context) error { return nil }, nil
	}

	// Build Basic auth header from Langfuse public/secret keys.
	creds := cfg.LangfusePublicKey + ":" + cfg.LangfuseSecretKey
	auth := base64.StdEncoding.EncodeToString([]byte(creds))

	endpoint := fmt.Sprintf("%s/api/public/otel/v1/traces", cfg.LangfuseBaseURL)

	exporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpointURL(endpoint),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization": "Basic " + auth,
		}),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("langfuse.environment", cfg.LangfuseEnv),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
