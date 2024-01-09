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
	"errors"
	"hash"
	"io"
	"log"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type (
	// Tracer is trace.Tracer
	Tracer = trace.Tracer
	// Provider is trace.TraceProvider
	TracerProvider = trace.TracerProvider

	// Meter is meter.Meter
	Meter = metric.Meter
	// MeterProvider is meter.MeterProvider
	MeterProvider = metric.MeterProvider

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

// LogTraceProvider wraps the Logger to as a Provider.
func LogTraceProvider(logger *log.Logger, serviceName, serviceVersion string) (TracerProvider, MeterProvider, func() error, error) {
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

	exporter.stop = func() error {
		ctx := context.Background()
		meterErr := meterProvider.Shutdown(ctx)
		tracerErr := tracerProvider.Shutdown(ctx)
		return errors.Join(meterErr, tracerErr)
	}
	return tracerProvider, meterProvider, exporter.stop, nil
}
func LogTraceMeter(logger *log.Logger, serviceName, serviceVersion string) (Tracer, Meter) {
	tp, mp, _, _ := LogTraceProvider(logger, serviceName, serviceVersion)
	return tp.Tracer(serviceName), mp.Meter(serviceName)
}

type LogExporter struct {
	metricHash     hash.Hash
	metricExporter sdkmetric.Exporter
	stop           func() error
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
	var err error
	if e.traceExporter != nil {
		err = e.traceExporter.Shutdown(ctx)
	}
	if stop != nil {
		if stopErr := stop(); stopErr != nil && err == nil {
			err = stopErr
		}
	}
	return err
}

type bufEncoder struct{ jsenc *json.Encoder }

func (be bufEncoder) Encode(v any) error { return be.jsenc.Encode(v) }
