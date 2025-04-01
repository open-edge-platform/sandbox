// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package migrate

import (
	"context"
	"os"
	"os/exec"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/endpointresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostgpuresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostnicresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hoststorageresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostusbresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/instanceresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/ipaddressresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/netlinkresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/networksegment"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/operatingsystemresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/ouresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/providerresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/regionresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/remoteaccessconfiguration"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/repeatedscheduleresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/singlescheduleresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/telemetrygroupresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/telemetryprofile"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/workloadmember"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/workloadresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/store"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var zlog = logging.GetLogger("migrate")

const (
	migrationTimeout = 60 * time.Second
)

// RunAtlasMigrations attempts to migrate the given database to the latest
// schema with the provided migration files. On success, the output of the atlas
// tool is returned as byte slice. On errors, an error will we returned and the
// byte slice might contain a diagnostic message. The state of the database is
// undefined at this point and will most likely require human intervention.
func RunAtlasMigrations(dbURL, migrationsDir string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	if migrationsDir == "" {
		const errMsg = "No migrations directory given. Is the `-migrationsDir` flag set?"
		return []byte(errMsg), errors.Errorfc(codes.InvalidArgument, errMsg)
	}

	args := []string{
		"migrate",
		"apply",
		"--dir", "file://" + migrationsDir,
		"--url", dbURL,
		"--baseline", "20230600000000",
	}
	zlog.Debug().Msgf("Prepared Atlas command: %v %v", "atlas", args)
	out, err := exec.CommandContext(ctx, "atlas", args...).CombinedOutput()
	zlog.Debug().Msgf("Atlas output: %s", string(out))

	if err != nil {
		zlog.InfraSec().InfraErr(err).Msg("Atlas migration failed.")
		err = errors.Wrap(err)
	} else {
		zlog.Debug().Msgf("Atlas migration successful.")
	}

	return out, err
}

type MigrationEnv struct {
	OrgID     string
	ProjectID string
}

// LookupMigrationEnv fetches the migration environment variables provided via k8s ConfigMap.
func LookupMigrationEnv() (*MigrationEnv, bool) {
	const (
		orgID = "MIGRATION_ORG_ID"
		prjID = "MIGRATION_PROJECT_ID"
	)

	env := &MigrationEnv{}
	var ok bool
	if env.OrgID, ok = os.LookupEnv(orgID); !ok {
		zlog.Debug().Msgf("%s env var is not set", orgID)
		return nil, false
	}
	if env.ProjectID, ok = os.LookupEnv(prjID); !ok {
		zlog.Debug().Msgf("%s env var is not set", prjID)
		return nil, false
	}
	return env, true
}

// PopulateTenantValues updates the tenant_id column in the defined tables,
// setting it to the specified tenantID where the current value is '0'.
// This function is useful for ensuring that tenant IDs are correctly populated
// after a data migration.
func PopulateTenantValues(dbURLWriter, dbURLReader, tenantID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	tables := []string{
		endpointresource.Table,
		hostresource.Table,
		hostgpuresource.Table,
		hostnicresource.Table,
		hoststorageresource.Table,
		hostusbresource.Table,
		instanceresource.Table,
		ipaddressresource.Table,
		netlinkresource.Table,
		networksegment.Table,
		operatingsystemresource.Table,
		ouresource.Table,
		providerresource.Table,
		regionresource.Table,
		remoteaccessconfiguration.Table,
		repeatedscheduleresource.Table,
		singlescheduleresource.Table,
		siteresource.Table,
		telemetrygroupresource.Table,
		telemetryprofile.Table,
		workloadmember.Table,
		workloadresource.Table,
	}

	invstore := store.NewStore(dbURLWriter, dbURLReader)
	defer func() {
		err := invstore.CloseEntClient()
		if err != nil {
			zlog.InfraSec().InfraErr(err).Msg("Error closing DB connection")
		}
	}()

	if err := invstore.UpdateDefaultTenantIDInTables(ctx, tables, tenantID); err != nil {
		zlog.InfraSec().InfraErr(err).Msg("Error populating tenant values")
		return errors.Wrap(err)
	}
	return nil
}
