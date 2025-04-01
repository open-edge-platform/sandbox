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

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hoststorageresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/regionresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

func Test_Create_Get_Delete_Hoststorage(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, region, nil)
	provider := inv_testing.CreateProvider(t, "Test Provider1")
	host := inv_testing.CreateHost(t, site, provider)

	testcases := map[string]struct {
		in    *computev1.HoststorageResource
		valid bool
	}{
		"CreateGoodHoststorage": {
			in: &computev1.HoststorageResource{
				Host:          host,
				Wwid:          "wwn-...",
				Serial:        "21FFFFFFFFFF",
				Vendor:        "WD",
				Model:         "WDC_FFFFFFFFFFF",
				CapacityBytes: 500107862016,
				DeviceName:    "sda",
			},
			valid: true,
		},
		"CreateBadHoststorageWithResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &computev1.HoststorageResource{
				ResourceId:    "hoststorage-12345678",
				Host:          host,
				Wwid:          "wwn-...",
				Serial:        "21FFFFFFFFFF",
				Vendor:        "WD",
				Model:         "WDC_FFFFFFFFFFF",
				CapacityBytes: 500107862016,
			},
			valid: false,
		},
		"CreateBadHoststorageWithInvalidResourceIdSet": {
			// This tests case verifies that create requests with a invalid resource ID
			// already set are rejected.
			in: &computev1.HoststorageResource{
				ResourceId:    "host-storage-12345678",
				Host:          host,
				Wwid:          "wwn-...",
				Serial:        "21FFFFFFFFFF",
				Vendor:        "WD",
				Model:         "WDC_FFFFFFFFFFF",
				CapacityBytes: 500107862016,
			},
			valid: false,
		},
		"CreateBadHoststorage_NoHostAssociated": {
			in:    &computev1.HoststorageResource{},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Hoststorage{Hoststorage: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// create hoststorage
			choststorageResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
			hoststorageResID := choststorageResp.GetHoststorage().GetResourceId()

			if err != nil {
				if tc.valid {
					t.Errorf("CreateHoststorage() failed: %s", err)
				}
			} else {
				tc.in.ResourceId = hoststorageResID // Update with created resource ID.
				tc.in.CreatedAt = choststorageResp.GetHoststorage().GetCreatedAt()
				tc.in.UpdatedAt = choststorageResp.GetHoststorage().GetUpdatedAt()
				assertSameResource(t, createresreq, choststorageResp, nil)
				if !tc.valid {
					t.Errorf("CreateHoststorage() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get non-existent first
				_, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, "hoststorage-12345678")
				require.Error(t, err)

				// get hoststorage
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, hoststorageResID)
				require.NoError(t, err, "GetHoststorage() failed")

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetHoststorage()); !eq {
					t.Errorf("GetHoststorage() data not equal: %v", diff)
				}

				// delete non-existent first
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, "hoststorage-12345678")
				require.Error(t, err)

				// delete hoststorage from API
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, hoststorageResID)
				if err != nil {
					t.Errorf("DeleteHoststorage() failed: %s", err)
				}

				// get after complete Delete of hoststorage, should fail as Hoststorage is 2-phase deleted
				_, err = inv_testing.TestClients[inv_testing.RMClient].Get(ctx, hoststorageResID)
				require.Error(t, err, "Failure - Hoststorage was not deleted, but should be deleted")
			}
		})
	}
}

func Test_UpdateHoststorage(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, region, nil)
	provider := inv_testing.CreateProvider(t, "Test Provider1")
	host := inv_testing.CreateHost(t, site, provider)

	// create Hoststorage to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hoststorage{
			Hoststorage: &computev1.HoststorageResource{
				Host:          host,
				Wwid:          "wwn-...",
				Serial:        "21FFFFFFFFFF",
				Vendor:        "WD",
				Model:         "WDC_FFFFFFFFFFF",
				CapacityBytes: 500107862016,
			},
		},
	}

	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cvmResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	require.NoError(t, err)
	hoststorageResID := inv_testing.GetResourceIDOrFail(t, cvmResp)
	t.Cleanup(func() { inv_testing.DeleteResource(t, hoststorageResID) })

	testcases := map[string]struct {
		in           *computev1.HoststorageResource
		resourceID   string
		fieldMask    *fieldmaskpb.FieldMask
		valid        bool
		expErrorCode codes.Code
	}{
		"UpdateHoststorage1": {
			in: &computev1.HoststorageResource{
				DeviceName: "storage0",
			},
			resourceID: hoststorageResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hoststorageresource.FieldDeviceName},
			},
			valid: true,
		},
		"UpdateHoststorage2": {
			in: &computev1.HoststorageResource{
				Kind: "some kind",
			},
			resourceID: hoststorageResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{
					hoststorageresource.FieldKind,
					hoststorageresource.FieldDeviceName,
				},
			},
			valid: true,
		},
		"UpdateHoststorage3": {
			in: &computev1.HoststorageResource{
				Host: host,
			},
			resourceID: hoststorageResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hoststorageresource.FieldDeviceName},
			},
			valid: true,
		},
		"UpdateNoFieldMask": {
			in: &computev1.HoststorageResource{
				Host: host,
				Kind: "some kind",
			},
			resourceID:   hoststorageResID,
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateInvalidFieldMask": {
			in: &computev1.HoststorageResource{
				Kind: "some kind",
			},
			resourceID: hoststorageResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"INVALID_FIELD"},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateFieldMaskNonClearableField": {
			in: &computev1.HoststorageResource{
				ResourceId: "proj-fb123457",
			},
			resourceID: hoststorageResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"resource"},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateResourceIDNotFound": {
			in: &computev1.HoststorageResource{
				DeviceName: "storage0",
			},
			resourceID: "hoststorage-12345678",
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hoststorageresource.FieldDeviceName},
			},
			valid:        false,
			expErrorCode: codes.NotFound,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Hoststorage{Hoststorage: tc.in},
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
				assert.Equal(t, tc.expErrorCode, status.Code(err))
				assert.Nil(t, upRes)
				return
			}
			require.NoErrorf(t, err, "UpdateResource() failed: %s", err)

			// validate update via a get
			getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, tc.resourceID)
			require.NoError(t, err, "GetResource() failed")

			assertSameResource(t, updateresreq, getresp.GetResource(), tc.fieldMask)
		})
	}
}

func Test_FilterHoststorages(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site1 := inv_testing.CreateSite(t, region, nil)
	site2 := inv_testing.CreateSite(t, region, nil)
	provider1 := inv_testing.CreateProvider(t, "Test Provider1")
	provider2 := inv_testing.CreateProvider(t, "Test Provider2")
	host1 := inv_testing.CreateHost(t, site1, provider1)
	host2 := inv_testing.CreateHost(t, site2, provider2)
	host3 := inv_testing.CreateHost(t, nil, nil)

	// create Hoststorages to find
	createresreq1 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hoststorage{
			Hoststorage: &computev1.HoststorageResource{
				Host:       host1,
				DeviceName: "sda",
			},
		},
	}

	createresreq2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hoststorage{
			Hoststorage: &computev1.HoststorageResource{
				Host:       host2,
				DeviceName: "nvme0",
			},
		},
	}

	createresreqEmpty := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hoststorage{
			Hoststorage: &computev1.HoststorageResource{
				Host: host3,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	choststorageResp1, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq1)
	hoststorageResID1 := inv_testing.GetResourceIDOrFail(t, choststorageResp1)
	choststorageResp2, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq2)
	hoststorageResID2 := inv_testing.GetResourceIDOrFail(t, choststorageResp2)
	choststorageEmpty, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreqEmpty)
	hoststorageResIDEmpty := inv_testing.GetResourceIDOrFail(t, choststorageEmpty)
	t.Cleanup(func() { inv_testing.DeleteResource(t, hoststorageResID1) })
	t.Cleanup(func() { inv_testing.DeleteResource(t, hoststorageResID2) })
	t.Cleanup(func() { inv_testing.DeleteResource(t, hoststorageResIDEmpty) })

	expHoststorage1 := createresreq1.GetHoststorage()
	expHoststorage1.ResourceId = hoststorageResID1

	expHoststorage2 := createresreq2.GetHoststorage()
	expHoststorage2.ResourceId = hoststorageResID2

	expHoststorageEmpty := createresreqEmpty.GetHoststorage()
	expHoststorageEmpty.ResourceId = hoststorageResIDEmpty

	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*computev1.HoststorageResource
		valid     bool
	}{
		"NoFilter": {
			in:        &inv_v1.ResourceFilter{},
			resources: []*computev1.HoststorageResource{expHoststorage1, expHoststorage2, expHoststorageEmpty},
			valid:     true,
		},
		"NoFilterOrderByResourceID": {
			in: &inv_v1.ResourceFilter{
				OrderBy: hoststorageresource.FieldResourceID,
			},
			resources: []*computev1.HoststorageResource{expHoststorage1, expHoststorage2, expHoststorageEmpty},
			valid:     true,
		},
		"FilterByEmptyResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Resource: &inv_v1.Resource{Resource: &inv_v1.Resource_Hostusb{}},
				Filter:   fmt.Sprintf(`%s = ""`, hoststorageresource.FieldResourceID),
			},
			resources: []*computev1.HoststorageResource{},
			valid:     true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, hoststorageresource.FieldResourceID, expHoststorage2.ResourceId),
			},
			resources: []*computev1.HoststorageResource{expHoststorage2},
			valid:     true,
		},
		"FilterHost": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, hoststorageresource.EdgeHost,
					hostresource.FieldResourceID, host2.GetResourceId()),
			},
			resources: []*computev1.HoststorageResource{expHoststorage2},
			valid:     true,
		},
		"FilterHostEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, hoststorageresource.EdgeHost),
			},
			valid: true, // HostStorage must have a Host
		},
		"FilterByHasHostHasSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, hoststorageresource.EdgeHost, hostresource.EdgeSite),
			},
			resources: []*computev1.HoststorageResource{expHoststorage1, expHoststorage2},
			valid:     true,
		},
		"FilterDeviceName": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, hoststorageresource.FieldDeviceName, expHoststorage2.GetDeviceName()),
			},
			resources: []*computev1.HoststorageResource{expHoststorage2},
			valid:     true,
		},
		"FilterDeviceNameEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = ""`, hoststorageresource.FieldDeviceName),
			},
			resources: []*computev1.HoststorageResource{expHoststorageEmpty},
			valid:     true,
		},
		"FilterInvalidField": {
			in: &inv_v1.ResourceFilter{
				Filter: `invalid_field = "foo"`,
			},
			valid: false,
		},
		"FilterInvalidEdge": {
			in: &inv_v1.ResourceFilter{
				Filter: `has(invalid_edge)`,
			},
			valid: false,
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
			resources: []*computev1.HoststorageResource{expHoststorage1, expHoststorage2, expHoststorageEmpty},
			valid:     true,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Hoststorage{}} // Set the resource kind
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterHoststorage() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterHoststorage() succeeded but should have failed")
				}
			}

			// only get/delete if valid test with non-zero returned response and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				if len(findres.Resources) != len(tc.resources) {
					t.Errorf("Expected to obtain %d Resource IDs, but obtained back %d Resource IDs",
						len(tc.resources), len(findres.Resources))
				}

				resIDs := inv_testing.GetSortedResourceIDSlice(tc.resources)
				inv_testing.SortHasResourceIDAndTenantID(findres.Resources)

				if !reflect.DeepEqual(resIDs, findres.Resources) {
					t.Errorf(
						"FilterHoststorage() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListHoststorage() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListHoststorage() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*computev1.HoststorageResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetHoststorage())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					hoststorageEdgesOnlyResourceID(expected)
					hoststorageEdgesOnlyResourceID(resources[i])

					// Skip check of CreatedAt and UpdatedAt.
					resources[i].CreatedAt = expected.CreatedAt
					resources[i].UpdatedAt = expected.UpdatedAt
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListHoststorage() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func hoststorageEdgesOnlyResourceID(expected *computev1.HoststorageResource) {
	if expected.Host != nil {
		expected.Host = &computev1.HostResource{ResourceId: expected.Host.ResourceId}
	}
}

func Test_NestedFilterHoststorage(t *testing.T) {
	region1 := inv_testing.CreateRegion(t, nil)
	site1 := inv_testing.CreateSite(t, region1, nil)
	host1 := inv_testing.CreateHost(t, site1, nil)
	host2 := inv_testing.CreateHost(t, nil, nil)

	hostStorage1 := inv_testing.CreateHostStorage(t, host1)
	hostStorage1.Host = host1
	hostStorage2 := inv_testing.CreateHostStorage(t, host2)
	hostStorage2.Host = host2

	testcases := map[string]struct {
		in                *inv_v1.ResourceFilter
		resources         []*computev1.HoststorageResource
		valid             bool
		expectedCodeError codes.Code
	}{
		"FilterBySiteID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s = %q`, hoststorageresource.EdgeHost, hostresource.EdgeSite,
					siteresource.FieldResourceID, site1.GetResourceId()),
			},
			resources: []*computev1.HoststorageResource{hostStorage1},
			valid:     true,
		},
		"FilterBySiteEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s.%s)`, hoststorageresource.EdgeHost, hostresource.EdgeSite),
			},
			resources: []*computev1.HoststorageResource{hostStorage2},
			valid:     true,
		},
		"FilterByHasSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, hoststorageresource.EdgeHost, hostresource.EdgeSite),
			},
			resources: []*computev1.HoststorageResource{hostStorage1},
			valid:     true,
		},
		"FilterByRegionID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s = %q`, hoststorageresource.EdgeHost, hostresource.EdgeSite,
					siteresource.EdgeRegion, regionresource.FieldResourceID, region1.GetResourceId()),
			},
			resources: []*computev1.HoststorageResource{hostStorage1},
			valid:     true,
		},
		"FailTooDeep": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s.%s.%s = %q`, hoststorageresource.EdgeHost, hostresource.EdgeSite,
					siteresource.EdgeRegion, regionresource.EdgeParentRegion, regionresource.EdgeParentRegion,
					regionresource.FieldResourceID, region1.GetResourceId()),
			},
			valid:             false,
			expectedCodeError: codes.InvalidArgument,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Hoststorage{}} // Set the resource kind

			// Test FIND
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)
			if !tc.valid {
				require.Error(t, err)
				assert.Equal(t, tc.expectedCodeError, status.Code(err))
			} else {
				require.NoError(t, err)

				resIDs := inv_testing.GetSortedResourceIDSlice(tc.resources)
				inv_testing.SortHasResourceIDAndTenantID(findres.Resources)

				if !reflect.DeepEqual(resIDs, findres.Resources) {
					t.Errorf(
						"FilterInstances() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			// Test LIST
			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)
			if !tc.valid {
				require.Error(t, err)
				assert.Equal(t, tc.expectedCodeError, status.Code(err))
			} else {
				require.NoError(t, err)

				resources := make([]*computev1.HoststorageResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetHoststorage())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListInstances() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func Test_StrongRelations_On_Delete_HostStorage(t *testing.T) {
	host := inv_testing.CreateHost(t, nil, nil)
	inv_testing.CreateHostStorage(t, host)

	err := inv_testing.HardDeleteHostAndReturnError(t, host.ResourceId)
	assertStrongRelationError(t, err, "violates foreign key constraint")
}

func TestHostStorageMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				parent := dao.CreateHost(t, tenantID)
				child := dao.CreateHostStorage(t, tenantID, parent)
				res, err := util.WrapResource(child)
				require.NoError(t, err)
				return child.GetResourceId(), res
			},
		},
	})
}

func TestDeleteResources_HostStorages(t *testing.T) {
	suite.Run(t, &struct{ hardDeleteAllResourcesSuite }{
		hardDeleteAllResourcesSuite: hardDeleteAllResourcesSuite{
			createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
				tenantID := uuid.NewString()
				host := dao.CreateHost(t, tenantID)
				return tenantID, len(
					[]any{
						dao.CreateHostStorageNoCleanup(t, tenantID, host),
						dao.CreateHostStorageNoCleanup(t, tenantID, host),
						dao.CreateHostStorageNoCleanup(t, tenantID, host),
					},
				)
			},
			resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_HOSTSTORAGE,
		},
	})
}
