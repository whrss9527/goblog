package slogx

import (
	"context"
	"log/slog"
)

var DefaultConfig = []HandlerOption{
	WithLevel("INFO"),
	WithReplaceAttr(SourceBase),
	WithAddSource(true),
	WithHandler(TraceID),
}

type SlogX struct {
	opt *Options

	*slog.TextHandler
}

func New(opts ...HandlerOption) *slog.Logger {
	s := NewLogger(opts...)
	return slog.New(s)
}

func InitSlogX(opts ...HandlerOption) {
	slog.SetDefault(New(opts...))
}

func (x *SlogX) Handle(ctx context.Context, record slog.Record) error {
	if x.opt.handleFunc != nil {
		x.opt.handleFunc(ctx, &record)
	}
	return x.TextHandler.Handle(ctx, record)
}
