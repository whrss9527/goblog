# slogx

slogx wrapped [log/slog](https://github.com/golang/go/tree/master/src/log/slog), simplify the difficulty of use.

## usage

- set default `slog.Logger`

```go

var DefaultConfig = []HandlerOption{
    WithLevel("INFO"),
    WithReplaceAttr(SourceBase),
    WithAddSource(true),
    WithHandler(TraceID),
}

// Call the "InitSlogX" method and then use the slog package according to the configuration you pass.
InitSlogX(DefaultConfig...)

ctx := WithTraceID(context.Background(), 123)

slog.InfoContext(ctx, "msg", "a", "b")

//!!! If you use DefaultConfig "TraceID", use "WithTraceID" set TraceID into ctx

```

***Output***

```json
time=2023-10-11T15:12:01.250+08:00 level=INFO source=slogx_test.go:17 msg=msg a=b trace_id=123
```

- define your handler

```go
type otherKey struct{}
    InitSlogX(
    WithLevel("INFO"),
    WithAddSource(true),
    WithReplaceAttr(SourceBase),
    WithHandler(func(ctx context.Context, record *slog.Record) {
        val := ToString(ctx.Value(otherKey{}))
            if len(val) != 0 {
                attr := slog.String("other_id", val)
                record.AddAttrs(attr)
            }
        }),
    )
    ctx = context.WithValue(ctx, otherKey{}, "abc")
    slog.InfoContext(ctx, "define handler")
```

***Output***

```json
time=2023-10-11T15:12:01.259+08:00 level=INFO msg="define handler" otherID=abc
```

- create a `slog.Logger` variable

```go

logger := New(WithLevel("INFO"), WithAddSource(true), WithReplaceAttr(SourceBase))
//The "New" method will return a "slog.Logger" variable for you to use everywhere.
logger.Info("logger", "la", "lb")
```

***Output***

```json
time=2023-10-10T17:41:55.639+08:00 level=INFO source=slogx_test.go:13 msg=logger la=lb
```

- create a gorm `logger.Logger` interface from `slog.Logger`
```go
NewSlogXGormLogger(New().Handler(), slog.LevelInfo, logger.Config{})
```

## Important note

go version:1.21+

