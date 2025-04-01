// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/cenkalti/backoff"
	"github.com/rs/zerolog"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
)

const (
	// Backoff parameters for verifying the connection to the database.
	backoffInterval    = 5 * time.Second
	backoffRetries     = 10
	connectionTimeout  = 10 * time.Second
	databaseDriverName = "pgx"
	// Default values for DB connection parameters.
	defaultDBMaxOpenConns    = 30
	defaultDBMaxIdleConns    = 30
	defaultDBConnMaxLifetime = 0 // By default, never closed due to a connection's age
	defaultDBConnMaxIdleTime = 5 * time.Minute
)

var (
	dbMaxOpenConns    = flag.Int("dbMaxOpenConns", defaultDBMaxOpenConns, "Sets the maximum number of open connections to the DB")
	dbMaxIdleConns    = flag.Int("dbMaxIdleConns", defaultDBMaxIdleConns, "Sets the maximum number of DB connections in idle")
	dbConnMaxLifetime = flag.Duration(
		"dbConnMaxLifetime",
		defaultDBConnMaxLifetime,
		"Sets the maximum amount of time a DB connection may be reused (total lifetime). "+
			" If <= 0, connections are not closed due to a connection's age.",
	)
	dbConnMaxIdleTime = flag.Duration(
		"dbConnMaxIdleTime",
		defaultDBConnMaxIdleTime,
		"Sets the maximum amount of time a DB connection may be idle. "+
			"If <= 0, connections are not closed due to a connection's idle time.",
	)
)

func openDB(databaseURL, debugQueryPrefix string) (dialect.Driver, error) {
	db, err := sql.Open(databaseDriverName, databaseURL)
	if err != nil {
		return nil, err
	}

	// Set configuration parameters for connection pool
	db.SetMaxOpenConns(*dbMaxOpenConns)
	db.SetMaxIdleConns(*dbMaxIdleConns)
	// This sets the max total lifetime of a connection (idle+active time).
	// After this total amount of time, the connection will be closed, and re-created as needed.
	db.SetConnMaxLifetime(*dbConnMaxLifetime)
	db.SetConnMaxIdleTime(*dbConnMaxIdleTime)

	// Verify connectivity to the database with Ping
	if err = backoff.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.TODO(), connectionTimeout)
		defer cancel()
		return db.PingContext(ctx)
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(backoffInterval), backoffRetries)); err != nil {
		zlog.InfraSec().Fatal().Err(err).Msg("Failed to connect to the DB")
	}
	driver := entsql.OpenDB(dialect.Postgres, db)
	if zerolog.GlobalLevel() <= zerolog.DebugLevel {
		return dialect.DebugWithContext(driver, func(ctx context.Context, v ...any) {
			z := zlog.TraceCtx(ctx)
			queryString := fmt.Sprintf("%v", v...)
			z.Debug().Msgf("%s%s", debugQueryPrefix, queryString)
		}), nil
	}
	return driver, nil
}

func openEntClient(writerDatabaseURL, readerDatabaseURL string) (*ent.Client, error) {
	writer, err := openDB(writerDatabaseURL, "")
	if err != nil {
		return nil, err
	}
	if readerDatabaseURL != "" {
		// If readerDatabaseURL is set when we use the multi driver to support multiple DB backends.
		reader, err := openDB(readerDatabaseURL, "RO: ")
		if err != nil {
			return nil, err
		}
		return ent.NewClient(ent.Driver(&multiDriver{writer: writer, reader: reader})), nil
	}
	return ent.NewClient(ent.Driver(writer)), nil
}

// ConnectEntDB creates a ent client with the given database URLs.
// writerDatabaseURL is the primary Database URL, that has write capabilities.
// readerDatabaseURL is the Database URL with write only access (reader replicas).
// If the reader URL is missing, the ent client will use only the writer/primary URL.
func ConnectEntDB(writerDatabaseURL, readerDatabaseURL string) *ent.Client {
	// Open connection to database.
	// Note: this won't properly connect to the database, the connection starts when we do the "WriteTo" call below.
	client, err := openEntClient(writerDatabaseURL, readerDatabaseURL)
	if err != nil {
		zlog.InfraSec().Fatal().Err(err).Msg("failed opening connection to database")
	}

	// Run the connection with retry to allow required services to come-up before we panic.
	b := bytes.Buffer{}
	if err = backoff.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.TODO(), connectionTimeout)
		defer cancel()
		// Open the database connection and create a schema diff. This should
		// always produce an empty diff, as versioned migrations are run before
		// this.
		return client.Schema.WriteTo(ctx, &b)
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(backoffInterval), backoffRetries)); err != nil {
		zlog.InfraSec().Fatal().Err(err).Msg("Failed creating schema resources")
	}

	if b.Len() > 0 {
		zlog.Fatal().Msgf("Unexpected database schema! Diff: %v", b.String())
	}

	return client
}

// Provides a "cluster" driver with multiple actual DB driver.
type multiDriver struct {
	reader, writer dialect.Driver
}

func (d *multiDriver) Query(ctx context.Context, query string, args, v any) error {
	// TODO: provide via context a way to use primary/write driver see example:
	//  https://github.com/ent/ent/issues/1580#issuecomment-968879339
	return d.reader.Query(ctx, query, args, v)
}

func (d *multiDriver) Exec(ctx context.Context, query string, args, v any) error {
	return d.writer.Exec(ctx, query, args, v)
}

func (d *multiDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	return d.writer.Tx(ctx)
}

func (d *multiDriver) BeginTx(ctx context.Context, opts *sql.TxOptions) (dialect.Tx, error) {
	e := d.writer
	if opts != nil && opts.ReadOnly {
		e = d.reader
	}

	txBeginner, ok := e.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	})
	if !ok {
		zlog.Error().Msgf("unexpected type for interface: %T", e)
		return nil, errors.Errorf("unexpected type for interface: %T", e)
	}

	return txBeginner.BeginTx(ctx, opts)
}

func (d *multiDriver) Close() error {
	rerr := d.reader.Close()
	werr := d.writer.Close()
	if rerr != nil {
		return rerr
	}
	if werr != nil {
		return werr
	}
	return nil
}

func (d *multiDriver) Dialect() string {
	return d.reader.Dialect()
}
