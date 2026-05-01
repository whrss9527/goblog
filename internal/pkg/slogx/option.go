package slogx

import (
	"context"
	"log/slog"
	"os"
)

// Handler slog handle
type (
	Handler func(ctx context.Context, record *slog.Record)

	Options struct {
		*slog.HandlerOptions
		handleFunc Handler
	}
)

type HandlerOption func(x *SlogX)

// WithAddSource AddSource causes the handler to compute the source code position
// of the log statement and add a SourceKey attribute to the output.
func WithAddSource(add bool) HandlerOption {
	return func(h *SlogX) {
		h.opt.AddSource = add
	}
}

// WithReplaceAttr ReplaceAttr is called to rewrite each non-group attribute before it is logged.
func WithReplaceAttr(replace func(groups []string, a slog.Attr) slog.Attr) HandlerOption {
	return func(h *SlogX) {
		h.opt.ReplaceAttr = replace
	}
}

// WithLevel Level reports the minimum record level that will be logged.
func WithLevel(level string) HandlerOption {
	return func(h *SlogX) {
		l := slog.LevelInfo
		_ = l.UnmarshalText([]byte(level))
		h.opt.Level = l
	}
}

// WithHandler Handle handles the Record.
func WithHandler(handler Handler) HandlerOption {
	return func(h *SlogX) {
		h.opt.handleFunc = handler
	}
}

func NewLogger(opts ...HandlerOption) *SlogX {
	h := &SlogX{opt: &Options{HandlerOptions: new(slog.HandlerOptions)}}
	for _, f := range opts {
		f(h)
	}
	h.TextHandler = slog.NewTextHandler(os.Stdout, h.opt.HandlerOptions)
	return h
}
