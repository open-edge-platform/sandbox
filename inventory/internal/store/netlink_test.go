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

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/endpointresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/hostresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/netlinkresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/regionresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/store"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	network_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/network/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/filters"
)

func Test_Create_Get_Delete_Netlink(t *testing.T) {
	src := inv_testing.CreateEndpoint(t, nil)
	dst := inv_testing.CreateEndpoint(t, nil)

	testcases := map[string]struct {
		in    *network_v1.NetlinkResource
		valid bool
	}{
		"CreateGoodNetlink": {
			in: &network_v1.NetlinkResource{
				Name:         "Netlink 1",
				DesiredState: network_v1.NetlinkState_NETLINK_STATE_ONLINE,
				Src:          src,
				Dst:          dst,
			},
			valid: true,
		},
		"CreateBadNetlinkWithResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &network_v1.NetlinkResource{
				ResourceId:   "netlink-12345678",
				Name:         "Netlink 2",
				DesiredState: network_v1.NetlinkState_NETLINK_STATE_ONLINE,
				Src:          src,
				Dst:          dst,
			},
			valid: false,
		},
		"CreateBadNetlinkWithInvalidResourceIdSet": {
			// This tests case verifies that create requests with a invalid resource ID
			// already set are rejected.
			in: &network_v1.NetlinkResource{
				ResourceId:   "net-link-12345678",
				Name:         "Netlink 2",
				DesiredState: network_v1.NetlinkState_NETLINK_STATE_ONLINE,
				Src:          src,
				Dst:          dst,
			},
			valid: false,
		},
		"CreateBadNetlink": {
			in:    &network_v1.NetlinkResource{},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Netlink{Netlink: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// create netlink
			cnetlinkResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
			netlinkResID := cnetlinkResp.GetNetlink().GetResourceId()

			if err != nil {
				if tc.valid {
					t.Errorf("CreateNetlink() failed: %s", err)
				}
			} else {
				tc.in.ResourceId = netlinkResID // Update with created resource ID.
				tc.in.CreatedAt = cnetlinkResp.GetNetlink().GetCreatedAt()
				tc.in.UpdatedAt = cnetlinkResp.GetNetlink().GetUpdatedAt()
				assertSameResource(t, createresreq, cnetlinkResp, nil)
				if !tc.valid {
					t.Errorf("CreateNetlink() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get non-existent first
				_, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, "netlink-12345678")
				require.Error(t, err)

				// get netlink
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, netlinkResID)
				require.NoError(t, err, "GetNetlink() failed")

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetNetlink()); !eq {
					t.Errorf("GetNetlink() data not equal: %v", diff)
				}

				// delete non-existent first
				err = inv_testing.DeleteResourceAndReturnError(t, "netlink-12345678")
				require.Error(t, err)

				// Remove netlink.
				inv_testing.HardDeleteNetlink(t, netlinkResID)

				// get after complete Delete of netlink, should fail as Netlink is 2-phase deleted
				_, err = inv_testing.TestClients[inv_testing.RMClient].Get(ctx, netlinkResID)
				require.Error(t, err, "Failure - Netlink was not deleted, but should be deleted")
			}
		})
	}
}

func Test_UpdateNetlink(t *testing.T) {
	src := inv_testing.CreateEndpoint(t, nil)
	dst := inv_testing.CreateEndpoint(t, nil)

	src1 := inv_testing.CreateEndpoint(t, nil)
	dst1 := inv_testing.CreateEndpoint(t, nil)

	// create Netlink to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Netlink{
			Netlink: &network_v1.NetlinkResource{
				Name:         "Netlink 2",
				DesiredState: network_v1.NetlinkState_NETLINK_STATE_ONLINE,
				Src:          src,
				Dst:          dst,
			},
		},
	}

	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cnetlinkResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	netlinkResID := inv_testing.GetResourceIDOrFail(t, cnetlinkResp)
	t.Cleanup(func() { inv_testing.HardDeleteNetlink(t, netlinkResID) })

	testcases := map[string]struct {
		in         *network_v1.NetlinkResource
		resourceID string
		fieldMask  *fieldmaskpb.FieldMask
		valid      bool
	}{
		"UpdateNetlink1": {
			in: &network_v1.NetlinkResource{
				CurrentState: network_v1.NetlinkState_NETLINK_STATE_OFFLINE,
			},
			resourceID: netlinkResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{netlinkresource.FieldCurrentState},
			},
			valid: true,
		},
		"UpdateNetlink2": {
			in: &network_v1.NetlinkResource{
				CurrentState: network_v1.NetlinkState_NETLINK_STATE_ERROR,
			},
			resourceID: netlinkResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{netlinkresource.FieldCurrentState, netlinkresource.FieldName},
			},
			valid: true,
		},
		"UpdateNetlink3": {
			in:         &network_v1.NetlinkResource{},
			resourceID: netlinkResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{netlinkresource.FieldName},
			},
			valid: true,
		},
		"UpdateNetlinkSrc": {
			in: &network_v1.NetlinkResource{
				Src: src1,
			},
			resourceID: netlinkResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{netlinkresource.EdgeSrc},
			},
			valid: true,
		},
		"UpdateNetlinkDst": {
			in: &network_v1.NetlinkResource{
				Dst: dst1,
			},
			resourceID: netlinkResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{netlinkresource.EdgeDst},
			},
			valid: true,
		},
		"UpdateNetlinkInvalidFieldMask": {
			in: &network_v1.NetlinkResource{
				CurrentState: network_v1.NetlinkState_NETLINK_STATE_UNSPECIFIED,
			},
			resourceID: netlinkResID,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"INVALID_FIELD"},
			},
			valid: false,
		},
		"UpdateNetlinkFieldMaskNonClearableField": {
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"resource"},
			},
			resourceID: netlinkResID,
			valid:      false,
		},
		"UpdateNoFieldMask": {
			in: &network_v1.NetlinkResource{
				CurrentState: network_v1.NetlinkState_NETLINK_STATE_UNSPECIFIED,
			},
			resourceID: netlinkResID,
			valid:      false,
		},
		"UpdateResourceIDNotFound": {
			in:         &network_v1.NetlinkResource{},
			resourceID: "netlink-12345678",
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{netlinkresource.FieldName},
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Netlink{Netlink: tc.in},
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

			if err != nil {
				if tc.valid {
					t.Errorf("UpdateResource() failed: %s", err)
				}
				// Useless to check if the value was updated if the Update was unsuccessful
				assert.Nil(t, upRes)
				return
			} else if !tc.valid {
				t.Errorf("UpdateResource() succeeded but should have failed")
				return
			}

			// Validate returned resource
			assertSameResource(t, updateresreq, upRes, tc.fieldMask)

			// validate update via a get
			getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, tc.resourceID)
			require.NoError(t, err, "GetResource() failed")

			assertSameResource(t, updateresreq, getresp.GetResource(), tc.fieldMask)
		})
	}
}

func Test_FilterNetlinks(t *testing.T) {
	host := inv_testing.CreateHost(t, nil, nil)
	src1 := inv_testing.CreateEndpoint(t, host)
	dst1 := inv_testing.CreateEndpoint(t, nil)
	src2 := inv_testing.CreateEndpoint(t, nil)
	dst2 := inv_testing.CreateEndpoint(t, nil)

	// create Netlinks to find
	createresreq1 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Netlink{
			Netlink: &network_v1.NetlinkResource{
				Name:         "Netlink 3",
				DesiredState: network_v1.NetlinkState_NETLINK_STATE_ONLINE,
				Src:          src1,
				Dst:          dst1,
			},
		},
	}

	createresreq2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Netlink{
			Netlink: &network_v1.NetlinkResource{
				Name:         "Netlink 4",
				DesiredState: network_v1.NetlinkState_NETLINK_STATE_ONLINE,
				Src:          src2,
				Dst:          dst2,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cnetlinkResp1, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq1)
	netlink1ResID := inv_testing.GetResourceIDOrFail(t, cnetlinkResp1)
	cnetlinkResp2, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq2)
	netlink2ResID := inv_testing.GetResourceIDOrFail(t, cnetlinkResp2)
	t.Cleanup(func() { inv_testing.HardDeleteNetlink(t, netlink1ResID) })
	t.Cleanup(func() { inv_testing.HardDeleteNetlink(t, netlink2ResID) })

	expNetlink1 := createresreq1.GetNetlink()
	expNetlink1.ResourceId = netlink1ResID

	expNetlink2 := createresreq2.GetNetlink()
	expNetlink2.ResourceId = netlink2ResID

	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*network_v1.NetlinkResource
		valid     bool
	}{
		"NoFilter": {
			in:        &inv_v1.ResourceFilter{},
			resources: []*network_v1.NetlinkResource{expNetlink1, expNetlink2},
			valid:     true,
		},
		"NoFilterOrderByResourceID": {
			in: &inv_v1.ResourceFilter{
				OrderBy: netlinkresource.FieldResourceID,
			},
			resources: []*network_v1.NetlinkResource{expNetlink1, expNetlink2},
			valid:     true,
		},
		"FilterByEmptyResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = ""`, netlinkresource.FieldResourceID),
			},
			resources: []*network_v1.NetlinkResource{},
			valid:     true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, netlinkresource.FieldResourceID, expNetlink1.ResourceId),
			},
			resources: []*network_v1.NetlinkResource{expNetlink1},
			valid:     true,
		},
		"FilterByDstByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, netlinkresource.EdgeDst, netlinkresource.FieldResourceID, dst1.ResourceId),
			},
			resources: []*network_v1.NetlinkResource{expNetlink1},
			valid:     true,
		},
		"FilterByHasDst": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, netlinkresource.EdgeDst),
			},
			resources: []*network_v1.NetlinkResource{expNetlink1, expNetlink2},
			valid:     true,
		},
		"FilterByHasSrcHasHost": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, netlinkresource.EdgeSrc, endpointresource.EdgeHost),
			},
			resources: []*network_v1.NetlinkResource{expNetlink1},
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
			resources: []*network_v1.NetlinkResource{expNetlink1, expNetlink2},
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Netlink{}} // Set the resource kind
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterNetlink() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterNetlink() succeeded but should have failed")
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
						"FilterNetlink() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListNetlink() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListNetlink() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*network_v1.NetlinkResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetNetlink())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					netlinkEdgesOnlyResourceID(expected)
					netlinkEdgesOnlyResourceID(resources[i])

					// We need to skip validation of CreatedAt and UpdatedAt
					resources[i].CreatedAt = expected.CreatedAt
					resources[i].UpdatedAt = expected.UpdatedAt
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListNetlink() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func netlinkEdgesOnlyResourceID(expected *network_v1.NetlinkResource) {
	if expected.Src != nil {
		expected.Src = &network_v1.EndpointResource{ResourceId: expected.Src.ResourceId}
	}
	if expected.Dst != nil {
		expected.Dst = &network_v1.EndpointResource{ResourceId: expected.Dst.ResourceId}
	}
}

func Test_NestedFilterNetlink(t *testing.T) {
	src1 := inv_testing.CreateEndpoint(t, nil)
	dst1 := inv_testing.CreateEndpoint(t, nil)
	src2 := inv_testing.CreateEndpoint(t, nil)
	dst2 := inv_testing.CreateEndpoint(t, nil)

	// create Netlinks to find
	createresreq1 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Netlink{
			Netlink: &network_v1.NetlinkResource{
				Name:         "Netlink 3",
				DesiredState: network_v1.NetlinkState_NETLINK_STATE_ONLINE,
				Src:          src1,
				Dst:          dst1,
			},
		},
	}

	createresreq2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Netlink{
			Netlink: &network_v1.NetlinkResource{
				Name:         "Netlink 4",
				DesiredState: network_v1.NetlinkState_NETLINK_STATE_ONLINE,
				Src:          src2,
				Dst:          dst2,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cnetlinkResp1, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq1)
	netlink1ResID := inv_testing.GetResourceIDOrFail(t, cnetlinkResp1)
	cnetlinkResp2, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq2)
	netlink2ResID := inv_testing.GetResourceIDOrFail(t, cnetlinkResp2)
	t.Cleanup(func() { inv_testing.HardDeleteNetlink(t, netlink1ResID) })
	t.Cleanup(func() { inv_testing.HardDeleteNetlink(t, netlink2ResID) })

	expNetlink1 := createresreq1.GetNetlink()
	expNetlink1.ResourceId = netlink1ResID

	expNetlink2 := createresreq2.GetNetlink()
	expNetlink2.ResourceId = netlink2ResID

	testcases := map[string]struct {
		in                *inv_v1.ResourceFilter
		resources         []*network_v1.NetlinkResource
		valid             bool
		expectedCodeError codes.Code
	}{
		"FilterEndpointID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, netlinkresource.EdgeSrc, endpointresource.FieldResourceID,
					src1.GetResourceId()),
			},
			resources: []*network_v1.NetlinkResource{expNetlink1},
			valid:     true,
		},
		"FailTooDeep": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s.%s.%s = %q`, netlinkresource.EdgeSrc, endpointresource.EdgeHost,
					hostresource.EdgeSite, siteresource.EdgeRegion, regionresource.EdgeParentRegion,
					regionresource.FieldResourceID, ""),
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Netlink{}} // Set the resource kind

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
						"FilterNetlink() failed - want: %s, got: %s",
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

				resources := make([]*network_v1.NetlinkResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetNetlink())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					// We need to skip validation of CreatedAt and UpdatedAt
					resources[i].CreatedAt = expected.CreatedAt
					resources[i].UpdatedAt = expected.UpdatedAt
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListInstances() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func Test_NetlinkEnumStateMap(t *testing.T) {
	v, err := store.NetlinkEnumStateMap("invalid_input", int32(network_v1.NetlinkState_NETLINK_STATE_ONLINE))
	assert.Error(t, err)
	assert.Nil(t, v)
}

func TestSoftDeleteResources_NetLinks(t *testing.T) {
	suite.Run(t, &softDeleteAllResourcesSuite{
		createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
			tenantID := uuid.NewString()
			return tenantID, len([]any{
				dao.CreateNetLink(t, tenantID, false),
				dao.CreateNetLink(t, tenantID, false),
				dao.CreateNetLink(t, tenantID, false),
			})
		},
		resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_NETLINK,
		deletedClause: filters.ValEq(
			network_v1.NetlinkResourceFieldDesiredState, network_v1.NetlinkState_NETLINK_STATE_DELETED),
		notDeletedClause: filters.ValNotEq(
			network_v1.NetlinkResourceFieldDesiredState, network_v1.NetlinkState_NETLINK_STATE_DELETED),
	})
}

func TestHardDeleteResources_NetLinks(t *testing.T) {
	suite.Run(t, &hardDeleteAllResourcesSuite{
		createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
			tenantID := uuid.NewString()
			return tenantID, len([]any{
				dao.CreateNetLink(t, tenantID, false),
				dao.CreateNetLink(t, tenantID, false),
				dao.CreateNetLink(t, tenantID, false),
			})
		},
		resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_NETLINK,
	})
}
