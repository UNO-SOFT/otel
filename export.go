// Copyright 2021 Tamás Gulácsi
//
//
// SPDX-License-Identifier: Apache-2.0

package otel

import (
	"context"
	"net/http"

	global "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	exportmetric "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	exporttrace "go.opentelemetry.io/otel/sdk/export/trace"
	"go.opentelemetry.io/otel/sdk/metric/controller/basic"
	ctrlbasic "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	procbasic "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
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
func LogTraceProvider(Log func(...interface{}) error) (Provider, error) {
	exporter := LogExporter{Log: Log}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
	pusher := ctrlbasic.New(
		procbasic.New(
			simple.NewWithExactDistribution(),
			exporter,
		),
		basic.WithPusher(exporter),
	)
	ctx := context.Background()
	pusher.Start(ctx)
	exporter.stop = func() { pusher.Stop(ctx) }
	return tp, nil
}
func LogTracer(Log func(...interface{}) error, name string) Tracer {
	tp, _ := LogTraceProvider(Log)
	return tp.Tracer(name)
}

type LogExporter struct {
	Log  func(...interface{}) error
	stop func()
}

// ExportSpans writes SpanData in json format to stdout.
func (e LogExporter) ExportSpans(ctx context.Context, data []*exporttrace.SpanSnapshot) error {
	var firstErr error
	for _, d := range data {
		/*
		   type SpanData struct {
		   	SpanContext  apitrace.SpanContext
		   	ParentSpanID apitrace.SpanID
		   	SpanKind     apitrace.SpanKind
		   	Name         string
		   	StartTime    time.Time
		   	// The wall clock time of EndTime will be adjusted to always be offset
		   	// from StartTime by the duration of the span.
		   	EndTime                  time.Time
		   	Attributes               []kv.KeyValue
		   	MessageEvents            []Event
		   	Links                    []apitrace.Link
		   	StatusCode               codes.Code
		   	StatusMessage            string
		   	HasRemoteParent          bool
		   	DroppedAttributeCount    int
		   	DroppedMessageEventCount int
		   	DroppedLinkCount         int

		   	// ChildSpanCount holds the number of child span created for this span.
		   	ChildSpanCount int

		   	// Resource contains attributes representing an entity that produced this span.
		   	Resource *resource.Resource

		   	// InstrumentationLibrary defines the instrumentation library used to
		   	// providing instrumentation.
		   	InstrumentationLibrary instrumentation.Library
		   }
		*/
		if err := e.Log("msg", "trace", "parent", d.ParentSpanID, "kind", d.SpanKind, "name", d.Name,
			"spanID", d.SpanContext.SpanID, "traceID", d.SpanContext.TraceID, "start", d.StartTime, "end", d.EndTime,
			"attrs", d.Attributes, "events", d.MessageEvents, "links", d.Links,
			"statusCode", d.StatusCode, "statusMsg", d.StatusMessage,
		); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
func (e LogExporter) Export(ctx context.Context, checkpointSet exportmetric.CheckpointSet) error {
	return checkpointSet.ForEach(exportmetric.StatelessExportKindSelector(), func(rec exportmetric.Record) error {
		return e.Log("msg", "labels", rec.Labels(), "resource", rec.Resource(), "start", rec.StartTime(), "end", rec.EndTime(), "agg", rec.Aggregation())
	})
}
func (e LogExporter) ExportKindFor(desc *metric.Descriptor, kind aggregation.Kind) exportmetric.ExportKind {
	return exportmetric.StatelessExportKindSelector().ExportKindFor(desc, kind)
}
func (e LogExporter) Shutdown(ctx context.Context) error {
	if e.stop != nil {
		e.stop()
	}
	return nil
}
