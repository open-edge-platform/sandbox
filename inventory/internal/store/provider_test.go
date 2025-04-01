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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/providerresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/store"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	provider_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/provider/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

func Test_Create_Get_Delete_Update_Provider(t *testing.T) {
	testcases := map[string]struct {
		in    *provider_v1.ProviderResource
		valid bool
	}{
		"CreateGoodProvider": {
			in: &provider_v1.ProviderResource{
				ProviderKind:   provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL,
				ProviderVendor: provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LXCA,
				Name:           "Test Provider 1",
				ApiEndpoint:    "192.168.201.3/discovery",
				ApiCredentials: []string{"test", "test"},
				Config:         "foobar",
			},
			valid: true,
		},
		"CreateBadProviderWithResourceIdSet": {
			in: &provider_v1.ProviderResource{
				ResourceId:     "provider-12345678",
				ProviderKind:   provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL,
				ProviderVendor: provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LOCA,
				Name:           "Test Provider 1",
				ApiEndpoint:    "192.168.201.3/discovery",
				ApiCredentials: []string{"test", "test"},
			},
			valid: false,
		},
		"CreateBadProviderWithInvalidResourceIdSet": {
			in: &provider_v1.ProviderResource{
				ResourceId:     "provide-test-12345678",
				ProviderKind:   provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL,
				ProviderVendor: provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LOCA,
				Name:           "Test Provider 1",
				ApiEndpoint:    "192.168.201.3/discovery",
				ApiCredentials: []string{"test", "test"},
			},
			valid: false,
		},
		"CreateBadProviderWithLongName": {
			in: &provider_v1.ProviderResource{
				ProviderKind:   provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL,
				ProviderVendor: provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LXCA,
				Name:           "Test Provider 123456789123456789123456789123456789123456789123456789123456789123456789",
				ApiEndpoint:    "192.168.201.3/discovery",
				ApiCredentials: []string{"test", "test"},
			},
			valid: false,
		},
		"CreateBadProviderWithConfig": {
			in: &provider_v1.ProviderResource{
				ProviderKind:   provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL,
				ProviderVendor: provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LXCA,
				Name:           "Test Provider",
				ApiEndpoint:    "192.168.201.3/discovery",
				ApiCredentials: []string{"test", "test"},
				Config:         inv_testing.RandomString(2001),
			},
			valid: false,
		},
		"CreateBadProviderWithWrongCredentials": {
			in: &provider_v1.ProviderResource{
				ProviderKind:   provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL,
				ProviderVendor: provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LXCA,
				Name:           "Test Provider 1",
				ApiEndpoint:    "192.168.201.3/discovery",
				ApiCredentials: []string{"test|", "test"},
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Provider{Provider: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// create
			cprovResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
			providerResID := cprovResp.GetProvider().GetResourceId()

			if err != nil {
				if tc.valid {
					t.Errorf("CreateProvider() failed: %s", err)
				}
			} else {
				tc.in.ResourceId = providerResID // Update with created resource ID.
				tc.in.CreatedAt = cprovResp.GetProvider().GetCreatedAt()
				tc.in.UpdatedAt = cprovResp.GetProvider().GetUpdatedAt()
				assertSameResource(t, createresreq, cprovResp, nil)
				if !tc.valid {
					t.Errorf("CreateProvider() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get non-existent first
				_, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, "provider-12345678")
				require.Error(t, err)

				// get
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, providerResID)
				require.NoError(t, err, "GetProvider() failed")

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetProvider()); !eq {
					t.Errorf("GetHost() data not equal: %v", diff)
				}

				// update
				updateresreq := &inv_v1.Resource{
					Resource: &inv_v1.Resource_Provider{
						Provider: &provider_v1.ProviderResource{
							Name: "Updated Name",
						},
					},
				}

				fieldMask := &fieldmaskpb.FieldMask{Paths: []string{providerresource.FieldName}}
				upRes, err := inv_testing.TestClients[inv_testing.RMClient].Update(
					ctx,
					providerResID,
					fieldMask,
					updateresreq,
				)
				if err != nil {
					t.Errorf("UpdateProvider() failed: %s", err)
				}

				// Validate returned resource
				assertSameResource(t, updateresreq, upRes, fieldMask)

				// delete non-existent first
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, "provider-12345678")
				require.Error(t, err)

				// delete
				_, err = inv_testing.TestClients[inv_testing.RMClient].Delete(
					ctx,
					providerResID,
				)
				if err != nil {
					t.Errorf("DeleteProvider() failed %s", err)
				}
			}
		})
	}
}

func Test_UniqueFields(t *testing.T) {
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Provider{
			Provider: &provider_v1.ProviderResource{
				ProviderKind:   provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL,
				ProviderVendor: provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LOCA,
				Name:           "Test Provider 1",
				ApiEndpoint:    "192.168.201.3/discovery",
				ApiCredentials: []string{"test", "test"},
			},
		},
	}

	// create - unique
	cprovResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	require.NoError(t, err)
	resID1 := inv_testing.GetResourceIDOrFail(t, cprovResp)
	t.Cleanup(func() { inv_testing.DeleteResource(t, resID1) })

	createresreq = &inv_v1.Resource{
		Resource: &inv_v1.Resource_Provider{
			Provider: &provider_v1.ProviderResource{
				ProviderKind:   provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL,
				ProviderVendor: provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LXCA,
				Name:           "Test Provider 1",
				ApiEndpoint:    "192.168.20.3/discovery",
				ApiCredentials: []string{"test", "teast"},
			},
		},
	}

	// create - not unique
	_, err = inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	require.Error(t, err)

	// create resource with same name but for the other tenant

	provider := *createresreq.GetProvider()
	provider.TenantId = tenantIDOne
	anotherCreateResourceRequest := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Provider{
			Provider: &provider,
		},
	}
	anotherProviderCreationResp, err := inv_testing.TestClients[inv_testing.APIClient].
		GetTenantAwareInventoryClient().
		Create(
			ctx,
			anotherCreateResourceRequest.GetProvider().GetTenantId(),
			anotherCreateResourceRequest)
	require.NoError(t, err)

	t.Cleanup(
		func() {
			localCtx, localCancel := context.WithTimeout(context.Background(), time.Second)
			defer localCancel()
			_, deletionErr := inv_testing.TestClients[inv_testing.APIClient].
				GetTenantAwareInventoryClient().
				Delete(
					localCtx,
					anotherCreateResourceRequest.GetProvider().GetTenantId(),
					anotherProviderCreationResp.GetProvider().GetResourceId(),
				)
			if deletionErr != nil {
				require.NoError(t, err)
			}
		})

	createresreq = &inv_v1.Resource{
		Resource: &inv_v1.Resource_Provider{
			Provider: &provider_v1.ProviderResource{
				ProviderKind:   provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL,
				ProviderVendor: provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LXCA,
				Name:           "Test Provider 2",
				ApiEndpoint:    "192.168.20.3/discovery",
				ApiCredentials: []string{"test", "teast"},
			},
		},
	}

	// create - unique
	cprovResp, err = inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	require.NoError(t, err)
	resID2 := inv_testing.GetResourceIDOrFail(t, cprovResp)
	t.Cleanup(func() { inv_testing.DeleteResource(t, resID2) })

	// update
	updateresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Provider{
			Provider: &provider_v1.ProviderResource{
				Name: "Test Provider 1",
			},
		},
	}

	// update - not unique
	_, err = inv_testing.TestClients[inv_testing.RMClient].Update(
		ctx,
		resID2,
		&fieldmaskpb.FieldMask{Paths: []string{providerresource.FieldName}},
		updateresreq,
	)
	require.Error(t, err)
}

func Test_FilterProviders(t *testing.T) {
	provider1 := inv_testing.CreateProvider(t, "Test Provider 1")

	createresreq2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Provider{
			Provider: &provider_v1.ProviderResource{
				ProviderKind:   provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL,
				Name:           "Test Provider 2",
				ApiEndpoint:    "192.168.201.3/discovery",
				ApiCredentials: []string{"test", "test"},
				Config:         "foo",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	provider2, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq2)
	require.NoError(t, err)
	provider2ResID := inv_testing.GetResourceIDOrFail(t, provider2)
	t.Cleanup(func() { inv_testing.DeleteResource(t, provider2ResID) })

	expProvider1 := provider1
	expProvider2 := provider2.GetProvider()
	expProvider2.ResourceId = provider2ResID

	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*provider_v1.ProviderResource
		valid     bool
	}{
		"NoFilter": {
			in:        &inv_v1.ResourceFilter{},
			resources: []*provider_v1.ProviderResource{expProvider1, expProvider2},
			valid:     true,
		},
		"NoFilterOrderByResourceID": {
			in: &inv_v1.ResourceFilter{
				OrderBy: providerresource.FieldResourceID,
			},
			resources: []*provider_v1.ProviderResource{expProvider1, expProvider2},
			valid:     true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, providerresource.FieldResourceID, expProvider1.ResourceId),
			},
			resources: []*provider_v1.ProviderResource{expProvider1},
			valid:     true,
		},
		"FilterByConfigEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, providerresource.FieldConfig, expProvider2.Config),
			},
			resources: []*provider_v1.ProviderResource{expProvider2},
			valid:     true,
		},
		"FilterMetal": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %s`, providerresource.FieldProviderKind,
					provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL),
			},
			resources: []*provider_v1.ProviderResource{expProvider1, expProvider2},
			valid:     true,
		},
		"FilterKindEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %s`, providerresource.FieldProviderKind, "null"),
			},
			resources: []*provider_v1.ProviderResource{},
			valid:     true,
		},
		"FilterKindUnspecified": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %s`, providerresource.FieldProviderKind,
					provider_v1.ProviderKind_PROVIDER_KIND_UNSPECIFIED),
			},
			resources: []*provider_v1.ProviderResource{},
			valid:     true,
		},
		"FilterVendor": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %s`, providerresource.FieldProviderVendor,
					provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LXCA),
			},
			resources: []*provider_v1.ProviderResource{expProvider1},
			valid:     true,
		},
		"FilterVendorEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %s`, providerresource.FieldProviderVendor, "null"),
			},
			resources: []*provider_v1.ProviderResource{expProvider2},
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
			resources: []*provider_v1.ProviderResource{expProvider1, expProvider2},
			valid:     true,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Provider{}}
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterProviders() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterProviders() succeeded but should have failed")
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
						"FilterProviders() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListProviders() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListProviders() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*provider_v1.ProviderResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetProvider())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListProviders() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func Test_UpdateProvider(t *testing.T) {
	// create Provider to update
	cprovResp := inv_testing.CreateProviderWithArgs(
		t,
		"Test Provider 2",
		"192.168.201.3/discovery",
		[]string{"test", "test"},
		provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL,
		provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LOCA,
	)

	testcases := map[string]struct {
		in           *provider_v1.ProviderResource
		resourceID   string
		fieldMask    *fieldmaskpb.FieldMask
		valid        bool
		expErrorCode codes.Code
	}{
		"Update1": {
			in: &provider_v1.ProviderResource{
				Name: "Updated Name",
			},
			resourceID: cprovResp.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{providerresource.FieldName},
			},
			valid: true,
		},
		// You cannot unset the provider kind
		"Update2": {
			in: &provider_v1.ProviderResource{
				Name: "Updated Name 2",
			},
			resourceID: cprovResp.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{providerresource.FieldName, providerresource.FieldProviderKind},
			},
			valid:        false,
			expErrorCode: codes.Internal,
		},
		"Update3": {
			in: &provider_v1.ProviderResource{
				ProviderVendor: provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LXCA,
			},
			resourceID: cprovResp.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{providerresource.FieldProviderVendor},
			},
			valid: true,
		},
		"Update4": {
			in: &provider_v1.ProviderResource{
				ProviderVendor: provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LOCA,
			},
			resourceID: cprovResp.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{providerresource.FieldProviderVendor},
			},
			valid: true,
		},
		"Update5": {
			in: &provider_v1.ProviderResource{
				Config: "bar",
			},
			resourceID: cprovResp.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{providerresource.FieldConfig},
			},
			valid: true,
		},
		"UpdateNoFieldMask": {
			in: &provider_v1.ProviderResource{
				Name: "Updated Name 4",
			},
			resourceID:   cprovResp.ResourceId,
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateInvalidFieldMask1": {
			in: &provider_v1.ProviderResource{
				Name: "Updated Name 5",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"INVALID_FIELD"},
			},
			resourceID:   cprovResp.ResourceId,
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateResourceIDNotFound": {
			in: &provider_v1.ProviderResource{
				Name: "Updated Name",
			},
			resourceID: "provider-12345678",
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{providerresource.FieldName},
			},
			valid:        false,
			expErrorCode: codes.NotFound,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Provider{Provider: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			upRes, err := inv_testing.TestClients[inv_testing.APIClient].Update(
				ctx,
				tc.resourceID,
				tc.fieldMask,
				updateresreq,
			)

			if !tc.valid {
				require.Errorf(t, err, "UpdateResource() succeeded but should have failed")
				assert.Equal(t, tc.expErrorCode, status.Code(err))
				assert.Nil(t, upRes)
				return
			}
			require.NoErrorf(t, err, "UpdateResource() failed: %s", err)

			// Validate returned resource
			assertSameResource(t, updateresreq, upRes, tc.fieldMask)

			// validate update via a get
			getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, tc.resourceID)
			require.NoError(t, err, "GetResource() failed")

			assertSameResource(t, updateresreq, getresp.GetResource(), tc.fieldMask)
		})
	}
}

func Test_StrongRelations_On_Delete_Provider(t *testing.T) {
	t.Run("Provider_Host", func(t *testing.T) {
		provider := inv_testing.CreateProvider(t, "Test Provider 1")
		inv_testing.CreateHost(t, nil, provider)

		err := inv_testing.DeleteResourceAndReturnError(t, provider.ResourceId)
		assertStrongRelationError(t, err, "the provider has a relation with host and cannot be deleted")
	})
	t.Run("Provider_Instance", func(t *testing.T) {
		provider := inv_testing.CreateProvider(t, "Test Provider 1")
		os := inv_testing.CreateOs(t)
		host := inv_testing.CreateHost(t, nil, nil)
		inv_testing.CreateInstanceWithProvider(t, host, os, provider)

		err := inv_testing.DeleteResourceAndReturnError(t, provider.ResourceId)
		assertStrongRelationError(t, err, "the provider has a relation with instance and cannot be deleted")
	})
	t.Run("Provider_Site", func(t *testing.T) {
		provider := inv_testing.CreateProvider(t, "Test Provider 1")
		inv_testing.CreateSiteWithArgs(t, "TEST", 0, 0, "", nil, nil, provider)

		err := inv_testing.DeleteResourceAndReturnError(t, provider.ResourceId)
		assertStrongRelationError(t, err, "the provider has a relation with site and cannot be deleted")
	})
}

func Test_ProviderEnumStateMap(t *testing.T) {
	v, err := store.ProviderEnumStateMap("invalid_input",
		int32(provider_v1.ProviderVendor_PROVIDER_VENDOR_LENOVO_LXCA))
	assert.Error(t, err)
	assert.Nil(t, v)
}

func TestProviderMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				provider := dao.CreateProvider(t, tenantID, uuid.NewString(),
					inv_testing.ProviderKind(provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL))
				res, err := util.WrapResource(provider)
				require.NoError(t, err)
				return provider.GetResourceId(), res
			},
		},
	})
}

func TestDeleteResources_Providers(t *testing.T) {
	suite.Run(t, &struct{ hardDeleteAllResourcesSuite }{
		hardDeleteAllResourcesSuite: hardDeleteAllResourcesSuite{
			createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
				tenantID := uuid.NewString()
				return tenantID, len([]any{
					dao.CreateProviderNoCleanup(t, tenantID, "anyProvider1",
						inv_testing.ProviderKind(provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL)),
					dao.CreateProviderNoCleanup(t, tenantID, "anyProvider2",
						inv_testing.ProviderKind(provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL)),
					dao.CreateProviderNoCleanup(t, tenantID, "anyProvider3",
						inv_testing.ProviderKind(provider_v1.ProviderKind_PROVIDER_KIND_BAREMETAL)),
				})
			},
			resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_PROVIDER,
		},
	})
}
