// Copyright 2020, 2023 Tamás Gulácsi
//
//
// SPDX-License-Identifier: Apache-2.0

package gtrace

import (
	"context"

	grpctrace "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
)

func Extract(ctx context.Context, metadata *metadata.MD, opts ...grpctrace.Option) (baggage.Baggage, trace.SpanContext) {
	return grpctrace.Extract(ctx, metadata, opts...)
}

func Inject(ctx context.Context, metadata *metadata.MD, opts ...grpctrace.Option) {
	grpctrace.Inject(ctx, metadata, opts...)
}

// ClientHandler instead of {Stream,Unary}ClientInterceptor - use with grpc.WithStatsHandler
func ClientHandler(opts ...grpctrace.Option) stats.Handler {
	return grpctrace.NewClientHandler(opts...)
}

// ServerHandler instead of {Stream,Unary}ServerInterceptor - use with grpc.StatsHandler
func ServerHandler(opts ...grpctrace.Option) stats.Handler {
	return grpctrace.NewServerHandler(opts...)
}

func WithPropagators(p propagation.TextMapPropagator) grpctrace.Option {
	return grpctrace.WithPropagators(p)
}

func WithTracerProvider(tp trace.TracerProvider) grpctrace.Option {
	return grpctrace.WithTracerProvider(tp)
}
