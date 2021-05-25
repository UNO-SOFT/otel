// Copyright 2020 Tamás Gulácsi
//
//
// SPDX-License-Identifier: Apache-2.0

package gtrace

import (
	"context"

	grpctrace "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func Extract(ctx context.Context, metadata *metadata.MD, opts ...grpctrace.Option) ([]attribute.KeyValue, trace.SpanContext) {
	return grpctrace.Extract(ctx, metadata, opts...)
}

func Inject(ctx context.Context, metadata *metadata.MD, opts ...grpctrace.Option) {
	grpctrace.Inject(ctx, metadata, opts...)
}
func StreamClientInterceptor(opts ...grpctrace.Option) grpc.StreamClientInterceptor {
	return grpctrace.StreamClientInterceptor(opts...)
}
func StreamServerInterceptor(opts ...grpctrace.Option) grpc.StreamServerInterceptor {
	return grpctrace.StreamServerInterceptor(opts...)
}

func UnaryClientInterceptor(opts ...grpctrace.Option) grpc.UnaryClientInterceptor {
	return grpctrace.UnaryClientInterceptor(opts...)
}
func UnaryServerInterceptor(opts ...grpctrace.Option) grpc.UnaryServerInterceptor {
	return grpctrace.UnaryServerInterceptor(opts...)
}

func WithPropagators(p propagation.TextMapPropagator) grpctrace.Option {
	return grpctrace.WithPropagators(p)
}

func WithTracerProvider(tp trace.TracerProvider) grpctrace.Option {
	return grpctrace.WithTracerProvider(tp)
}
