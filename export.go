// Copyright 2021, 2023 Tamás Gulácsi
//
//
// SPDX-License-Identifier: Apache-2.0

// Package otel tries to simplify usage of OpenTelemetry.
//
// A nice write-up of using OpenTelemetry (directly) is at https://www.komu.engineer/blogs/11/opentelemetry-and-go
package otel

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"log"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	olog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type (
	// Tracer is trace.Tracer
	Tracer = trace.Tracer
	// TacerProvider is trace.TraceProvider
	TracerProvider = trace.TracerProvider

	// Meter is meter.Meter
	Meter = metric.Meter
	// MeterProvider is meter.MeterProvider
	MeterProvider = metric.MeterProvider

	// Logger is log.Logger
	Logger = olog.Logger
	// LoggerProvider is sdklog.LoggerProvider
	LoggerProvider = sdklog.LoggerProvider

	Exporter interface {
		Shutdown(ctx context.Context) error
		ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error
	}
)

func SetGlobalTracerProvider(provider TracerProvider) { otel.SetTracerProvider(provider) }
func GlobalTracerProvider() TracerProvider            { return otel.GetTracerProvider() }
func GlobalTracer(name string) Tracer                 { return otel.Tracer(name) }

func SetGlobalMeterProvider(provider MeterProvider) { otel.SetMeterProvider(provider) }
func GlobalMeterProvider() MeterProvider            { return otel.GetMeterProvider() }
func GlobalMeter(name string) Meter                 { return otel.Meter(name) }

func StartTrace(ctx context.Context, name, traceID, spanID string) (context.Context, trace.Span) {
	ctx = ContextWithTraceSpan(ctx, traceID, spanID)
	return GlobalTracer(name).Start(ctx, trace.SpanContextFromContext(ctx).SpanID().String())
}
func ContextWithTraceSpan(ctx context.Context, traceID, spanID string) context.Context {
	if traceID != "" {
		if len(traceID) < 32 {
			traceID = strings.Repeat("0", 32-len(traceID)) + traceID
		} else if len(traceID) > 32 {
			traceID = traceID[:32]
		}
		if traceID, err := trace.TraceIDFromHex(traceID); err == nil && traceID.IsValid() {
			ctx = trace.ContextWithSpanContext(ctx, trace.SpanContextFromContext(ctx).
				WithTraceID(traceID))
		}
	}
	if spanID == "" {
		spanID = fmt.Sprintf("%016x", time.Now().UnixMicro())
	} else if len(spanID) < 16 {
		spanID = strings.Repeat("0", 32-len(spanID))
	} else if len(spanID) > 16 {
		spanID = spanID[:16]
	}
	if spanID, err := trace.SpanIDFromHex(spanID); err == nil && spanID.IsValid() {

		ctx = trace.ContextWithSpanContext(ctx, trace.SpanContextFromContext(ctx).
			WithSpanID(spanID))
	}
	return ctx
}

// LogTraceProvider wraps the Logger to as a Provider.
func LogTraceProvider(logger *log.Logger, serviceName, serviceVersion string) (TracerProvider, MeterProvider, func(context.Context) error, error) {
	exporter := &LogExporter{Logger: logger, metricHash: sha256.New224()}
	te, err := stdouttrace.New(stdouttrace.WithWriter(&exporter.traceBuf))
	if err != nil {
		return nil, nil, nil, err
	}
	me, err := stdoutmetric.New(
		stdoutmetric.WithEncoder(bufEncoder{
			jsenc: json.NewEncoder(io.MultiWriter(&exporter.metricBuf, exporter.metricHash)),
		}))
	if err != nil {
		return nil, nil, nil, err
	}
	exporter.traceExporter, exporter.metricExporter = te, me

	res, err := NewResource(serviceName, serviceVersion)
	if err != nil {
		return nil, nil, nil, err
	}

	meterProvider := NewMeterProvider(exporter, res)
	tracerProvider := NewTracerProvider(exporter, res)

	exporter.stop = func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		errCh := make(chan error, 2)
		go func() { errCh <- meterProvider.Shutdown(ctx) }()
		go func() { errCh <- tracerProvider.Shutdown(ctx) }()
		select {
		case err := <-errCh:
			return err
		case <-ctx.Done():
			select {
			case err := <-errCh:
				return err
			default:
				return ctx.Err()
			}
		}
	}
	return tracerProvider, meterProvider, exporter.stop, nil
}
func LogTraceMeterLogger(logger *log.Logger, serviceName, serviceVersion string) (Tracer, Meter) {
	tp, mp, _, _ := LogTraceProvider(logger, serviceName, serviceVersion)
	return tp.Tracer(serviceName), mp.Meter(serviceName)
}

type LogExporter struct {
	metricHash     hash.Hash
	metricExporter sdkmetric.Exporter
	stop           func(context.Context) error
	traceExporter  *stdouttrace.Exporter
	*log.Logger
	traceBuf   strings.Builder
	metricBuf  strings.Builder
	lastMetric [sha256.Size224]byte
}

var _ sdkmetric.Exporter = ((*LogExporter)(nil))

// Temporality returns the Temporality to use for an instrument kind.
func (e *LogExporter) Temporality(kind sdkmetric.InstrumentKind) metricdata.Temporality {
	return e.metricExporter.Temporality(kind)
}

// Aggregation returns the Aggregation to use for an instrument kind.
func (e *LogExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(kind)
}

// ForceFlush flushes any metric data held by an exporter.
//
// The deadline or cancellation of the passed context must be honored. An
// appropriate error should be returned in these situations.
func (e *LogExporter) ForceFlush(ctx context.Context) error { return nil }

// ExportSpans exports a batch of spans.
//
// This function is called synchronously, so there is no concurrency
// safety requirement. However, due to the synchronous calling pattern,
// it is critical that all timeouts and cancellations contained in the
// passed context must be honored.
//
// Any retry logic must be contained in this function. The SDK that
// calls this function will not implement any retry logic. All errors
// returned by this function are considered unrecoverable and will be
// reported to a configured error Handler.
func (e *LogExporter) ExportSpans(ctx context.Context, data []sdktrace.ReadOnlySpan) error {
	e.traceBuf.Reset()
	e.traceBuf.WriteString("exportSpans trace=")
	if err := e.traceExporter.ExportSpans(ctx, data); err != nil {
		return err
	}
	e.Logger.Output(2, e.traceBuf.String())
	return nil
}

// Export serializes and transmits metric data to a receiver.
//
// This is called synchronously, there is no concurrency safety
// requirement. Because of this, it is critical that all timeouts and
// cancellations of the passed context be honored.
//
// All retry logic must be contained in this function. The SDK does not
// implement any retry logic. All errors returned by this function are
// considered unrecoverable and will be reported to a configured error
// Handler.
func (e *LogExporter) Export(ctx context.Context, resource *metricdata.ResourceMetrics) error {
	e.metricBuf.Reset()
	e.metricBuf.WriteString("export metric=")
	e.metricHash.Reset()
	if err := e.metricExporter.Export(ctx, resource); err != nil {
		return err
	}
	if e.metricBuf.Len() == 0 {
		return nil
	}
	var hsh [sha256.Size224]byte
	e.metricHash.Sum(hsh[:0])
	if hsh == e.lastMetric {
		return nil
	}
	copy(e.lastMetric[:], hsh[:])
	e.Logger.Output(2, e.metricBuf.String())
	return nil
}

// Shutdown flushes all metric data held by an exporter and releases any
// held computational resources.
//
// The deadline or cancellation of the passed context must be honored. An
// appropriate error should be returned in these situations.
//
// After Shutdown is called, calls to Export will perform no operation and
// instead will return an error indicating the shutdown state.
func (e *LogExporter) Shutdown(ctx context.Context) error {
	stop := e.stop
	e.stop = nil
	if stop == nil && e.traceExporter == nil {
		return nil
	}
	// e.Logger.Printf("Shutdown te=%p stop=%p", e.traceExporter, stop)
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	errCh := make(chan error, 2)
	if e.traceExporter != nil {
		go func() { errCh <- e.traceExporter.Shutdown(ctx) }()
	}
	if stop != nil {
		go func() { errCh <- stop(ctx) }()
	}
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		select {
		case err := <-errCh:
			return err
		default:
			return ctx.Err()
		}
	}
}

type bufEncoder struct{ jsenc *json.Encoder }

func (be bufEncoder) Encode(v any) error { return be.jsenc.Encode(v) }
