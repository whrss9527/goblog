package slogx

import (
	"context"
	"log/slog"
)

// TraceID handler

type TraceIDKey struct{}

// TraceID Get the traceID  from ctx and add into record
func TraceID(ctx context.Context, record *slog.Record) {
	traceID := ToString(ctx.Value(TraceIDKey{}))
	if len(traceID) != 0 {
		attr := slog.String("trace_id", traceID)
		record.AddAttrs(attr)
	}
}

// WithTraceID Set the traceID into ctx
func WithTraceID(ctx context.Context, traceID any) context.Context {
	return context.WithValue(ctx, TraceIDKey{}, traceID)
}
