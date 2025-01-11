// Copyright 2021, 2025 Tamás Gulácsi
//
//
// SPDX-License-Identifier: Apache-2.0

package otel

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// LogRecord is log.Record
type LogRecord = log.Record

func LogStringValue(v string) log.Value       { return log.StringValue(v) }
func LogInt64(k string, v int64) log.KeyValue { return log.Int64(k, v) }
func LogString(k, v string) log.KeyValue      { return log.String(k, v) }

// https://opentelemetry.io/docs/instrumentation/go/getting-started/

func NewResource(serviceName, serviceVersion string) (*resource.Resource, error) {
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		))
}

func NewPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// NewTraceProvider returns a TraceProvider which exports using the given traceExporter,
// with defailt 10s BatchSpanProcessor timeout.
func NewTracerProvider(traceExporter Exporter, res *resource.Resource, options ...trace.BatchSpanProcessorOption) *trace.TracerProvider {
	return trace.NewTracerProvider(
		trace.WithBatcher(traceExporter,
			append(append(make([]trace.BatchSpanProcessorOption, 0, 1+len(options)),
				trace.WithBatchTimeout(10*time.Second),
			), options...)...),
		trace.WithResource(res),
	)
}

// NewMeterProvider returns a MeterProvider which exports using the given metricExporter,
// with default 1m PeriodicReader.
func NewMeterProvider(metricExporter metric.Exporter, res *resource.Resource, options ...metric.Option) *metric.MeterProvider {
	return metric.NewMeterProvider(
		append(append(make([]metric.Option, 0, 2+len(options)),
			metric.WithResource(res),
			metric.WithReader(metric.NewPeriodicReader(
				metricExporter, metric.WithInterval(1*time.Minute),
			)),
		), options...)...,
	)
}

// NewLoggerProvider returns a log.LoggerProvider which exports using the given loggerExporter,
// with default 1MiB BatchProcessor.
//
// Such a loggerExporter can be created with otlploghttp, for example.
func NewLoggerProvider(loggerExporter sdklog.Exporter, res *resource.Resource, options ...sdklog.BatchProcessorOption) *sdklog.LoggerProvider {
	return sdklog.NewLoggerProvider(
		sdklog.WithProcessor(
			sdklog.NewBatchProcessor(loggerExporter,
				append(append(make([]sdklog.BatchProcessorOption, 0, 3+len(options)),
					sdklog.WithExportBufferSize(1<<20),
					sdklog.WithExportMaxBatchSize(1<<20),
					sdklog.WithExportTimeout(24*time.Hour),
				), options...)...,
			),
		),
		sdklog.WithResource(res),
	)
}

func LoggerEnabled(ctx context.Context, logger log.Logger) bool {
	return logger.Enabled(ctx, log.EnabledParameters{})
}
