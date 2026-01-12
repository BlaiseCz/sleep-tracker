package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Tracing starts an OpenTelemetry span for each HTTP request and
// propagates the context to downstream handlers and services.
func Tracing(next http.Handler) http.Handler {
	tracer := otel.Tracer("sleep-tracker-api/http")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		spanName := r.Method + " " + r.URL.Path
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.target", r.URL.Path),
			),
		)

		// Attach generic HTTP request metadata as Langfuse input
		inputPayload := map[string]any{
			"method": r.Method,
			"path":   r.URL.Path,
		}
		if r.URL.RawQuery != "" {
			inputPayload["query"] = r.URL.RawQuery
		}
		if r.Host != "" {
			inputPayload["host"] = r.Host
		}
		if r.RemoteAddr != "" {
			inputPayload["remote_addr"] = r.RemoteAddr
		}
		if inJSON, err := json.Marshal(inputPayload); err == nil {
			span.SetAttributes(attribute.String("langfuse.observation.input", string(inJSON)))
		}

		// Wrap ResponseWriter to capture status code
		tw := &traceResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(tw, r.WithContext(ctx))

		duration := time.Since(start)
		span.SetAttributes(attribute.Int("http.status_code", tw.statusCode))
		outputPayload := map[string]any{
			"status_code": tw.statusCode,
			"duration_ms": duration.Milliseconds(),
		}
		if outJSON, err := json.Marshal(outputPayload); err == nil {
			span.SetAttributes(attribute.String("langfuse.observation.output", string(outJSON)))
		}

		span.End()
	})
}

type traceResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (tw *traceResponseWriter) WriteHeader(code int) {
	tw.statusCode = code
	tw.ResponseWriter.WriteHeader(code)
}
