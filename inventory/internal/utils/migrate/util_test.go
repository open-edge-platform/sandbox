// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package migrate_test

import (
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/utils/migrate"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

func TestMain(m *testing.M) {
	flag.String(
		"policyBundle",
		"/rego/policy_bundle.tar.gz",
		"Path of policy rego file",
	)
	flag.Parse()
	os.Exit(m.Run()) // run all tests
}

func TestRunAtlasMigrations(t *testing.T) {
	dbURL := util.GetDBURL(util.LookupDBTestEnv())
	t.Run("ValidMigrationsGiven", func(t *testing.T) {
		path, err := os.Getwd()
		require.NoError(t, err)
		migrationsDir := filepath.Dir(path) + "/../ent/migrate/migrations"
		if out, err := migrate.RunAtlasMigrations(dbURL, migrationsDir); err != nil {
			t.Fatalf("Database migration failed. Aborting. Atlas output: %v", string(out))
		} else {
			t.Log(string(out))
		}
	})
	t.Run("NoMigrationDirGivenFail", func(t *testing.T) {
		out, err := migrate.RunAtlasMigrations(dbURL, "")
		assert.Error(t, err)
		assert.Contains(t, string(out), "No migrations directory given")
	})
}

func TestLookupMigrationEnv(t *testing.T) {
	t.Run("MissingOrgIdEnv", func(t *testing.T) {
		t.Setenv("MIGRATION_PROJECT_ID", "orgID")

		env, exists := migrate.LookupMigrationEnv()
		assert.False(t, exists)
		assert.Empty(t, env)
	})
	t.Run("MissingProjectIdEnv", func(t *testing.T) {
		t.Setenv("MIGRATION_ORG_ID", "prjID")

		env, exists := migrate.LookupMigrationEnv()
		assert.False(t, exists)
		assert.Empty(t, env)
	})
	t.Run("ExisitingMigrationEnv", func(t *testing.T) {
		migrEnv := migrate.MigrationEnv{
			OrgID:     "orgID",
			ProjectID: "prjID",
		}

		t.Setenv("MIGRATION_ORG_ID", migrEnv.OrgID)
		t.Setenv("MIGRATION_PROJECT_ID", migrEnv.ProjectID)

		env, exists := migrate.LookupMigrationEnv()
		assert.True(t, exists)
		assert.True(t, reflect.DeepEqual(migrEnv, *env))
	})
}

func TestPopulateTenantValues(t *testing.T) {
	dbURL := util.GetDBURL(util.LookupDBTestEnv())
	tenantID := "11111111-1111-1111-1111-111111111111"

	err := migrate.PopulateTenantValues(dbURL, dbURL, tenantID)
	assert.NoError(t, err)
}
