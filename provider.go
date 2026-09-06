// Copyright 2021, 2026 Tamás Gulácsi
//
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package otel

import (
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/propagation"
)

// LogRecord is log.Record
type LogRecord = log.Record

// https://opentelemetry.io/docs/instrumentation/go/getting-started/

func NewPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}
