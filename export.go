// Copyright 2021, 2022 Tamás Gulácsi
//
//
// SPDX-License-Identifier: Apache-2.0

package otel

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-logr/logr"
	global "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	ctrlbasic "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/metric/export"
	"go.opentelemetry.io/otel/sdk/metric/export/aggregation"
	procbasic "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/sdkapi"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
)

type (
	Tracer   = trace.Tracer
	Provider = trace.TracerProvider
)

func SetGlobalTraceProvider(provider Provider) { global.SetTracerProvider(provider) }
func GlobalTraceProvider() Provider            { return global.GetTracerProvider() }
func GlobalTracer(name string) Tracer          { return global.Tracer(name) }

var HTTPPropagators = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{}, propagation.Baggage{},
)

func ExtractHTTP(ctx context.Context, headers http.Header) context.Context {
	return HTTPPropagators.Extract(ctx, propagation.HeaderCarrier(headers))
}
func InjectHTTP(ctx context.Context, headers http.Header) {
	HTTPPropagators.Inject(ctx, propagation.HeaderCarrier(headers))
}

func HTTPMiddleware(tracer Tracer, hndl http.Handler) http.Handler {
	if tracer == nil {
		tracer = global.Tracer("github.com/UNO-SOFT/otel")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := ExtractHTTP(r.Context(), r.Header)
		ctx, span := tracer.Start(ctx, r.URL.Path)
		InjectHTTP(ctx, w.Header())
		hndl.ServeHTTP(w, r)
		span.End()
	})
}

// nil sampler means sdktrace.AlwaysSample.
func LogTraceProvider(logger logr.Logger) (Provider, error) {
	exporter := &LogExporter{Logger: logger}
	var err error
	if exporter.traceExporter, err = stdouttrace.New(stdouttrace.WithWriter(&exporter.traceBuf)); err != nil {
		return nil, err
	}
	if exporter.metricExporter, err = stdoutmetric.New(stdoutmetric.WithWriter(&exporter.metricBuf)); err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
	pusher := ctrlbasic.New(
		procbasic.NewFactory(
			simple.NewWithHistogramDistribution(),
			exporter,
		),
		ctrlbasic.WithExporter(exporter),
	)
	ctx := context.Background()
	err = pusher.Start(ctx)
	exporter.stop = func() error { return pusher.Stop(ctx) }
	return tp, err
}
func LogTracer(logger logr.Logger, name string) Tracer {
	tp, _ := LogTraceProvider(logger)
	return tp.Tracer(name)
}

type LogExporter struct {
	logr.Logger
	stop           func() error
	traceBuf       bytes.Buffer
	traceExporter  *stdouttrace.Exporter
	metricBuf      bytes.Buffer
	metricExporter *stdoutmetric.Exporter
}

var _ export.Exporter = ((*LogExporter)(nil))

// ExportSpans writes SpanData in json format to stdout.
func (e *LogExporter) ExportSpans(ctx context.Context, data []sdktrace.ReadOnlySpan) error {
	e.traceBuf.Reset()
	if err := e.traceExporter.ExportSpans(ctx, data); err != nil {
		return err
	}
	e.Info("exportSpans", "trace", json.RawMessage(e.traceBuf.Bytes()))
	return nil
}
func (e *LogExporter) Export(ctx context.Context, resource *resource.Resource, checkpointSet export.InstrumentationLibraryReader) error {
	e.metricBuf.Reset()
	if err := e.metricExporter.Export(ctx, resource, checkpointSet); err != nil {
		return err
	}
	e.Info("export", "metric", json.RawMessage(e.metricBuf.Bytes()))
	return nil
}

// TemporalitySelector is a sub-interface of Exporter used to indicate whether the Processor should compute Delta or Cumulative Aggregations.
func (e *LogExporter) TemporalityFor(desc *sdkapi.Descriptor, kind aggregation.Kind) aggregation.Temporality {
	return e.metricExporter.TemporalityFor(desc, kind)
}
func (e *LogExporter) Shutdown(ctx context.Context) error {
	stop := e.stop
	e.stop = nil
	var err error
	if e.traceExporter != nil {
		err = e.traceExporter.Shutdown(ctx)
	}
	if stop != nil {
		if stopErr := e.stop(); stopErr != nil && err == nil {
			err = stopErr
		}
	}
	return err
}
