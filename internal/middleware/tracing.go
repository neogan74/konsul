package middleware

import (
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware creates a middleware for OpenTelemetry tracing
func TracingMiddleware(serviceName string) fiber.Handler {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(c *fiber.Ctx) error {
		// Extract context from incoming request
		ctx := propagator.Extract(c.UserContext(), &fiberCarrier{c: c})

		// Start a new span
		spanName := c.Method() + " " + c.Route().Path
		if spanName == " " {
			spanName = c.Method() + " " + c.Path()
		}

		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethod(c.Method()),
				semconv.HTTPURL(c.OriginalURL()),
				semconv.HTTPRoute(c.Route().Path),
				semconv.HTTPScheme(c.Protocol()),
				semconv.HTTPTarget(c.Path()),
				semconv.NetHostName(c.Hostname()),
				semconv.HTTPUserAgent(c.Get("User-Agent")),
				attribute.String("http.client_ip", c.IP()),
			),
		)
		defer span.End()

		// Store context in fiber context
		c.SetUserContext(ctx)

		// Store trace ID in locals for logging correlation
		if span.SpanContext().HasTraceID() {
			c.Locals("trace_id", span.SpanContext().TraceID().String())
			c.Set("X-Trace-Id", span.SpanContext().TraceID().String())
		}

		// Continue processing
		err := c.Next()

		// Set span status based on response
		statusCode := c.Response().StatusCode()
		span.SetAttributes(semconv.HTTPStatusCode(statusCode))

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		// Set span status based on HTTP status code
		if statusCode >= 500 {
			span.SetStatus(codes.Error, "Internal server error")
		} else if statusCode >= 400 {
			span.SetStatus(codes.Error, "Client error")
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return nil
	}
}

// fiberCarrier adapts fiber.Ctx to propagation.TextMapCarrier
type fiberCarrier struct {
	c *fiber.Ctx
}

func (fc *fiberCarrier) Get(key string) string {
	return fc.c.Get(key)
}

func (fc *fiberCarrier) Set(key, value string) {
	fc.c.Set(key, value)
}

func (fc *fiberCarrier) Keys() []string {
	keys := make([]string, 0)
	fc.c.Request().Header.VisitAll(func(key, _ []byte) {
		keys = append(keys, string(key))
	})
	return keys
}
