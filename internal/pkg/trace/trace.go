package trace

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"wuwen.com/wwuc/internal/conf"
)

// Init 初始化全局 TracerProvider 并注册 W3C 传播器。
// 未开启（c.Enable=false 或 c 为 nil）时保持默认 noop，返回空 shutdown，不报错。
// endpoint 为 OTLP gRPC 上报地址（腾讯云 APM 或自建 Jaeger/Collector），
// token 非空时通过 X-CTG-Token 请求头鉴权（腾讯云 APM 约定）。
func Init(ctx context.Context, c *conf.Trace, serviceName string) (func(context.Context) error, error) {
	if c == nil || !c.Enable || c.Endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(c.Endpoint),
		otlptracegrpc.WithTimeout(10 * time.Second),
	}
	if c.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	if c.Token != "" {
		opts = append(opts, otlptracegrpc.WithHeaders(map[string]string{"X-CTG-Token": c.Token}))
	}
	exp, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	ratio := c.SamplerRatio
	if ratio <= 0 {
		ratio = 0.1
	}
	if len(c.ServiceEnv) == 0 {
		c.ServiceEnv = serviceName
	}
	resOpts := []resource.Option{
		resource.WithAttributes(semconv.ServiceName(c.ServiceEnv)),
	}
	if c.ServiceEnv != "" {
		resOpts = append(resOpts, resource.WithAttributes(
			semconv.DeploymentEnvironment(c.ServiceEnv),
		))
	}
	if c.Token != "" {
		resOpts = append(resOpts, resource.WithAttributes(
			attribute.Key("token").String(c.Token),
		))
	}
	res, err := resource.New(ctx, resOpts...)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		// ParentBased：跟随上游采样决定（跨服务续链时不丢），否则按比例采样
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	return tp.Shutdown, nil
}

// Detach 返回脱离请求生命周期但保留 span 关联的 context：
// 用于异步 goroutine，请求 ctx 取消后任务继续执行，
// 且链路上仍挂在发起请求的 span 下，不会被截断。
func Detach(ctx context.Context) context.Context {
	return oteltrace.ContextWithSpanContext(context.Background(), oteltrace.SpanContextFromContext(ctx))
}
