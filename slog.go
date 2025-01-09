// Copyright 2021, 2025 Tamás Gulácsi
//
//
// SPDX-License-Identifier: Apache-2.0

package otel

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/log"
)

func NewSLogHandler(logger log.Logger) slog.Handler {
	return slogHandler{logger: logger}
}

type slogHandler struct {
	logger log.Logger
	attrs  []log.KeyValue
	group  string
}

func (h slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Convert early, convert once
	as := make([]log.KeyValue, 0, len(attrs))
	for _, a := range attrs {
		if kv := attrAttr(a); kv.Key != "" {
			as = append(as, kv)
		}
	}
	return slogHandler{logger: h.logger, attrs: append(h.attrs, as...), group: h.group}
}

func (h slogHandler) WithGroup(group string) slog.Handler {
	if group == "" {
		return h
	}
	g := h.group
	if g != "" {
		g = h.group + "." + group
	}
	return slogHandler{logger: h.logger, attrs: h.attrs, group: g}
}

func (h slogHandler) Handle(ctx context.Context, record slog.Record) error {
	// fmt.Printf("slogHandler.Handle(%+v)\n", record)
	var rec log.Record
	rec.SetBody(log.StringValue(record.Message))
	rec.SetObservedTimestamp(record.Time)
	rec.SetTimestamp(record.Time)
	rec.SetSeverity(LevelSeverity(record.Level))
	if h.group != "" {
		rec.AddAttributes(log.String("group", h.group))
	}
	if len(h.attrs) != 0 {
		rec.AddAttributes(h.attrs...)
	}
	record.Attrs(func(a slog.Attr) bool {
		if kv := attrAttr(a); kv.Key != "" {
			rec.AddAttributes(kv)
		}
		return true
	})
	h.logger.Emit(ctx, rec)
	// fmt.Printf("h.%v.Emit(%+v)\n", h.logger, rec)
	return nil
}

func (h slogHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.logger.Enabled(ctx, log.EnabledParameters{Severity: LevelSeverity(lvl)})
}

func LevelSeverity(lvl slog.Level) log.Severity {
	switch lvl {
	case slog.LevelDebug:
		return log.SeverityDebug
	case slog.LevelInfo:
		return log.SeverityInfo
	case slog.LevelWarn:
		return log.SeverityWarn
	case slog.LevelError:
		return log.SeverityError
	}
	return log.SeverityUndefined
}

func attrAttr(a slog.Attr) log.KeyValue {
	var kv log.KeyValue
	switch a.Value.Kind() {
	case slog.KindBool:
		kv = log.Bool(a.Key, a.Value.Bool())
	case slog.KindDuration:
		kv = log.String(a.Key, a.Value.Duration().String())
	case slog.KindFloat64:
		kv = log.Float64(a.Key, a.Value.Float64())
	case slog.KindInt64:
		kv = log.Int64(a.Key, a.Value.Int64())
	case slog.KindString:
		kv = log.String(a.Key, a.Value.String())
	case slog.KindTime:
		kv = log.String(a.Key, a.Value.Time().Format(time.RFC3339))
	case slog.KindUint64:
		kv = log.Int64(a.Key, int64(a.Value.Uint64()))
	default:
		kv = log.String(a.Key, fmt.Sprintf("%v", a.Value))
	}
	return kv
}
