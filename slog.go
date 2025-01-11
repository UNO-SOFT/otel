// Copyright 2021, 2025 Tamás Gulácsi
//
//
// SPDX-License-Identifier: Apache-2.0

package otel

import (
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

func NewSLogHandler(name string, lp *LoggerProvider, options ...otelslog.Option) slog.Handler {
	if lp != nil {
		options = append(options, otelslog.WithLoggerProvider(lp))
	}
	return otelslog.NewHandler(name, options...)
}

func NewSLogger(name string, lp *LoggerProvider, options ...otelslog.Option) *slog.Logger {
	if lp != nil {
		options = append(options, otelslog.WithLoggerProvider(lp))
	}
	return otelslog.NewLogger(name, options...)
}

func WithSchemaURL(schemaURL string) otelslog.Option { return otelslog.WithSchemaURL(schemaURL) }
func WithSource(source bool) otelslog.Option         { return otelslog.WithSource(source) }
func WithVersion(version string) otelslog.Option     { return otelslog.WithVersion(version) }
