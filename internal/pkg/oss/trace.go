package oss

import (
	"context"

	kr "github.com/go-kratos/kratos/v2/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var ossTracer = otel.Tracer("wuwen.com/wwuc/internal/pkg/oss")

// startOssSpan 为 oss 操作开启 client span，挂在请求链路上（与 GORM、服务端 span 同一 trace）。
// 返回携带 span 的 ctx 与结束函数；end 的入参兼容 *kr.Error（含 typed-nil）与 error，nil 表示成功。
// 入参用 any 是因为 StorageService 各方法返回 *kr.Error 或 error，
// 若直接收 error，typed-nil 的 *kr.Error 会被误判为失败。
func startOssSpan(ctx context.Context, op string, attrs ...attribute.KeyValue) (context.Context, func(err any)) {
	ctx, span := ossTracer.Start(ctx, op,
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(attrs...),
	)
	return ctx, func(err any) {
		switch e := err.(type) {
		case nil:
		case *kr.Error:
			if e != nil {
				span.RecordError(e)
				span.SetStatus(codes.Error, e.GetMessage())
			}
		case error:
			if e != nil {
				span.RecordError(e)
				span.SetStatus(codes.Error, e.Error())
			}
		}
		span.End()
	}
}
