// Copyright 2021, 2026 Tamás Gulácsi
//
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package otel tries to simplify usage of OpenTelemetry.
//
// A nice write-up of using OpenTelemetry (directly) is at https://www.komu.engineer/blogs/11/opentelemetry-and-go
package otel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	olog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
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
	LoggerProvider = olog.LoggerProvider
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
