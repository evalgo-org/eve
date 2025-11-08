package otel

import (
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

// GetTraceID extracts the OpenTelemetry trace ID from the current context
func GetTraceID(c echo.Context) string {
	span := trace.SpanFromContext(c.Request().Context())
	if !span.IsRecording() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// GetSpanID extracts the OpenTelemetry span ID from the current context
func GetSpanID(c echo.Context) string {
	span := trace.SpanFromContext(c.Request().Context())
	if !span.IsRecording() {
		return ""
	}
	return span.SpanContext().SpanID().String()
}

// AddCorrelationToBaggage adds semantic workflow correlation IDs to OpenTelemetry baggage
// This allows OTel traces to reference semantic workflow data
func AddCorrelationToBaggage(c echo.Context, correlationID, operationID string) {
	ctx := c.Request().Context()

	// Get existing baggage
	bag := baggage.FromContext(ctx)

	// Add correlation IDs
	member1, _ := baggage.NewMember("correlation_id", correlationID)
	member2, _ := baggage.NewMember("operation_id", operationID)

	bag, _ = bag.SetMember(member1)
	bag, _ = bag.SetMember(member2)

	// Update context
	newCtx := baggage.ContextWithBaggage(ctx, bag)
	c.SetRequest(c.Request().WithContext(newCtx))
}

// AddActionMetadataToBaggage adds semantic action metadata to OTel baggage
func AddActionMetadataToBaggage(c echo.Context, actionType, objectType string) {
	ctx := c.Request().Context()

	bag := baggage.FromContext(ctx)

	member1, _ := baggage.NewMember("action_type", actionType)
	member2, _ := baggage.NewMember("object_type", objectType)

	bag, _ = bag.SetMember(member1)
	bag, _ = bag.SetMember(member2)

	newCtx := baggage.ContextWithBaggage(ctx, bag)
	c.SetRequest(c.Request().WithContext(newCtx))
}

// GetCorrelationFromBaggage retrieves correlation ID from OTel baggage
// Useful for downstream services to extract workflow context
func GetCorrelationFromBaggage(c echo.Context) (correlationID, operationID string) {
	ctx := c.Request().Context()
	bag := baggage.FromContext(ctx)

	if member := bag.Member("correlation_id"); member.Value() != "" {
		correlationID = member.Value()
	}

	if member := bag.Member("operation_id"); member.Value() != "" {
		operationID = member.Value()
	}

	return
}
