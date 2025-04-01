// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/localaccountresource"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	localaccount_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/localaccount/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
)

//nolint:funlen // length due to test cases
func Test_Create_Get_Delete_LocalAccount(t *testing.T) {
	testcases := map[string]struct {
		in    *localaccount_v1.LocalAccountResource
		valid bool
	}{
		"CreateGoodLocalAccount": {
			in: &localaccount_v1.LocalAccountResource{
				Username: "test-user",
				SshKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/" +
					" test-user@example.com",
			},
			valid: true,
		},
		"CreateGoodLocalAccountEcdsa-sha2-nistp521": {
			in: &localaccount_v1.LocalAccountResource{
				Username: "test-user",
				SshKey: "ecdsa-sha2-nistp521 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/ " +
					"test-user@example.com",
			},
			valid: true,
		},
		"CreateBadLocalAccountWithResourceIdSet": {
			in: &localaccount_v1.LocalAccountResource{
				ResourceId: "localaccount-12345678",
				Username:   "test-user",
				SshKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/" +
					"test-user@example.com",
			},
			valid: false,
		},
		"CreateBadLocalAccountWithInvalidResourceIdSet": {
			in: &localaccount_v1.LocalAccountResource{
				ResourceId: "localaccount-test-12345678",
				Username:   "test-user",
				SshKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/ " +
					"test-user@example.com",
			},
			valid: false,
		},
		"CreateBadLocalAccountWithLongName": {
			in: &localaccount_v1.LocalAccountResource{
				Username: inv_testing.RandomString(2001),
				SshKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/ " +
					"test-user@example.com",
			},
			valid: false,
		},
		"CreateBadLocalAccountWithInvalidSshKey": {
			in: &localaccount_v1.LocalAccountResource{
				Username: "test-user",
				SshKey:   inv_testing.RandomString(2001),
			},
			valid: false,
		},
		"CreateBadLocalAccountWithSshKeyUnsupportedAlgo": {
			in: &localaccount_v1.LocalAccountResource{
				Username: "test-user",
				SshKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDc0c/VcGcatxNSCefhbn2AhKhs1oWxkQZeZrDx3/V5" +
					"fLDRksR8bR2r61iG+VAboHbMuMLlieH5DCn6zHYC67x5xprNRXelnQNzXvFN5Drw2pGN1TEl+IkUbh/Os/UrDKjZt" +
					"4jT0P+vbtHigCqwF2nRSwNNSlj70P9GRbMF5XY9MW+U+vndqMHkoECUgvyRcrFlePchyN2jo/Rlv6RFNwzLCrUwoFexm+" +
					"KYW/79+iebolGVUdgQySJOIE1iO/aGwnkw/GYleZoY/X8cCujxhjhaAvBw35SgQCAUQJVHloxTIB14jHBMeTgaU1fGTh+187+" +
					"dtCky8yyPJoWrLyEoEYiyLlM4U9fU3KXQoR20qr01b2GTzerj4xKcM7LMVfaevX5bjgbfLj/dukeg8JCElJIqrtHk6OpI+UFAQ3" +
					"1HrKovFl20/wJHAs7wbHnDMhRLE2IMGx5n/5P5uX357Bc0hmdb2IepCF/iPnBIlcPDe9tGDHhcbh11j4Rfu8vUCnJBtoc" +
					"= test-user@example.com",
			},
			valid: false,
		},
		"CreateBadLocalAccountWithEmptySshKey": {
			in: &localaccount_v1.LocalAccountResource{
				Username: "test-user",
				SshKey:   "",
			},
			valid: false,
		},
		"CreateBadLocalAccountWithMissingSshKey": {
			in: &localaccount_v1.LocalAccountResource{
				Username: "test-user",
			},
			valid: false,
		},
		"CreateBadLocalAccountWithInvalidUsername": {
			in: &localaccount_v1.LocalAccountResource{
				Username: " ",
				SshKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDc0c/VcGcatxNSCefhbn2AhKhs1oWxkQZeZrDx3/V5" +
					"fLDRksR8bR2r61iG+VAboHbMuMLlieH5DCn6zHYC67x5xprNRXelnQNzXvFN5Drw2pGN1TEl+IkUbh/Os/UrDKjZt" +
					"4jT0P+vbtHigCqwF2nRSwNNSlj70P9GRbMF5XY9MW+U+vndqMHkoECUgvyRcrFlePchyN2jo/Rlv6RFNwzLCrUwoFexm+" +
					"KYW/79+iebolGVUdgQySJOIE1iO/aGwnkw/GYleZoY/X8cCujxhjhaAvBw35SgQCAUQJVHloxTIB14jHBMeTgaU1fGTh+187+" +
					"dtCky8yyPJoWrLyEoEYiyLlM4U9fU3KXQoR20qr01b2GTzerj4xKcM7LMVfaevX5bjgbfLj/dukeg8JCElJIqrtHk6OpI+UFAQ3" +
					"1HrKovFl20/wJHAs7wbHnDMhRLE2IMGx5n/5P5uX357Bc0hmdb2IepCF/iPnBIlcPDe9tGDHhcbh11j4Rfu8vUCnJBtoc" +
					"= test-user@example.com",
			},
			valid: false,
		},
		"CreateBadLocalAccountWithEmptyUsername": {
			in: &localaccount_v1.LocalAccountResource{
				Username: "",
				SshKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDc0c/VcGcatxNSCefhbn2AhKhs1oWxkQZeZrDx3/V5" +
					"fLDRksR8bR2r61iG+VAboHbMuMLlieH5DCn6zHYC67x5xprNRXelnQNzXvFN5Drw2pGN1TEl+IkUbh/Os/UrDKjZt" +
					"4jT0P+vbtHigCqwF2nRSwNNSlj70P9GRbMF5XY9MW+U+vndqMHkoECUgvyRcrFlePchyN2jo/Rlv6RFNwzLCrUwoFexm+" +
					"KYW/79+iebolGVUdgQySJOIE1iO/aGwnkw/GYleZoY/X8cCujxhjhaAvBw35SgQCAUQJVHloxTIB14jHBMeTgaU1fGTh+187+" +
					"dtCky8yyPJoWrLyEoEYiyLlM4U9fU3KXQoR20qr01b2GTzerj4xKcM7LMVfaevX5bjgbfLj/dukeg8JCElJIqrtHk6OpI+UFAQ3" +
					"1HrKovFl20/wJHAs7wbHnDMhRLE2IMGx5n/5P5uX357Bc0hmdb2IepCF/iPnBIlcPDe9tGDHhcbh11j4Rfu8vUCnJBtoc" +
					"= test-user@example.com",
			},
			valid: false,
		},
		"CreateBadLocalAccountWithMissingUsername": {
			in: &localaccount_v1.LocalAccountResource{
				SshKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/ " +
					"test-user@example.com",
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_LocalAccount{LocalAccount: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// create
			cprovResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
			localAccountResID := cprovResp.GetLocalAccount().GetResourceId()

			if err != nil {
				if tc.valid {
					t.Errorf("CreateLocalAccount() failed: %s", err)
				}
			} else {
				tc.in.ResourceId = localAccountResID // Update with created resource ID.
				tc.in.CreatedAt = cprovResp.GetLocalAccount().GetCreatedAt()
				tc.in.UpdatedAt = cprovResp.GetLocalAccount().GetUpdatedAt()
				assertSameResource(t, createresreq, cprovResp, nil)
				if !tc.valid {
					t.Errorf("CreateLocalAccount() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get non-existent first
				_, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, "localaccount-12345678")
				require.Error(t, err)

				// getl
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, localAccountResID)
				require.NoError(t, err, "CreateLocalAccount() failed")

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetLocalAccount()); !eq {
					t.Errorf("GetLocalAccount() data not equal: %v", diff)
				}

				// delete non-existent first
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, "localAccount-12345678")
				require.Error(t, err)

				// delete
				_, err = inv_testing.TestClients[inv_testing.RMClient].Delete(
					ctx,
					localAccountResID,
				)
				if err != nil {
					t.Errorf("DeleteLocalAccount() failed %s", err)
				}
			}
		})
	}
}

//nolint:cyclop,funlen // length due to test cases
func Test_FilterLocalAccount(t *testing.T) {
	LocalAccount1 := inv_testing.CreateLocalAccount(t,
		"test-user",
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/ test-user@example.com")

	createresreq2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_LocalAccount{
			LocalAccount: &localaccount_v1.LocalAccountResource{
				Username: "test-user1",
				SshKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/ " +
					"test-user1@example.com",
			},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	LocalAccount2, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq2)
	require.NoError(t, err)
	// Get the resource ID for LocalAccount2
	localAccount2ResID := inv_testing.GetResourceIDOrFail(t, LocalAccount2)
	// Clean up the resource after the test
	t.Cleanup(func() { inv_testing.DeleteResource(t, localAccount2ResID) })

	expLocalAccount1 := LocalAccount1
	expLocalAccount2 := LocalAccount2.GetLocalAccount()
	expLocalAccount2.ResourceId = localAccount2ResID
	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*localaccount_v1.LocalAccountResource
		valid     bool
	}{
		"NoFilter": {
			in:        &inv_v1.ResourceFilter{},
			resources: []*localaccount_v1.LocalAccountResource{expLocalAccount1, expLocalAccount2},
			valid:     true,
		},
		"NoFilterOrderByResourceID": {
			in: &inv_v1.ResourceFilter{
				OrderBy: localaccountresource.FieldResourceID,
			},
			resources: []*localaccount_v1.LocalAccountResource{expLocalAccount1, expLocalAccount2},
			valid:     true,
		},
		"FilterByUsernameEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, localaccountresource.FieldUsername, expLocalAccount2.Username),
			},
			resources: []*localaccount_v1.LocalAccountResource{expLocalAccount2},
			valid:     true,
		},
		"FilterBySshkeyEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, localaccountresource.FieldSSHKey, expLocalAccount1.SshKey),
			},
			resources: []*localaccount_v1.LocalAccountResource{expLocalAccount1},
			valid:     true,
		},
		"FilterWithOffsetLimit1": {
			in: &inv_v1.ResourceFilter{
				Offset: 5,
				Limit:  0,
			},
			valid: true,
		},
		"FilterWithOffsetLimit2": {
			in: &inv_v1.ResourceFilter{
				Offset: 0,
				Limit:  5,
			},
			resources: []*localaccount_v1.LocalAccountResource{expLocalAccount1, expLocalAccount2},
			valid:     true,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_LocalAccount{}}
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterLocalAccount() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterLocalAccount() succeeded but should have failed")
				}
			}

			// only get/delete if valid test with non-zero returned response and hasn't failed, otherwise may segfault
			if !t.Failed() && tc.valid {
				if len(findres.Resources) != len(tc.resources) {
					t.Errorf("Expected to obtain %d Resource IDs, but obtained back %d Resource IDs",
						len(tc.resources), len(findres.Resources))
				}

				resIDs := inv_testing.GetSortedResourceIDSlice(tc.resources)
				inv_testing.SortHasResourceIDAndTenantID(findres.Resources)

				if !reflect.DeepEqual(resIDs, findres.Resources) {
					t.Errorf(
						"FilterLocalAccount() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListLocalAccount() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListLocalAccount() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*localaccount_v1.LocalAccountResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetLocalAccount())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListLocalAccount() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func TestDeleteResources_LocalAccount(t *testing.T) {
	suite.Run(t, &struct{ hardDeleteAllResourcesSuite }{
		hardDeleteAllResourcesSuite: hardDeleteAllResourcesSuite{
			createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
				tenantID := uuid.NewString()
				return tenantID, len([]any{
					dao.CreateLocalAccountNoCleanup(t, tenantID, "test-user1",
						"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/ "+
							"test-user1@example.com"),
					dao.CreateLocalAccountNoCleanup(t, tenantID, "test-user2",
						"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/ "+
							"test-user2@example.com"),
					dao.CreateLocalAccountNoCleanup(t, tenantID, "test-user3",
						"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/ "+
							"test-user3@example.com"),
				})
			},
			resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_LOCALACCOUNT,
		},
	})
}

func Test_StrongRelations_On_Delete_LocalAccount(t *testing.T) {
	t.Run("LocalAccount_Instance", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		os := inv_testing.CreateOs(t)
		host := inv_testing.CreateHost(t, nil, nil)
		localaccount := inv_testing.CreateLocalAccount(t,
			"test-user",
			"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/ test-user1@example.com",
		)
		_ = inv_testing.CreateInstanceWithLocalAccount(t, host, os, localaccount)

		_, err := inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, localaccount.ResourceId)

		require.Error(t, err, "DeleteInstance() should fail")
	})
}

func Test_Unique_LocalAccount_On_Create(t *testing.T) {
	t.Run("LocalAccount_Instance", func(t *testing.T) {
		createresreq := &inv_v1.Resource{
			Resource: &inv_v1.Resource_LocalAccount{
				LocalAccount: &localaccount_v1.LocalAccountResource{
					Username: "test-user",
					SshKey: "ecdsa-sha2-nistp521 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/ " +
						"test-user@example.com",
				},
			},
		}

		// build a context for gRPC
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// create
		_, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
		require.NoError(t, err, "CreateLocalAccount() should Not fail")
		// create another localaccount with same username
		_, err = inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
		require.Error(t, err, "CreateLocalAccount() should fail")
		// create another localaccount with same username second time
		_, err = inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
		require.Error(t, err, "CreateLocalAccount() should fail")
	})
}

func Test_UpdateLocalAccount(t *testing.T) {
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_LocalAccount{
			LocalAccount: &localaccount_v1.LocalAccountResource{
				Username: "test-user1",
				SshKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/ " +
					"test-user1@example.com",
			},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	LocalAccount, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	require.NoError(t, err)

	// Get the resource ID for LocalAccount2
	localAccountResID := inv_testing.GetResourceIDOrFail(t, LocalAccount)
	// Clean up the resource after the test
	t.Cleanup(func() { inv_testing.DeleteResource(t, localAccountResID) })

	testcases := map[string]struct {
		in         *localaccount_v1.LocalAccountResource
		resourceID string
		fieldMask  *fieldmaskpb.FieldMask
		valid      bool
	}{
		"UpdateLocalAccountUsername": {
			in: &localaccount_v1.LocalAccountResource{
				Username: "test-user2",
				SshKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/" +
					" test-user@example.com",
			},
			resourceID: localAccountResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					localaccountresource.FieldUsername,
				},
			},
			valid: false,
		},
		"UpdateLocalAccountSshKey": {
			in: &localaccount_v1.LocalAccountResource{
				Username: "test-user",
				SshKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/" +
					" test-user2@example.com",
			},
			resourceID: localAccountResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					localaccountresource.FieldSSHKey,
				},
			},
			valid: false,
		},
		"UpdateLocalAccountUsernameAndSshKey": {
			in: &localaccount_v1.LocalAccountResource{
				Username: "test-user2",
				SshKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILtu+7Pdtj6ihyFynecnd+155AdxqvHhMRxvxdcQ8/D/" +
					" test-user2@example.com",
			},
			resourceID: localAccountResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					localaccountresource.FieldUsername,
					localaccountresource.FieldSSHKey,
				},
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_LocalAccount{LocalAccount: tc.in},
			}
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			upRes, err := inv_testing.TestClients[inv_testing.RMClient].Update(
				ctx,
				tc.resourceID,
				tc.fieldMask,
				updateresreq,
			)
			if !tc.valid {
				require.Errorf(t, err, "UpdateResource() succeeded but should have failed")
				assert.Nil(t, upRes)
				return
			}
		})
	}
}
