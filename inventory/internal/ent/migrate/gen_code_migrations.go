//go:build exclude

// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os/signal"
	"syscall"

	atlas "ariga.io/atlas/sql/migrate"
	_ "github.com/lib/pq"
	"golang.org/x/exp/slices"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/migrate/code_migrations"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var zlog = logging.GetLogger("migrate")

// This file is the driver for generating database migrations from Go code. It
// is run as part of the "migration-generate" make target.
// Instructions for writing a new code-driven migration:
//   - Create a new code-driven migration, copy code_migrations/example.go for a quick start.
//   - Add your migration to the `codeDrivenMigration` slice below.
//   - Run the make target.
//   - Commit the generated SQL code, updated atlas.sum, your migration and this file.

type codeDrivenMigration interface {
	// Name is the unique identifier for this migration. It is used to generate a
	// file name and ensure that this migration is only generated once. Avoid using
	// spaces or other characters that could cause problems when used as file names.
	// All lowercase and underscores are recommended.
	Name() string

	// Do runs the actual migration.
	Do(context.Context, *atlas.LocalDir) error
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	// Find the local migration directory containing the Atlas migration files.
	dir, err := atlas.NewLocalDir("internal/ent/migrate/migrations/")
	if err != nil {
		zlog.Fatal().Err(err).Msg("failed creating atlas migration directory")
	}
	fs, err := dir.Files()
	if err != nil {
		zlog.Fatal().Err(err).Msg("failed reading files in atlas migration directory")
	}
	seenMigrations := []string{}
	for _, f := range fs {
		// Note: f.Desc() returns the name, while f.Name() returns the file name.
		seenMigrations = append(seenMigrations, f.Desc())
	}

	// Put your new data migration here.
	ms := []codeDrivenMigration{
		code_migrations.ExampleMigration{},
	}

	// Generate the code migrations, skipping existing ones.
	for _, m := range ms {
		if slices.Contains(seenMigrations, m.Name()) {
			zlog.Info().Msgf("Skipped already run migration '%v'", m.Name())
			continue
		}
		err = m.Do(ctx, dir)
		if err != nil {
			zlog.Fatal().Err(err).Msgf("failed generating migration '%v'", m.Name())
		}
	}
}
