// Copyright 2026 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package otelslog

import (
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/log"
)

type (
	Handler = otelslog.Handler
	Option  = otelslog.Option
)

// AddHandler is a convenience wrapper for slog.NewMultiHandler
func AddHandler(logger *slog.Logger, hndl slog.Handler) *slog.Logger {
	return slog.New(slog.NewMultiHandler(logger.Handler(), hndl))
}

// NewLogger creates an otelslog.Logger.
func NewLogger(name string, options ...Option) *slog.Logger {
	return otelslog.NewLogger(name, options...)
}

// NewHandler creates an otelslog.Handler
func NewHandler(name string, options ...Option) *Handler {
	return otelslog.NewHandler(name, options...)
}

// func WithAttributes(attributes ...attribute.KeyValue) Option { return otelslog.WithAttributes(attributes...)}

func WithLoggerProvider(provider log.LoggerProvider) Option {
	return otelslog.WithLoggerProvider(provider)
}
func WithSchemaURL(schemaURL string) Option { return otelslog.WithSchemaURL(schemaURL) }
func WithSource(source bool) Option         { return otelslog.WithSource(source) }
func WithVersion(version string) Option     { return otelslog.WithVersion(version) }
