// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	embed "embed"
)

const (
	MigrationsDir            = "migrationsDir"
	MigrationsDirDescription = "Path to the DB migrations directory. Cannot be empty."
)

var (
	//go:embed *.sql
	//go:embed atlas.sum
	MigrationsFolder embed.FS
)
