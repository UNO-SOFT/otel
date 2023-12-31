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
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
)

func Extract(ctx context.Context, metadata *metadata.MD, opts ...grpctrace.Option) (baggage.Baggage, trace.SpanContext) {
	return grpctrace.Extract(ctx, metadata, opts...)
}

func Inject(ctx context.Context, metadata *metadata.MD, opts ...grpctrace.Option) {
	grpctrace.Inject(ctx, metadata, opts...)
}

// StreamClientInterceptor
//
// Deprecated: use ClientInterceptor instead.
func StreamClientInterceptor(opts ...grpctrace.Option) grpc.StreamClientInterceptor {
	return grpctrace.StreamClientInterceptor(opts...)
}

// StreamServerInterceptor
//
// Deprecated: use ServerInterceptor instead
func StreamServerInterceptor(opts ...grpctrace.Option) grpc.StreamServerInterceptor {
	return grpctrace.StreamServerInterceptor(opts...)
}

// UnaryClientInterceptor
//
// Deprecated: use ClientInterceptor instead
func UnaryClientInterceptor(opts ...grpctrace.Option) grpc.UnaryClientInterceptor {
	return grpctrace.UnaryClientInterceptor(opts...)
}

// UnaryServerInterceptor
//
// Deprecated: use ServerInterceptor instead
func UnaryServerInterceptor(opts ...grpctrace.Option) grpc.UnaryServerInterceptor {
	return grpctrace.UnaryServerInterceptor(opts...)
}

func ClientInterceptor(opts ...grpctrace.Option) stats.Handler {
	return grpctrace.NewClientHandler(opts...)
}

func ServerInterceptor(opts ...grpctrace.Option) stats.Handler {
	return grpctrace.NewServerHandler(opts...)
}

func WithPropagators(p propagation.TextMapPropagator) grpctrace.Option {
	return grpctrace.WithPropagators(p)
}

func WithTracerProvider(tp trace.TracerProvider) grpctrace.Option {
	return grpctrace.WithTracerProvider(tp)
}
