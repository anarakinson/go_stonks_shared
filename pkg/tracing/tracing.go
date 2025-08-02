package tracing

import (
	"context"
	"crypto/tls"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"google.golang.org/grpc/credentials"
)

func InitTracerProvider(jaegerEndpoint, serviceName, serviceVersion, deploymentEnv string, tlsConfig *tls.Config) (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	// Подключаемся к Jaeger через OTLP/gRPC (порт 4317)
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(jaegerEndpoint), // "jaeger:4317" или localhost:4317
	}

	if tlsConfig != nil {
		// Дефолтная конфигурация TLS если не передана
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		opts = append(opts, otlptracegrpc.WithInsecure()) // для тестов (без TLS)
	}

	traceExporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Ресурсы трейсов (метаданные сервиса)
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),       // "1.0.0"
			semconv.DeploymentEnvironment(deploymentEnv), // production / development
		),
	)
	if err != nil {
		return nil, err
	}

	// Настраиваем TracerProvider
	traceIDRatio := 1.0
	if deploymentEnv == "production" {
		traceIDRatio = 0.1
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(
			sdktrace.ParentBased(
				sdktrace.TraceIDRatioBased(traceIDRatio), // 100% трейсов (для прода — 0.1)
			),
		),
	)

	// Устанавливаем глобальные настройки
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return tp, nil
}
