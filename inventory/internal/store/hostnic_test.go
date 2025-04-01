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

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostnicresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/regionresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/store"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

func Test_StrongRelations_On_Delete_Hostnic(t *testing.T) {
	host := inv_testing.CreateHost(t, nil, nil)
	hostNic := inv_testing.CreateHostNic(t, host)
	inv_testing.CreateIPAddress(t, hostNic, true)

	err := inv_testing.DeleteResourceAndReturnError(t, hostNic.ResourceId)
	assertStrongRelationError(t, err, "constraint failed: ERROR: update or delete on table")
}

func Test_Create_Get_Delete_Hostnic(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, region, nil)
	provider := inv_testing.CreateProvider(t, "Test Provider1")
	host := inv_testing.CreateHost(t, site, provider)

	testcases := map[string]struct {
		in    *computev1.HostnicResource
		valid bool
	}{
		"CreateGoodHostnic": {
			in: &computev1.HostnicResource{
				Host:          host,
				DeviceName:    "eno1",
				PciIdentifier: "0000:03:00.0",
				SriovEnabled:  true,
				SriovVfsNum:   7,
				SriovVfsTotal: 128,
				LinkState:     computev1.NetworkInterfaceLinkState_NETWORK_INTERFACE_LINK_STATE_UP,
				Mtu:           1500,
				BmcInterface:  false,
			},
			valid: true,
		},
		"CreateBadHostnicWithResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &computev1.HostnicResource{
				ResourceId:    "hostnic-12345678",
				Host:          host,
				DeviceName:    "eno1",
				PciIdentifier: "0000:03:00.0",
				SriovEnabled:  true,
				SriovVfsNum:   7,
				SriovVfsTotal: 128,
			},
			valid: false,
		},
		"CreateBadHostnicWithInvalidResourceIdSet": {
			// This tests case verifies that create requests with a invalid resource ID
			// already set are rejected.
			in: &computev1.HostnicResource{
				ResourceId:    "host-nic-12345678",
				Host:          host,
				DeviceName:    "eno1",
				PciIdentifier: "0000:03:00.0",
				SriovEnabled:  true,
				SriovVfsNum:   7,
				SriovVfsTotal: 128,
			},
			valid: false,
		},
		"CreateBadHostnic_NoHostAssociated": {
			in:    &computev1.HostnicResource{},
			valid: false,
		},
		"CreateBadHostnicNonExistingHost": {
			// This tests case verifies that NICs must point to a valid
			// existing host.
			in: &computev1.HostnicResource{
				Host: &computev1.HostResource{
					ResourceId: "host-12345678",
				},
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Hostnic{Hostnic: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// create hostnic
			chostnicResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(
				ctx,
				createresreq,
			)
			hostnicResID := chostnicResp.GetHostnic().GetResourceId()

			if err != nil {
				if tc.valid {
					t.Errorf("CreateHostnic() failed: %s", err)
				}
			} else {
				tc.in.ResourceId = hostnicResID // Update with created resource ID.
				tc.in.CreatedAt = chostnicResp.GetHostnic().GetCreatedAt()
				tc.in.UpdatedAt = chostnicResp.GetHostnic().GetUpdatedAt()
				assertSameResource(t, createresreq, chostnicResp, nil)
				if !tc.valid {
					t.Errorf("CreateHostnic() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get non-existent first
				_, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, "hostnic-12345678")
				require.Error(t, err)

				// get hostnic
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, hostnicResID)
				require.NoError(t, err, "GetHostnic() failed")

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetHostnic()); !eq {
					t.Errorf("GetHostnic() data not equal: %v", diff)
				}

				// delete non-existent first
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, "hostnic-12345678")
				require.Error(t, err)

				// delete hostnic from API
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, hostnicResID)
				if err != nil {
					t.Errorf("DeleteHostnic() failed: %s", err)
				}

				// get after complete Delete of hostnic, should fail as Hostnic is 2-phase deleted
				_, err = inv_testing.TestClients[inv_testing.RMClient].Get(ctx, hostnicResID)
				require.Error(t, err, "Failure - Hostnic was not deleted, but should be deleted")
			}
		})
	}
}

func Test_UpdateHostnic(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site := inv_testing.CreateSite(t, region, nil)
	provider := inv_testing.CreateProvider(t, "Test Provider1")
	host := inv_testing.CreateHost(t, site, provider)

	// create Hostnic to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hostnic{
			Hostnic: &computev1.HostnicResource{
				Host: host,
			},
		},
	}

	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cvmResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	hostnicResID := inv_testing.GetResourceIDOrFail(t, cvmResp)
	t.Cleanup(func() { inv_testing.DeleteResource(t, hostnicResID) })

	testcases := map[string]struct {
		in           *computev1.HostnicResource
		resourceID   string
		fieldMask    *fieldmaskpb.FieldMask
		valid        bool
		expErrorCode codes.Code
	}{
		"Update1": {
			in: &computev1.HostnicResource{
				Kind: "some kind",
			},
			resourceID: hostnicResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostnicresource.FieldKind},
			},
			valid: true,
		},
		"Update2": {
			in: &computev1.HostnicResource{
				DeviceName: "nic0",
			},
			resourceID: hostnicResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostnicresource.FieldDeviceName},
			},
			valid: true,
		},
		"Update3": {
			in: &computev1.HostnicResource{
				Host: host,
			},
			resourceID: hostnicResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostnicresource.FieldDeviceName},
			},
			valid: true,
		},
		"Update4": {
			in: &computev1.HostnicResource{
				LinkState: computev1.NetworkInterfaceLinkState_NETWORK_INTERFACE_LINK_STATE_DOWN,
			},
			resourceID: hostnicResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostnicresource.FieldLinkState},
			},
			valid: true,
		},
		"Update5": {
			in: &computev1.HostnicResource{
				Mtu: 1500,
			},
			resourceID: hostnicResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostnicresource.FieldMtu},
			},
			valid: true,
		},
		"Update6": {
			in: &computev1.HostnicResource{
				BmcInterface: false,
			},
			resourceID: hostnicResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostnicresource.FieldBmcInterface},
			},
			valid: true,
		},
		"UpdateNoFieldMask": {
			in: &computev1.HostnicResource{
				Host: host,
			},
			resourceID:   hostnicResID,
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateInvalidFieldMask": {
			in: &computev1.HostnicResource{
				Kind: "some kind",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"jklljkjlklkj"},
			},
			resourceID:   hostnicResID,
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateResourceIDNotFound": {
			in: &computev1.HostnicResource{
				Kind: "some kind",
			},
			resourceID: "hostnic-12345678",
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostnicresource.FieldKind},
			},
			valid:        false,
			expErrorCode: codes.NotFound,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Hostnic{Hostnic: tc.in},
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

			// Validate returned resource
			assertSameResource(t, updateresreq, upRes, tc.fieldMask)

			// validate update via a get
			getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, tc.resourceID)
			require.NoError(t, err, "GetResource() failed")

			assertSameResource(t, updateresreq, getresp.GetResource(), tc.fieldMask)
		})
	}
}

func Test_FilterHostnics(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site1 := inv_testing.CreateSite(t, region, nil)
	provider1 := inv_testing.CreateProvider(t, "Test Provider1")
	provider2 := inv_testing.CreateProvider(t, "Test Provider2")
	host1 := inv_testing.CreateHost(t, site1, provider1)
	host2 := inv_testing.CreateHost(t, site1, provider2)
	host3 := inv_testing.CreateHost(t, nil, nil)

	// create Hostnics to find
	createresreq1 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hostnic{
			Hostnic: &computev1.HostnicResource{
				Host:         host1,
				LinkState:    computev1.NetworkInterfaceLinkState_NETWORK_INTERFACE_LINK_STATE_UP,
				Mtu:          1500,
				BmcInterface: false,
			},
		},
	}

	createresreq2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hostnic{
			Hostnic: &computev1.HostnicResource{
				Host:         host2,
				BmcInterface: true,
			},
		},
	}

	createresreqEmpty := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hostnic{
			Hostnic: &computev1.HostnicResource{
				Host: host3,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	chostnicResp1, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq1)
	hostnicResID1 := inv_testing.GetResourceIDOrFail(t, chostnicResp1)
	chostnicResp2, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq2)
	hostnicResID2 := inv_testing.GetResourceIDOrFail(t, chostnicResp2)
	chostnicRespEmtpy, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreqEmpty)
	hostnicResIDEmpty := inv_testing.GetResourceIDOrFail(t, chostnicRespEmtpy)
	t.Cleanup(func() { inv_testing.DeleteResource(t, hostnicResID1) })
	t.Cleanup(func() { inv_testing.DeleteResource(t, hostnicResID2) })
	t.Cleanup(func() { inv_testing.DeleteResource(t, hostnicResIDEmpty) })

	expHostNic1 := createresreq1.GetHostnic()
	expHostNic1.ResourceId = hostnicResID1

	expHostNic2 := createresreq2.GetHostnic()
	expHostNic2.ResourceId = hostnicResID2

	expHostNicEmpty := createresreqEmpty.GetHostnic()
	expHostNicEmpty.ResourceId = hostnicResIDEmpty

	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*computev1.HostnicResource
		valid     bool
	}{
		"NoFilter": {
			in:        &inv_v1.ResourceFilter{},
			resources: []*computev1.HostnicResource{expHostNic1, expHostNic2, expHostNicEmpty},
			valid:     true,
		},
		"NoFilterOrderByResourceID": {
			in: &inv_v1.ResourceFilter{
				OrderBy: hostnicresource.FieldResourceID,
			},
			resources: []*computev1.HostnicResource{expHostNic1, expHostNic2, expHostNicEmpty},
			valid:     true,
		},
		"FilterByEmptyResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = ""`, hostnicresource.FieldResourceID),
			},
			resources: []*computev1.HostnicResource{},
			valid:     true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, hostnicresource.FieldResourceID, expHostNic2.ResourceId),
			},
			resources: []*computev1.HostnicResource{expHostNic2},
			valid:     true,
		},
		"FilterByBmcInterfaceTrue": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = true`, hostnicresource.FieldBmcInterface),
			},
			resources: []*computev1.HostnicResource{expHostNic2},
			valid:     true,
		},
		"FilterByBmcInterfaceNotFalse": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s != false`, hostnicresource.FieldBmcInterface),
			},
			resources: []*computev1.HostnicResource{expHostNic2},
			valid:     true,
		},
		"FilterByBmcInterfaceFalse": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = false`, hostnicresource.FieldBmcInterface),
			},
			resources: []*computev1.HostnicResource{expHostNic1, expHostNicEmpty},
			valid:     true,
		},
		"FilterByBmcInterfaceNotTrue": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s != true`, hostnicresource.FieldBmcInterface),
			},
			resources: []*computev1.HostnicResource{expHostNic1, expHostNicEmpty},
			valid:     true,
		},
		"FilterByNotHasHost": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, hostnicresource.EdgeHost),
			},
			valid: true, // HostNic must have a Host
		},
		"FilterByHostId": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, hostnicresource.EdgeHost, hostresource.FieldResourceID, host2.GetResourceId()),
			},
			resources: []*computev1.HostnicResource{expHostNic2},
			valid:     true,
		},
		"FilterByHasHostHasSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, hostnicresource.EdgeHost, hostresource.EdgeSite),
			},
			resources: []*computev1.HostnicResource{expHostNic1, expHostNic2},
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
			resources: []*computev1.HostnicResource{expHostNic1, expHostNic2, expHostNicEmpty},
			valid:     true,
		},
		"FilterInvalidEdge": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, "invalid_edge"),
			},
			valid: false,
		},
		"FilterInvalidField": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, "invalid_field", "some-value"),
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Hostnic{}} // Set the resource kind
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterHostnic() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterHostnic() succeeded but should have failed")
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
						"FilterHostnic() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListHostnic() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListHostnic() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*computev1.HostnicResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetHostnic())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					hostnicEdgesOnlyResourceID(expected)
					hostnicEdgesOnlyResourceID(resources[i])

					// Skip check of CreatedAt and UpdatedAt.
					resources[i].CreatedAt = expected.CreatedAt
					resources[i].UpdatedAt = expected.UpdatedAt
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListHostnic() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func hostnicEdgesOnlyResourceID(expected *computev1.HostnicResource) {
	if expected.Host != nil {
		expected.Host = &computev1.HostResource{ResourceId: expected.Host.ResourceId}
	}
}

func Test_NestedFilterHostnic(t *testing.T) {
	region1 := inv_testing.CreateRegion(t, nil)
	site1 := inv_testing.CreateSite(t, region1, nil)
	host1 := inv_testing.CreateHost(t, site1, nil)
	host2 := inv_testing.CreateHost(t, nil, nil)

	hostNic1 := inv_testing.CreateHostNic(t, host1)
	hostNic1.Host = host1
	hostNic2 := inv_testing.CreateHostNic(t, host2)
	hostNic2.Host = host2

	testcases := map[string]struct {
		in                *inv_v1.ResourceFilter
		resources         []*computev1.HostnicResource
		valid             bool
		expectedCodeError codes.Code
	}{
		"FilterBySiteID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s = %q`, hostnicresource.EdgeHost, hostresource.EdgeSite,
					siteresource.FieldResourceID, site1.GetResourceId()),
			},
			resources: []*computev1.HostnicResource{hostNic1},
			valid:     true,
		},
		"FilterBySiteEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s.%s)`, hostnicresource.EdgeHost, hostresource.EdgeSite),
			},
			resources: []*computev1.HostnicResource{hostNic2},
			valid:     true,
		},
		"FilterByHasSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, hostnicresource.EdgeHost, hostresource.EdgeSite),
			},
			resources: []*computev1.HostnicResource{hostNic1},
			valid:     true,
		},
		"FilterByRegionID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s = %q`, hostnicresource.EdgeHost, hostresource.EdgeSite,
					siteresource.EdgeRegion, regionresource.FieldResourceID, region1.GetResourceId()),
			},
			resources: []*computev1.HostnicResource{hostNic1},
			valid:     true,
		},
		"FailTooDeep": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s.%s.%s = %q`, hostnicresource.EdgeHost, hostresource.EdgeSite,
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Hostnic{}} // Set the resource kind

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

				resources := make([]*computev1.HostnicResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetHostnic())
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

func Test_StrongRelations_On_Delete_HostNic(t *testing.T) {
	host := inv_testing.CreateHost(t, nil, nil)
	inv_testing.CreateHostNic(t, host)

	err := inv_testing.HardDeleteHostAndReturnError(t, host.ResourceId)
	assertStrongRelationError(t, err, "violates foreign key constraint")
}

func Test_HostnicEnumStateMap(t *testing.T) {
	v, err := store.HostnicEnumStateMap("invalid_input",
		int32(computev1.NetworkInterfaceLinkState_NETWORK_INTERFACE_LINK_STATE_UP))
	assert.Error(t, err)
	assert.Nil(t, v)
}

func TestHostNicMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				host := dao.CreateHost(t, tenantID)
				hostNIC := dao.CreateHostNic(t, tenantID, host)
				res, err := util.WrapResource(hostNIC)
				require.NoError(t, err)
				return hostNIC.GetResourceId(), res
			},
		},
	})
}

func TestDeleteResources_HostNICs(t *testing.T) {
	suite.Run(t, &struct{ hardDeleteAllResourcesSuite }{
		hardDeleteAllResourcesSuite: hardDeleteAllResourcesSuite{
			createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
				tenantID := uuid.NewString()
				host := dao.CreateHost(t, tenantID)
				return tenantID, len(
					[]any{
						dao.CreateHostNicNoCleanup(t, tenantID, host),
						dao.CreateHostNicNoCleanup(t, tenantID, host),
						dao.CreateHostNicNoCleanup(t, tenantID, host),
					},
				)
			},
			resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_HOSTNIC,
		},
	})
}
