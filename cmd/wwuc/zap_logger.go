package main

import (
    "context"
    "fmt"

    "github.com/go-kratos/kratos/v2/log"
    "go.opentelemetry.io/otel/trace"
    "go.uber.org/zap"

    "wuwen.com/wwuc/internal/biz/common"
)

var kratosLogValuer log.Valuer = func(ctx context.Context) interface{} {
    reqCtx := common.ReqCtxFromContext(ctx)

    return reqCtx.GetRequestId()
}

// traceLogValuer 从 ctx 提取当前 trace id（无链路上下文时返回 "-"），
// 与 requestId 并列输出，便于从日志反查链路瀑布图。
var traceLogValuer log.Valuer = func(ctx context.Context) interface{} {
    if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
        return sc.TraceID().String()
    }
    return "-"
}

var _ log.Logger = (*Logger)(nil)

type Logger struct {
    log    *zap.Logger
    msgKey string
}

func NewLogger(zlog *zap.Logger) *Logger {
    return &Logger{
        log:    zlog,
        msgKey: log.DefaultMessageKey,
    }
}

func (l *Logger) Log(level log.Level, keyvals ...interface{}) error {
    var (
        msg       = ""
        requestId = ""
        keylen    = len(keyvals)
    )
    if keylen == 0 || keylen%2 != 0 {
        l.log.Warn(fmt.Sprint("Keyvalues must appear in pairs: ", keyvals))
        return nil
    }

    data := make([]zap.Field, 0, (keylen/2)+1)
    for i := 0; i < keylen; i += 2 {
        if keyvals[i].(string) == l.msgKey {
            msg, _ = keyvals[i+1].(string)
            continue
        }
        if keyvals[i].(string) == "requestId" {
            requestId, _ = keyvals[i+1].(string)
            continue
        }
        data = append(data, zap.Any(fmt.Sprint(keyvals[i]), keyvals[i+1]))
    }

    switch level {
    case log.LevelDebug:
        l.log.Debug(fmt.Sprintf("requestId:%s msg:%s", requestId, msg), data...)
    case log.LevelInfo:
        l.log.Info(fmt.Sprintf("requestId:%s msg:%s", requestId, msg), data...)
    case log.LevelWarn:
        l.log.Warn(fmt.Sprintf("requestId:%s msg:%s", requestId, msg), data...)
    case log.LevelError:
        l.log.Error(fmt.Sprintf("requestId:%s msg:%s", requestId, msg), data...)
    case log.LevelFatal:
        l.log.Fatal(fmt.Sprintf("requestId:%s msg:%s", requestId, msg), data...)
    }
    return nil
}

func (l *Logger) Sync() error {
    return l.log.Sync()
}

func (l *Logger) Close() error {
    return l.Sync()
}
