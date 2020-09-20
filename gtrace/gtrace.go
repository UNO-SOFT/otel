// Copyright 2020 Tamás Gulácsi
//
//
// SPDX-License-Identifier: Apache-2.0

package gtrace

import (
	"context"

	grpctrace "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func Extract(ctx context.Context, metadata *metadata.MD, opts ...grpctrace.Option) ([]label.KeyValue, trace.SpanContext) {
	return grpctrace.Extract(ctx, metadata, opts...)
}

func Inject(ctx context.Context, metadata *metadata.MD, opts ...grpctrace.Option) {
	grpctrace.Inject(ctx, metadata, opts...)
}
func StreamClientInterceptor(tracer trace.Tracer, opts ...grpctrace.Option) grpc.StreamClientInterceptor {
	return grpctrace.StreamClientInterceptor(tracer, opts...)
}
func StreamServerInterceptor(tracer trace.Tracer, opts ...grpctrace.Option) grpc.StreamServerInterceptor {
	return grpctrace.StreamServerInterceptor(tracer, opts...)
}

func UnaryClientInterceptor(tracer trace.Tracer, opts ...grpctrace.Option) grpc.UnaryClientInterceptor {
	return grpctrace.UnaryClientInterceptor(tracer, opts...)
}
func UnaryServerInterceptor(tracer trace.Tracer, opts ...grpctrace.Option) grpc.UnaryServerInterceptor {
	return grpctrace.UnaryServerInterceptor(tracer, opts...)
}
