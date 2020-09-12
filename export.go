// Copyright 2020 Tamás Gulácsi
//
//
// SPDX-License-Identifier: Apache-2.0

package otel

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/propagation"
	"go.opentelemetry.io/otel/api/trace"
	setrace "go.opentelemetry.io/otel/sdk/export/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type (
	Tracer   = trace.Tracer
	Provider = trace.Provider
)

func SetGlobalTraceProvider(provider Provider) { global.SetTraceProvider(provider) }
func GlobalTraceProvider() Provider            { return global.TraceProvider() }
func GlobalTracer(name string) Tracer          { return global.Tracer(name) }

var HTTPPropagators = propagation.New(
	propagation.WithExtractors(trace.DefaultHTTPPropagator(), trace.B3{}),
	propagation.WithInjectors(trace.DefaultHTTPPropagator(), trace.B3{}),
)

func ExtractHTTP(ctx context.Context, headers http.Header) context.Context {
	return propagation.ExtractHTTP(ctx, HTTPPropagators, headers)
}
func InjectHTTP(ctx context.Context, headers http.Header) {
	propagation.InjectHTTP(ctx, HTTPPropagators, headers)
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
func LogTraceProvider(Log func(...interface{}) error, sampler sdktrace.Sampler) (Provider, error) {
	if sampler == nil {
		sampler = sdktrace.AlwaysSample()
	}
	return sdktrace.NewProvider(
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sampler}),
		sdktrace.WithSyncer(LogExporter{Log: Log}),
	)
}
func LogTracer(Log func(...interface{}) error, name string) Tracer {
	if Log == nil {
		return global.Tracer(name)
	}
	exporter := LogExporter{Log: Log}
	tp, err := sdktrace.NewProvider(
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(exporter),
	)
	if err != nil {
		panic(err)
	}
	return tp.Tracer(name)
}

type LogExporter struct {
	Log func(...interface{}) error
}

// ExportSpans writes SpanData in json format to stdout.
func (e LogExporter) ExportSpans(ctx context.Context, data []*setrace.SpanData) {
	for _, d := range data {
		e.ExportSpan(ctx, d)
	}
}

// ExportSpan writes a SpanData in json format to stdout.
func (e LogExporter) ExportSpan(ctx context.Context, data *setrace.SpanData) {
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
	e.Log("msg", "trace", "parent", data.ParentSpanID, "kind", data.SpanKind, "name", data.Name,
		"spanID", data.SpanContext.SpanID, "traceID", data.SpanContext.TraceID, "start", data.StartTime, "end", data.EndTime,
		"attrs", data.Attributes, "events", data.MessageEvents, "links", data.Links,
		"statusCode", data.StatusCode, "statusMsg", data.StatusMessage,
	)
}
