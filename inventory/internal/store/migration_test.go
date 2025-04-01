// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/regionresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/store"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	provider_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/provider/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

const migrationFilePath = "../ent/migrate/migrations/20241021072611_create-license-providers.sql"

type provider struct {
	baremetalProvider       *provider_v1.ProviderResource
	expectedBareMetalConfig string
}

//nolint:cyclop,funlen // long and complex test function due to use of raw client
func TestUpdateTenantIDInTables(t *testing.T) {
	// For testing purposes we use the same URL for both writer and reader
	dbURL := util.GetDBURL(util.LookupDBTestEnv())
	invstore := store.NewStore(dbURL, dbURL)
	defer func() {
		err := invstore.CloseEntClient()
		assert.NoError(t, err)
	}()

	type args struct {
		ctx              context.Context
		tables           []string
		expectedTenantID string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Happy path - tenantID is updated",
			args: args{
				ctx: context.Background(),
				tables: []string{
					regionresource.Table,
					hostresource.Table,
				},
				expectedTenantID: "11111111-1111-1111-1111-111111111111",
			},
			wantErr: false,
		},
		{
			name: "TenantID is not updated due to one non-existent table in the table list",
			args: args{
				ctx: context.Background(),
				tables: []string{
					regionresource.Table,
					"bad-table-name",
					hostresource.Table,
				},
				expectedTenantID: "0",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regionIDZeroTID := "region-12345678"
			regionIDNonZeroTID := "region-87654321"
			tenantIDNonZero := "1234-5678-9012-3456"
			ctx := context.Background()

			// Create region with Raw client, in cleanup we need to take care about deleting them via the raw client as well.
			err := store.ExecuteInTx(invstore)(ctx, func(ctx context.Context, tx *ent.Tx) error {
				insertRegion := func(ctx context.Context, tx *ent.Tx, tenantID, resourceID string) error {
					nowString := time.Now().UTC().Format("2006-01-02 15:04:05.999")
					query := fmt.Sprintf("INSERT INTO %s (%s, %s, %s, %s) VALUES ($1, $2, $3, $4)",
						regionresource.Table, regionresource.FieldTenantID, regionresource.FieldResourceID,
						regionresource.FieldCreatedAt, regionresource.FieldUpdatedAt)
					if _, err := tx.ExecContext(ctx, query, tenantID, resourceID, nowString, nowString); err != nil {
						return err
					}
					return nil
				}
				// Create a region with a tenantID "zero" (default tenant ID during migration).
				if err := insertRegion(ctx, tx, "0", regionIDZeroTID); err != nil {
					return err
				}
				// Create a region with a non-zero tenantID
				if err := insertRegion(ctx, tx, tenantIDNonZero, regionIDNonZeroTID); err != nil {
					return err
				}
				return nil
			})
			require.NoError(t, err)

			// Cleanup using raw client.
			t.Cleanup(func() {
				err = store.ExecuteInTx(invstore)(ctx, func(ctx context.Context, tx *ent.Tx) error {
					query := fmt.Sprintf("DELETE FROM %s WHERE %s IN ($1, $2)",
						regionresource.Table, regionresource.FieldResourceID)
					if _, qerr := tx.ExecContext(ctx, query, regionIDZeroTID, regionIDNonZeroTID); qerr != nil {
						return qerr
					}
					return nil
				})
				require.NoError(t, err)
			})

			err = invstore.UpdateDefaultTenantIDInTables(tt.args.ctx, tt.args.tables, tt.args.expectedTenantID)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateTenantIDInTables() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Use raw client to get the region resources, otherwise if tenantID is invalid it might fail validate rules
			err = store.ExecuteInTx(invstore)(ctx, func(ctx context.Context, tx *ent.Tx) error {
				checkTenantID := func(resourceID, expectedTenantID string) error {
					query := fmt.Sprintf("SELECT %s FROM %s WHERE resource_id = $1",
						regionresource.FieldTenantID, regionresource.Table)
					resp, qerr := tx.QueryContext(ctx, query, resourceID)
					if qerr != nil {
						return qerr
					}
					defer resp.Close()
					for resp.Next() {
						var tenantID sql.NullString
						if serr := resp.Scan(&tenantID); serr != nil {
							return serr
						}
						assert.Equal(t, expectedTenantID, tenantID.String)
					}
					return nil
				}
				err = checkTenantID(regionIDNonZeroTID, tenantIDNonZero)
				if err != nil {
					return err
				}
				err = checkTenantID(regionIDZeroTID, tt.args.expectedTenantID)
				if err != nil {
					return err
				}
				return nil
			})
			assert.NoError(t, err)
		})
	}
}

//nolint:funlen // long test function due to test cases
func Test_MigrateProviders(t *testing.T) {
	t.Skip("These test doesn't make sense anymore, because we cannot apply a single migration in isolation")
	dbURL := util.GetDBURL(util.LookupDBTestEnv())
	c := store.ConnectEntDB(dbURL, "")
	defer c.Close()

	t.Run("MigrateNoProviders", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// get all providers before migration
		findQueryRes, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx,
			&inv_v1.ResourceFilter{Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Provider{}}})
		require.NoError(t, err, "find request has rejected")
		// check if no providers are created
		require.Len(t, findQueryRes.GetResources(), 0,
			"find request has returned unexpected number of resources")

		// read migration queries
		sqlMigrationQueries, err := os.ReadFile(migrationFilePath)
		require.NoError(t, err)

		// migrate providers
		_, err = c.ExecContext(ctx, string(sqlMigrationQueries))
		require.NoError(t, err)

		// get all providers after migration
		findQueryRes, err = inv_testing.TestClients[inv_testing.APIClient].Find(ctx,
			&inv_v1.ResourceFilter{Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Provider{}}})
		require.NoError(t, err, "find request has rejected")
		// confirm no providers are created
		require.Len(t, findQueryRes.GetResources(), 0,
			"find request has returned unexpected number of resources")
	})

	t.Run("ConfirmOnlyFmOnboardingProviderMigration", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		providers := map[string]provider{
			"fm_onboarding": {
				baremetalProvider: &provider_v1.ProviderResource{
					ProviderKind:   provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL,
					ProviderVendor: provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LXCA,
					Name:           "fm_onboarding",
					ApiEndpoint:    "192.168.201.3/discovery",
					ApiCredentials: []string{"test-1", "test-2"},
					Config:         "{\"defaultOs\":\"'os-11111111'\",\"autoProvision\":false}",
					TenantId:       "00000000-0000-0000-0000-000000000000",
				},
				expectedBareMetalConfig: "{\"defaultOs\" : \"os-11111111\", \"autoProvision\" : \"false\"}",
			},
			"lenovo_provider": {
				baremetalProvider: &provider_v1.ProviderResource{
					ProviderKind:   provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL,
					ProviderVendor: provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LXCA,
					Name:           "lenovo_provider",
					ApiEndpoint:    "192.168.201.4/discovery",
					ApiCredentials: []string{"test-3", "test-4"},
					Config: "{\"defaultOs\":\"'os-22222222'\",\"autoProvision\":true," +
						"\"customerID\":\"'testCustomID2'\",\"enProductKeyIDs\":\"'testProdID2'\"}",
					TenantId: "00000000-0000-0000-0000-000000000000",
				},
			},
		}

		// create providers to be migrated
		for _, prov := range providers {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Provider{Provider: prov.baremetalProvider},
			}

			// create
			cprovResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
			require.NoError(t, err)
			prov.baremetalProvider = cprovResp.GetProvider()
			assertSameResource(t, createresreq, cprovResp, nil)
		}

		// get all providers before migration
		findQueryRes, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx,
			&inv_v1.ResourceFilter{Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Provider{}}})
		require.NoError(t, err, "find request has rejected")
		// check if all providers are created
		require.Len(t, findQueryRes.GetResources(), len(providers),
			"find request has returned unexpected number of resources")

		// read migration queries
		sqlMigrationQueries, err := os.ReadFile(migrationFilePath)
		require.NoError(t, err)

		// migrate providers
		_, err = c.ExecContext(ctx, string(sqlMigrationQueries))
		require.NoError(t, err)

		// get all providers after migration
		findQueryRes, err = inv_testing.TestClients[inv_testing.APIClient].Find(ctx,
			&inv_v1.ResourceFilter{Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Provider{}}})
		require.NoError(t, err, "find request has rejected")
		// check if an additional license provider is created based on the "fm_onboarding" provider
		require.Len(t, findQueryRes.GetResources(), len(providers)+1,
			"find request has returned unexpected number of resources")

		// get lenovo provider
		getresp, errInv := inv_testing.TestClients[inv_testing.APIClient].Get(ctx,
			providers["lenovo_provider"].baremetalProvider.ResourceId)
		require.NoError(t, errInv, "GetProvider() failed")
		// confirm lenovo provider is not modified during migration
		if eq, diff := inv_testing.ProtoEqualOrDiff(providers["lenovo_provider"].baremetalProvider,
			getresp.GetResource().GetProvider()); !eq {
			t.Errorf("GetProvider() data not equal: %v", diff)
		}

		// get fm_onboarding provider
		getresp, errInv = inv_testing.TestClients[inv_testing.APIClient].Get(ctx,
			providers["fm_onboarding"].baremetalProvider.ResourceId)
		require.NoError(t, errInv, "GetProvider() failed")
		// update expected config after migration
		providers["fm_onboarding"].baremetalProvider.Config = providers["fm_onboarding"].expectedBareMetalConfig
		// confirm fm_onboarding provider is modified during migration
		if eq, diff := inv_testing.ProtoEqualOrDiff(providers["fm_onboarding"].baremetalProvider,
			getresp.GetResource().GetProvider()); !eq {
			t.Errorf("GetProvider() data not equal: %v", diff)
		}

		// clean up all providers
		findRes, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx,
			&inv_v1.ResourceFilter{Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Provider{}}})
		require.NoError(t, err, "find request has rejected")
		for _, resID := range findRes.GetResources() {
			_, err = inv_testing.TestClients[inv_testing.RMClient].Delete(ctx, resID.GetResourceId())
			if err != nil {
				t.Errorf("DeleteProvider() failed %s", err)
			}
		}
	})
}
