package otelsdk

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/UNO-SOFT/otel/otelslog"
	olog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	_ "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	_ "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
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

	Resource = resource.Resource
)

func NewResource(serviceName, serviceVersion string) (*Resource, error) {
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		))
}

// NewTraceProvider returns a TraceProvider which exports using the given traceExporter,
// with defailt 10s BatchSpanProcessor timeout.
func NewTracerProvider(traceExporter sdktrace.SpanExporter, res *Resource, options ...sdktrace.BatchSpanProcessorOption) TracerProvider {
	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter,
			append(append(make([]sdktrace.BatchSpanProcessorOption, 0, 1+len(options)),
				sdktrace.WithBatchTimeout(10*time.Second),
			), options...)...),
		sdktrace.WithResource(res),
	)
}

// NewMeterProvider returns a MeterProvider which exports using the given metricExporter,
// with default 1m PeriodicReader.
func NewMeterProvider(metricExporter sdkmetric.Exporter, res *Resource, options ...sdkmetric.Option) metric.MeterProvider {
	return sdkmetric.NewMeterProvider(
		append(append(make([]sdkmetric.Option, 0, 2+len(options)),
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
				metricExporter, sdkmetric.WithInterval(1*time.Minute),
			)),
			sdkmetric.WithResource(res),
		), options...)...,
	)
}

// NewLoggerProvider returns a log.LoggerProvider which exports using the given loggerExporter,
//
// Such a loggerExporter can be created with otlploghttp, for example.
func NewLoggerProvider(loggerExporter sdklog.Exporter, res *Resource, options ...sdklog.BatchProcessorOption) *sdklog.LoggerProvider {
	return sdklog.NewLoggerProvider(
		sdklog.WithProcessor(
			sdklog.NewBatchProcessor(loggerExporter,
				append(append(make([]sdklog.BatchProcessorOption, 0, 3+len(options)),
					sdklog.WithExportMaxBatchSize(1<<20),
					sdklog.WithExportTimeout(24*time.Hour),
				), options...)...,
			),
		),
		sdklog.WithResource(res),
	)
}

func LoggerEnabled(ctx context.Context, logger olog.Logger) bool {
	return logger.Enabled(ctx, olog.EnabledParameters{})
}

// LogTraceProvider wraps the Logger to as a Provider.
func New(res *Resource) (TracerProvider, MeterProvider, LoggerProvider, func(context.Context) error, error) {
	tbc := make([]interface{ Shutdown(context.Context) error }, 0, 3)
	shutdown := func(ctx context.Context) error {
		errs := make([]error, 0, len(tbc))
		for _, x := range tbc {
			if x != nil {
				if ctx != nil {
					errs = append(errs, x.Shutdown(ctx))
				} else {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					errs = append(errs, x.Shutdown(ctx))
					cancel()
				}
			}
		}
		return errors.Join(errs...)
	}

	te, err := stdouttrace.New()
	if err != nil {
		shutdown(nil)
		return nil, nil, nil, nil, err
	}
	tbc = append(tbc, te)
	me, err := stdoutmetric.New()
	if err != nil {
		shutdown(nil)
		return nil, nil, nil, nil, err
	}
	tbc = append(tbc, me)
	lg, err := stdoutlog.New()
	if err != nil {
		shutdown(nil)
		return nil, nil, nil, nil, err
	}
	tbc = append(tbc, lg)

	return NewTracerProvider(te, res), NewMeterProvider(me, res), NewLoggerProvider(lg, res), shutdown, nil
}

// SetupOTLP returns an slog.Handler and a shutdown function,
// iff OTEL_EXPORTER_OTLP_LOGS_ENDPOINT is specified.
//
// VL_ACCOUNT_ID+VL_PROJECT_ID or VL_TENANT_ID is used for providing henaders (AccountID, ProjectID) for VictoriaLogs.
func SetupOTLP(ctx context.Context, serviceNameAtVersion string) (otelslogHandlerProvider, func(context.Context), error) {
	logsURL := os.Getenv("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT")
	if logsURL == "" {
		return otelslogHandlerProvider{}, nil, nil
	}
	serviceName, serviceVersion, _ := strings.Cut(serviceNameAtVersion, "@")
	resource, err := NewResource(serviceName, serviceVersion)
	if err != nil {
		return otelslogHandlerProvider{}, nil, err
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
		return otelslogHandlerProvider{}, nil, err
	}
	lp := NewLoggerProvider(logExporter, resource)
	return otelslogHandlerProvider{
		LoggerProvider: lp,
		Handler:        otelslog.NewHandler(serviceName, otelslog.WithLoggerProvider(lp)),
	}, func(context.Context) { lp.Shutdown(ctx); logExporter.Shutdown(ctx) }, nil
}

type otelslogHandlerProvider struct {
	slog.Handler
	LoggerProvider
}
