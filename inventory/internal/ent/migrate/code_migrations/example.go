// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package code_migrations

import (
	"context"
	"fmt"

	"ariga.io/atlas/sql/migrate"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostresource"
)

// Define an exported struct that satisfies the `codeDrivenMigration` interface.
type ExampleMigration struct{}

func (ExampleMigration) Name() string {
	return "example" // Must be unique among all migrations!
}

func (e ExampleMigration) Do(ctx context.Context, dir *migrate.LocalDir) error {
	// Get a new writer and client interface.
	w := &schema.DirWriter{Dir: dir}
	client := ent.NewClient(ent.Driver(schema.NewWriteDriver(dialect.Postgres, w)))

	// In this example we want to back-fill all empty hosts' names with the
	// default value 'Unknown'. While the client interface is the same as in
	// actual inventory code, you cannot read any database content or depend on
	// it in your queries. Remember, this code is not run against a live database,
	// but rather used to describe a SQL statement with the familiar Ent interface.
	err := client.HostResource.
		Update().
		Where(
			hostresource.NameEQ(""),
		).
		SetName("Unknown").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed generating statement: %w", err)
	}

	// Write the content to the migration directory.
	// Uncomment this in your migration!
	// return w.FlushChange(
	// 	e.Name(),
	// 	"Backfill all empty host names with default value 'unknown'.",
	// )
	return nil
}
