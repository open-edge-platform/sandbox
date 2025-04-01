// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"os"

	"entgo.io/ent/dialect"
	_ "github.com/lib/pq"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

var zlog = logging.GetLogger("SqlSchemaWriter")

var sqlFileName = flag.String("sqlSchemaFilePath", "sql/inventory.sql", "Path to the SQL schema file")

// This program takes the ent schema and generates equivalent SQL schema.
func main() {
	envPrimary, _, err := util.LookupDBEnv()
	if err != nil {
		zlog.Fatal().Msgf("failed to get DB environment: %v", err)
	}
	atlasDBURLWriter := util.GetDBURL(envPrimary)
	client, err := ent.Open(dialect.Postgres, atlasDBURLWriter)
	if err != nil {
		zlog.Fatal().Msgf("failed connecting to postgresql: %v", err)
	}
	defer client.Close()

	file, err := os.Create(*sqlFileName)
	if err != nil {
		zlog.Fatal().Msgf("Failed to open file %s: %s\n", *sqlFileName, err)
	}
	defer file.Close()

	// Dump migration changes to SQL schema file.
	ctx := context.Background()
	if err := client.Schema.WriteTo(ctx, file); err != nil {
		zlog.Fatal().Msgf("failed printing schema changes: %v", err)
	}
}
