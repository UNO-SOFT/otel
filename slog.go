// Copyright 2021, 2025 Tamás Gulácsi
//
//
// SPDX-License-Identifier: Apache-2.0

package otel

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
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

func LogWithSchemaURL(schemaURL string) otelslog.Option { return otelslog.WithSchemaURL(schemaURL) }
func LogWithSource(source bool) otelslog.Option         { return otelslog.WithSource(source) }
func LogWithVersion(version string) otelslog.Option     { return otelslog.WithVersion(version) }

// SetupOTLP returns an slog.Handler and a shutdown function,
// iff OTEL_EXPORTER_OTLP_LOGS_ENDPOINT is specified.
//
// VL_ACCOUNT_ID+VL_PROJECT_ID or VL_TENANT_ID is used for providing henaders (AccountID, ProjectID) for VictoriaLogs.
func SetupOTLP(ctx context.Context, serviceName, version string) (slog.Handler, func(context.Context), error) {
	logsURL := os.Getenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT")
	if logsURL == "" {
		return nil, nil, nil
	}
	resource, err := NewResource(serviceName, version)
	if err != nil {
		return nil, nil, err
	}
	opts := []otlploghttp.Option{otlploghttp.WithEndpointURL(logsURL), nil, nil}[:1]
	if acc, proj := os.Getenv("VL_ACCOUNT_ID"), os.Getenv("VL_PROJECT_ID"); acc != "" && proj != "" {
		opts = append(opts, otlploghttp.WithHeaders(map[string]string{"AccountID": acc, "ProjectID": proj}))
	} else if acc, proj, ok := strings.Cut(os.Getenv("VL_TENANT_ID"), ":"); ok && acc != "" && proj != "" {
		opts = append(opts, otlploghttp.WithHeaders(map[string]string{"AccountID": acc, "ProjectID": proj}))
	}
	if os.Getenv("OTEL_EXPORTER_OTLP_TIMEOUT") == "" {
		opts = append(opts, otlploghttp.WithTimeout(time.Hour))
	}
	logExporter, err := otlploghttp.New(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}
	lp := NewLoggerProvider(logExporter, resource)
	return NewSLogHandler(serviceName, lp), func(context.Context) { lp.Shutdown(ctx); logExporter.Shutdown(ctx) }, nil
}
