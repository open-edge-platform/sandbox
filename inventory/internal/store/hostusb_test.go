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
	hostusb "github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostusbresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/regionresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

func Test_Create_Get_Delete_Hostusb(t *testing.T) {
	host := inv_testing.CreateHost(t, nil, nil)

	testcases := map[string]struct {
		in    *computev1.HostusbResource
		valid bool
	}{
		"CreateGoodHostusb": {
			in: &computev1.HostusbResource{
				Host:      host,
				Idvendor:  "1d6b",
				Idproduct: "0003",
				Bus:       1,
				Addr:      10,
			},
			valid: true,
		},
		"CreateBadHostusbWithResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &computev1.HostusbResource{
				ResourceId: "hostusb-12345678",
				Host:       host,
				Idvendor:   "1d6b",
				Idproduct:  "0003",
				Bus:        1,
				Addr:       10,
			},
			valid: false,
		},
		"CreateBadHostusbWithInvalidResourceIdSet": {
			// This tests case verifies that create requests with a invalid resource ID
			// already set are rejected.
			in: &computev1.HostusbResource{
				ResourceId: "host-usb-12345678",
				Host:       host,
				Idvendor:   "1d6b",
				Idproduct:  "0003",
				Bus:        1,
				Addr:       10,
			},
			valid: false,
		},
		"CreateBadHostusb_NoHostAssociated": {
			in:    &computev1.HostusbResource{},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Hostusb{Hostusb: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// create hostusb
			chostusbResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
			hostusbResID := chostusbResp.GetHostusb().GetResourceId()

			if err != nil {
				if tc.valid {
					t.Errorf("CreateHostusb() failed: %s", err)
				}
			} else {
				tc.in.ResourceId = hostusbResID // Update with created resource ID.
				tc.in.CreatedAt = chostusbResp.GetHostusb().GetCreatedAt()
				tc.in.UpdatedAt = chostusbResp.GetHostusb().GetUpdatedAt()
				assertSameResource(t, createresreq, chostusbResp, nil)
				if !tc.valid {
					t.Errorf("CreateHostusb() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get non-existent first
				_, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, "hostusb-12345678")
				require.Error(t, err)

				// get hostusb
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, hostusbResID)
				require.NoError(t, err, "GetHostusb() failed")

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetHostusb()); !eq {
					t.Errorf("GetHostusb() data not equal: %v", diff)
				}

				// delete non-existent first
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, "hostusb-12345678")
				require.Error(t, err)

				// delete hostusb from API
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, hostusbResID)
				if err != nil {
					t.Errorf("DeleteHostusb() failed: %s", err)
				}
				// get after complete Delete of hostusb, should fail as Hostusb is 2-phase deleted
				_, err = inv_testing.TestClients[inv_testing.RMClient].Get(ctx, hostusbResID)
				require.Error(t, err, "Failure - Hostusb was not deleted, but should be deleted")
			}
		})
	}
}

func Test_UpdateHostusb(t *testing.T) {
	host := inv_testing.CreateHost(t, nil, nil)
	host2 := inv_testing.CreateHost(t, nil, nil)

	// create Hostusb to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hostusb{
			Hostusb: &computev1.HostusbResource{
				Host:      host,
				Idvendor:  "1d6b",
				Idproduct: "0003",
				Bus:       2,
				Addr:      10,
			},
		},
	}

	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cvmResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	hostusbResID := inv_testing.GetResourceIDOrFail(t, cvmResp)
	t.Cleanup(func() { inv_testing.DeleteResource(t, hostusbResID) })

	testcases := map[string]struct {
		in           *computev1.HostusbResource
		resourceID   string
		fieldMask    *fieldmaskpb.FieldMask
		valid        bool
		expErrorCode codes.Code
	}{
		"Update1": {
			in: &computev1.HostusbResource{
				Kind: "some kind",
			},
			resourceID: hostusbResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostusb.FieldKind},
			},
			valid: true,
		},
		"Update2": {
			in: &computev1.HostusbResource{
				Kind:       "some kind",
				DeviceName: "usb XYZ",
			},
			resourceID: hostusbResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostusb.FieldDeviceName, hostusb.FieldKind},
			},
			valid: true,
		},
		"Update3": {
			in:         &computev1.HostusbResource{},
			resourceID: hostusbResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostusb.FieldDeviceName},
			},
			valid: true,
		},
		"UpdateHost": {
			in:         &computev1.HostusbResource{Host: host2},
			resourceID: hostusbResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostusb.EdgeHost},
			},
			valid: true,
		},
		"UpdateNoFieldMask": {
			in: &computev1.HostusbResource{
				Addr: 2,
			},
			resourceID:   hostusbResID,
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateInvalidFieldMask": {
			in: &computev1.HostusbResource{
				Kind: "some kind",
			},
			resourceID: hostusbResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"INVALID_FIELD"},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateResourceIDNotFound": {
			in: &computev1.HostusbResource{
				Kind: "some kind",
			},
			resourceID: "hostusb-12345678",
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{hostusb.FieldKind},
			},
			valid:        false,
			expErrorCode: codes.NotFound,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Hostusb{Hostusb: tc.in},
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

func Test_FilterHostusbs(t *testing.T) {
	site := inv_testing.CreateSite(t, nil, nil)
	host1 := inv_testing.CreateHost(t, site, nil)
	host2 := inv_testing.CreateHost(t, nil, nil)

	// create Hostusbs to find
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hostusb{
			Hostusb: &computev1.HostusbResource{
				DeviceName: "Hostusb 4",
				Host:       host1,
			},
		},
	}

	createresreqEmpty := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Hostusb{
			Hostusb: &computev1.HostusbResource{
				Host:       host2,
				DeviceName: "Hostusb 4",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	chostusbResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	require.NoError(t, err)
	hostusbResID := inv_testing.GetResourceIDOrFail(t, chostusbResp)
	t.Cleanup(func() { inv_testing.DeleteResource(t, hostusbResID) })
	chostusbRespEmpty, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreqEmpty)
	require.NoError(t, err)
	hostusbResIDEmpty := inv_testing.GetResourceIDOrFail(t, chostusbRespEmpty)
	t.Cleanup(func() { inv_testing.DeleteResource(t, hostusbResIDEmpty) })

	expHostusb := createresreq.GetHostusb()
	expHostusb.ResourceId = hostusbResID

	expHostusbEmpty := createresreqEmpty.GetHostusb()
	expHostusbEmpty.ResourceId = hostusbResIDEmpty

	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*computev1.HostusbResource
		valid     bool
	}{
		"NoFilter": {
			in:        &inv_v1.ResourceFilter{},
			resources: []*computev1.HostusbResource{expHostusb, expHostusbEmpty},
			valid:     true,
		},
		"NoFilterOrderByResourceID": {
			in: &inv_v1.ResourceFilter{
				OrderBy: hostusb.FieldResourceID,
			},
			resources: []*computev1.HostusbResource{expHostusb, expHostusbEmpty},
			valid:     true,
		},
		"FilterByEmptyResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = ""`, hostusb.FieldResourceID),
			},
			resources: []*computev1.HostusbResource{},
			valid:     true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, hostusb.FieldResourceID, expHostusb.ResourceId),
			},
			resources: []*computev1.HostusbResource{expHostusb},
			valid:     true,
		},
		"FilterHost": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, hostusb.EdgeHost, hostresource.FieldResourceID, host1.ResourceId),
			},
			resources: []*computev1.HostusbResource{expHostusb},
			valid:     true,
		},
		"FilterHostEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, hostusb.EdgeHost),
			},
			valid: true, // HostNic must have a Host
		},
		"FilterByHasHost": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, hostusb.EdgeHost),
			},
			resources: []*computev1.HostusbResource{expHostusb, expHostusbEmpty},
			valid:     true,
		},
		"FilterByHasHostHasSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, hostusb.EdgeHost, hostresource.EdgeSite),
			},
			resources: []*computev1.HostusbResource{expHostusb},
			valid:     true,
		},
		"FilterWithOffsetLimit1": {
			in: &inv_v1.ResourceFilter{
				Offset: 2,
				Limit:  0,
			},
			valid: true,
		},
		"FilterWithOffsetLimit2": {
			in: &inv_v1.ResourceFilter{
				Offset: 0,
				Limit:  5,
			},
			resources: []*computev1.HostusbResource{expHostusb, expHostusbEmpty},
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Hostusb{}} // Set the resource kind
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterHostusb() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterHostusb() succeeded but should have failed")
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
						"FilterHostusb() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListHostusb() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListHostusb() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*computev1.HostusbResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetHostusb())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					hostusbEdgesOnlyResourceID(expected)
					hostusbEdgesOnlyResourceID(resources[i])

					// Skip check of CreatedAt and UpdatedAt.
					resources[i].CreatedAt = expected.CreatedAt
					resources[i].UpdatedAt = expected.UpdatedAt
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListHostusb() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func hostusbEdgesOnlyResourceID(expected *computev1.HostusbResource) {
	if expected.Host != nil {
		expected.Host = &computev1.HostResource{ResourceId: expected.Host.ResourceId}
	}
}

func Test_NestedFilterHostusb(t *testing.T) {
	region1 := inv_testing.CreateRegion(t, nil)
	site1 := inv_testing.CreateSite(t, region1, nil)
	host1 := inv_testing.CreateHost(t, site1, nil)
	host2 := inv_testing.CreateHost(t, nil, nil)

	hostUsb1 := inv_testing.CreateHostusb(t, host1)
	hostUsb1.Host = host1
	hostUsb2 := inv_testing.CreateHostusb(t, host2)
	hostUsb2.Host = host2

	testcases := map[string]struct {
		in                *inv_v1.ResourceFilter
		resources         []*computev1.HostusbResource
		valid             bool
		expectedCodeError codes.Code
	}{
		"FilterBySiteID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s = %q`, hostusb.EdgeHost, hostresource.EdgeSite,
					siteresource.FieldResourceID, site1.GetResourceId()),
			},
			resources: []*computev1.HostusbResource{hostUsb1},
			valid:     true,
		},
		"FilterBySiteEmpty": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s.%s)`, hostusb.EdgeHost, hostresource.EdgeSite),
			},
			resources: []*computev1.HostusbResource{hostUsb2},
			valid:     true,
		},
		"FilterByHasSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, hostusb.EdgeHost, hostresource.EdgeSite),
			},
			resources: []*computev1.HostusbResource{hostUsb1},
			valid:     true,
		},
		"FilterByRegionID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s = %q`, hostusb.EdgeHost, hostresource.EdgeSite,
					siteresource.EdgeRegion, regionresource.FieldResourceID, region1.GetResourceId()),
			},
			resources: []*computev1.HostusbResource{hostUsb1},
			valid:     true,
		},
		"FailTooDeep": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s.%s.%s = %q`, hostusb.EdgeHost, hostresource.EdgeSite,
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Hostusb{}} // Set the resource kind

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
						"FilterHostusb() failed - want: %s, got: %s",
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

				resources := make([]*computev1.HostusbResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetHostusb())
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

func Test_StrongRelations_On_Delete_HostUsb(t *testing.T) {
	host := inv_testing.CreateHost(t, nil, nil)
	inv_testing.CreateHostusb(t, host)

	err := inv_testing.HardDeleteHostAndReturnError(t, host.ResourceId)
	assertStrongRelationError(t, err, "violates foreign key constraint")
}

func TestHostUsbMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				parent := dao.CreateHost(t, tenantID)
				child := dao.CreateHostUsb(t, tenantID, parent)
				res, err := util.WrapResource(child)
				require.NoError(t, err)
				return child.GetResourceId(), res
			},
		},
	})
}

func TestDeleteResources_HostUSBs(t *testing.T) {
	suite.Run(t, &struct{ hardDeleteAllResourcesSuite }{
		hardDeleteAllResourcesSuite: hardDeleteAllResourcesSuite{
			createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
				tenantID := uuid.NewString()
				host := dao.CreateHost(t, tenantID)
				return tenantID, len(
					[]any{
						dao.CreateHostUsbNoCleanup(t, tenantID, host),
						dao.CreateHostUsbNoCleanup(t, tenantID, host),
						dao.CreateHostUsbNoCleanup(t, tenantID, host),
					},
				)
			},
			resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_HOSTUSB,
		},
	})
}
