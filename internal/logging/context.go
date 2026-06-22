package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type contextKey string

const traceIDKey contextKey = "trace_id"

// GenerateTraceID returns a 16-character hex trace ID.
func GenerateTraceID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// WithTraceID injects a trace ID into the context.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}

// GetTraceID extracts the trace ID from the context.
func GetTraceID(ctx context.Context) string {
	v, _ := ctx.Value(traceIDKey).(string)
	return v
}
