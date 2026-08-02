// Copyright 2026 Tamás Gulácsi. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package otelsql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"log/slog"
	"time"

	"go.nhat.io/otelsql"

	"github.com/godror/godror"
	"go.opentelemetry.io/otel/semconv/v1.41.0"

	_ "github.com/godror/godror"
)

// RegisterDriver registers the driver and returns a (unique) driver name to be used for opening the connection.
func RegisterDriver(driverName string, options ...otelsql.DriverOption) (string, error) {
	return otelsql.Register(driverName, options...)
}

// Open as sql.Open, but registers the driver for OpenTelemetry instrumentation.
func Open(driverName, dsn string) (*sql.DB, error) {
	options := []otelsql.DriverOption{
		otelsql.AllowRoot(),
		otelsql.TraceQueryWithoutArgs(),
		otelsql.TraceRowsClose(),
		otelsql.TraceRowsAffected(),
	}
	drv, err := RegisterDriver(driverName, options...)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(drv, dsn)
	if err != nil {
		return nil, err
	}
	var domain, name, instance, service string
	if _, ok := db.Driver().(interface {
		ClientVersion() godror.VersionInfo
	}); ok {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := godror.Raw(ctx, db, func(conn godror.Conn) error {
			// if sver, err := conn.ServerVersion(); err != nil {
			// 	slog.Warn("ServerVersion", "error", err)
			// } else {
			const qry = `SELECT
				SYS_CONTEXT('USERENV', 'DB_DOMAIN'),
				SYS_CONTEXT('USERENV', 'DB_NAME'),
				SYS_CONTEXT('USERENV', 'INSTANCE_NAME'),
				SYS_CONTEXT('USERENV', 'SERVICE_NAME')
				FROM DUAL`
			if stmt, err := conn.PrepareContext(ctx, qry); err != nil {
				slog.Warn("BeginTx", "error", err)
			} else {
				defer stmt.Close()
				if rows, err := stmt.(driver.StmtQueryContext).QueryContext(ctx, nil); err != nil {
					slog.Warn("Query", "qry", qry, "error", err)
				} else {
					defer rows.Close()
					if err = rows.Next([]driver.Value{&domain, &name, &instance, &service}); err != nil {
						slog.Warn("rows.Next", "qry", qry, "error", err)
					} else {
					}
				}
			}
			// }
			return nil
		}); err != nil {
			slog.Warn("instantiate otel", "error", err)
		} else if drv, err := RegisterDriver(driverName, append(options,
			otelsql.WithDatabaseName(name),
			otelsql.WithInstanceName(instance),
			otelsql.WithSystem(semconv.DBSystemNameOracleDB),
			// semconv.OracleDBDomain(domain),
			// semconv.OracleDBName(name),
			// semconv.OracleDBInstanceName(instance),
			// semconv.OracleDBService(service),
			// semconv.ServiceVersion(sver.String()),
		)...); err != nil {
			slog.Warn("RegisterDriver2", "error", err)
		} else {
			db.Close()
			return sql.Open(drv, dsn)
		}
	}
	return db, err
}
