// Copyright 2020 Tamás Gulácsi
//
//
// SPDX-License-Identifier: Apache-2.0

package grpctrace

import (
	"context"

	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/trace"
	gtrace "go.opentelemetry.io/otel/instrumentation/grpctrace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func Extract(ctx context.Context, metadata *metadata.MD, opts ...gtrace.Option) ([]kv.KeyValue, trace.SpanContext) {
	return gtrace.Extract(ctx, metadata, opts...)
}

func Inject(ctx context.Context, metadata *metadata.MD, opts ...gtrace.Option) {
	gtrace.Inject(ctx, metadata, opts...)
}
func StreamClientInterceptor(tracer trace.Tracer, opts ...gtrace.Option) grpc.StreamClientInterceptor {
	return gtrace.StreamClientInterceptor(tracer, opts...)
}
func StreamServerInterceptor(tracer trace.Tracer, opts ...gtrace.Option) grpc.StreamServerInterceptor {
	return gtrace.StreamServerInterceptor(tracer, opts...)
}

func UnaryClientInterceptor(tracer trace.Tracer, opts ...gtrace.Option) grpc.UnaryClientInterceptor {
	return gtrace.UnaryClientInterceptor(tracer, opts...)
}
func UnaryServerInterceptor(tracer trace.Tracer, opts ...gtrace.Option) grpc.UnaryServerInterceptor {
	return gtrace.UnaryServerInterceptor(tracer, opts...)
}
