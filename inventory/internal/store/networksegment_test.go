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

	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/networksegment"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/regionresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	location_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/location/v1"
	network_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/network/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

const (
	name1              = "Network Segment 1"
	name2              = "Network Segment 2"
	name3              = "Network Segment 3"
	name4              = "Network Segment 4"
	vlanID1      int32 = 2
	vlanID2      int32 = 21
	wrongVlanID  int32 = 1
	wrongVlanID2 int32 = -11
	wrongVlanID3 int32 = 5000
	wrongVlanID4 int32 = 4095
	zeroVlanID   int32 = 0 // it is 0 value by default
)

func Test_Create_Get_Delete_NetworkSegment(t *testing.T) {
	site := inv_testing.CreateSite(t, nil, nil)

	testcases := map[string]struct {
		in    *network_v1.NetworkSegment
		valid bool
	}{
		"CreateGoodNetworkSegment": {
			in: &network_v1.NetworkSegment{
				Name:   name1,
				Site:   site,
				VlanId: vlanID1,
			},
			valid: true,
		},
		"CreateBadNetworkSegmentWithResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &network_v1.NetworkSegment{
				ResourceId: "netseg-12345678",
				Name:       name2,
				Site:       site,
				VlanId:     vlanID2,
			},
			valid: false,
		},
		"CreateBadNetworkSegment": {
			in:    &network_v1.NetworkSegment{},
			valid: false,
		},
		"CreateGoodNetworkSegmentSingleVlan": {
			in: &network_v1.NetworkSegment{
				Name:   name1,
				Site:   site,
				VlanId: zeroVlanID,
			},
			valid: true,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_NetworkSegment{NetworkSegment: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// create network
			cnetworkResp, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
			networksegResID := cnetworkResp.GetNetworkSegment().GetResourceId()

			if err != nil {
				if tc.valid {
					t.Errorf("CreateNetworkSegment() failed: %s", err)
				}
			} else {
				tc.in.ResourceId = networksegResID // Update with created resource ID.
				tc.in.CreatedAt = cnetworkResp.GetNetworkSegment().GetCreatedAt()
				tc.in.UpdatedAt = cnetworkResp.GetNetworkSegment().GetUpdatedAt()
				assertSameResource(t, createresreq, cnetworkResp, nil)
				if !tc.valid {
					t.Errorf("CreateNetworkSegment() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get network
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, networksegResID)
				require.NoError(t, err, "GetNetworkSegment() failed")

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetNetworkSegment()); !eq {
					t.Errorf("GetNetworkSegment() data not equal: %v", diff)
				}

				// delete non-existent first
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, "netseg-12345678")
				require.Error(t, err)

				// delete network from API
				inv_testing.DeleteResource(t, networksegResID)

				// get after complete Delete of network, should fail as Network is 2-phase deleted
				_, err = inv_testing.TestClients[inv_testing.RMClient].Get(ctx, networksegResID)
				require.Error(t, err, "Failure - NetworkSegment was not deleted, but should be deleted")
			}
		})
	}
}

func Test_UpdateNetworkSegment(t *testing.T) {
	site := inv_testing.CreateSite(t, nil, nil)
	site2 := inv_testing.CreateSite(t, nil, nil)

	// create Network to update and set again the edge nilled by the helper
	netseg1 := inv_testing.CreateNetworkSegment(t, name1, site, vlanID1)
	netseg1.Site = site
	inv_testing.CreateNetworkSegment(t, name1, site2, vlanID1)

	testcases := map[string]struct {
		in           *network_v1.NetworkSegment
		resourceID   string
		fieldMask    *fieldmaskpb.FieldMask
		valid        bool
		expErrorCode codes.Code
	}{
		"UpdateNetworkSegmentPUT": {
			in: &network_v1.NetworkSegment{
				Name:   name4,
				Site:   site,
				VlanId: vlanID2,
			},
			resourceID:   netseg1.ResourceId,
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateNetworkSegment1": {
			in: &network_v1.NetworkSegment{
				Name:   name1,
				Site:   site,
				VlanId: vlanID1,
			},
			resourceID: netseg1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"vlan_id", "name", "site"},
			},
			valid: true,
		},
		"UpdateNetworkSegment2": {
			in: &network_v1.NetworkSegment{
				VlanId: vlanID2,
				Site:   site,
			},
			resourceID: netseg1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"vlan_id", "site"},
			},
			valid: true,
		},
		// Site remains the same.
		"UpdateNetworkSegment3": {
			in: &network_v1.NetworkSegment{
				VlanId: vlanID2,
			},
			resourceID: netseg1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"vlan_id"},
			},
			valid: true,
		},
		"UpdateNetworkSegment4": {
			in: &network_v1.NetworkSegment{
				Site:   site2,
				VlanId: vlanID2 + 1,
			},
			resourceID: netseg1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"vlan_id", "site"},
			},
			valid: true,
		},
		"UpdateNetworkSegment5": {
			in: &network_v1.NetworkSegment{
				Site: site,
			},
			resourceID: netseg1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"site"},
			},
			valid: true,
		},
		"UpdateNetworkInvalidFieldMask": {
			in: &network_v1.NetworkSegment{
				Name: name2,
			},
			resourceID: netseg1.ResourceId,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"INVALID_FIELD"},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateResourceIDNotFound": {
			in: &network_v1.NetworkSegment{
				Name: name1,
			},
			resourceID: "netseg-12345678",
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{networksegment.FieldName},
			},
			valid:        false,
			expErrorCode: codes.NotFound,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_NetworkSegment{NetworkSegment: tc.in},
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

func Test_FilterNetworkSegments(t *testing.T) {
	site1 := inv_testing.CreateSite(t, nil, nil)
	site2 := inv_testing.CreateSite(t, nil, nil)

	// Setting again the edge nilled by the helper
	expNetwork1 := inv_testing.CreateNetworkSegment(t, name3, site1, zeroVlanID)
	expNetwork1.Site = site1
	expNetwork2 := inv_testing.CreateNetworkSegment(t, name4, site2, zeroVlanID)
	expNetwork2.Site = site2

	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*network_v1.NetworkSegment
		valid     bool
	}{
		"NoFilter": {
			in:        &inv_v1.ResourceFilter{},
			resources: []*network_v1.NetworkSegment{expNetwork1, expNetwork2},
			valid:     true,
		},
		"NoFilterOrderByResourceID": {
			in: &inv_v1.ResourceFilter{
				OrderBy: networksegment.FieldResourceID,
			},
			resources: []*network_v1.NetworkSegment{expNetwork1, expNetwork2},
			valid:     true,
		},
		"FilterByEmptyResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = ""`, networksegment.FieldResourceID),
			},
			valid: true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, networksegment.FieldResourceID, expNetwork1.ResourceId),
			},
			resources: []*network_v1.NetworkSegment{expNetwork1},
			valid:     true,
		},
		"FilterSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, networksegment.EdgeSite,
					siteresource.FieldResourceID, site1.GetResourceId()),
			},
			resources: []*network_v1.NetworkSegment{expNetwork1},
			valid:     true,
		},
		"FilterByHasSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, networksegment.EdgeSite),
			},
			resources: []*network_v1.NetworkSegment{expNetwork1, expNetwork2},
			valid:     true,
		},
		"FilterByHasSiteRegion": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, networksegment.EdgeSite, siteresource.EdgeRegion),
			},
			resources: []*network_v1.NetworkSegment{},
			valid:     true,
		},
		"FilterWithOffsetLimit": {
			in: &inv_v1.ResourceFilter{
				Offset: 5,
				Limit:  0,
			},
			valid: true,
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_NetworkSegment{}} // Set the resource kind
			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterSegment) failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterNetworkSegment() succeeded but should have failed")
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
						"FilterNetworkSegment() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.APIClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListNetworkSegment() failed: %s", err)
				}
			} else {
				if !tc.valid {
					t.Errorf("ListNetworkSegment() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*network_v1.NetworkSegment, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetNetworkSegment())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					netsegEdgesOnlyResourceID(expected)
					netsegEdgesOnlyResourceID(resources[i])

					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListNetworkSegment() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func netsegEdgesOnlyResourceID(expected *network_v1.NetworkSegment) {
	if expected.Site != nil {
		expected.Site = &location_v1.SiteResource{ResourceId: expected.Site.ResourceId}
	}
}

// This test case verifies that it is not possible to create a Network Segment with
// VLAN ID out of required range. Test case verifies corner cases, when VlanID is being
// set to 1, 4095, negative number, and 0 (to existing Site with non-zero Vlan IDs).
func Test_BadNetworkSegments(t *testing.T) {
	site := inv_testing.CreateSite(t, nil, nil)

	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Verifying Create behavior
	createresreq1 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_NetworkSegment{
			NetworkSegment: &network_v1.NetworkSegment{
				Name:   name1,
				Site:   site,
				VlanId: wrongVlanID,
			},
		},
	}
	// Create should fail because of wrong VLAN ID (1 is a reserved value and can't be set)
	_, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq1)
	require.Error(t, err)

	createresreq2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_NetworkSegment{
			NetworkSegment: &network_v1.NetworkSegment{
				Name:   name2,
				Site:   site,
				VlanId: wrongVlanID2,
			},
		},
	}
	// Create should fail because of wrong VLAN ID - it can't be a negative number
	_, err = inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq2)
	require.Error(t, err)

	createresreq3 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_NetworkSegment{
			NetworkSegment: &network_v1.NetworkSegment{
				Name:   name2,
				Site:   site,
				VlanId: wrongVlanID3,
			},
		},
	}
	// Create should fail because of wrong VLAN ID - number is out of range (5'000 is higher than the upperbound, 4'095)
	_, err = inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq3)
	require.Error(t, err)

	createresreq4 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_NetworkSegment{
			NetworkSegment: &network_v1.NetworkSegment{
				Name:   name2,
				Site:   site,
				VlanId: wrongVlanID4,
			},
		},
	}
	// Create should fail because of wrong VLAN ID (4'095 is a reserved value and can't be set)
	_, err = inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq4)
	require.Error(t, err)

	// Verifying Update (with PUT) behavior
	netseg := inv_testing.CreateNetworkSegment(t, name1, site, zeroVlanID)
	// Setting again the edge nilled by the helper
	netseg.Site = site

	// Update should fail because of wrong VLAN ID (1 is a reserved value and can't be set)
	_, err = inv_testing.TestClients[inv_testing.APIClient].Update(
		ctx,
		netseg.ResourceId,
		&fieldmaskpb.FieldMask{Paths: []string{networksegment.FieldVlanID}},
		createresreq1,
	)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	// checking that the Network Segment stayed unchanged
	checkNetworkSegmentIsEqual(t, netseg.ResourceId, netseg)

	// Update should fail because of wrong VLAN ID - it can't be a negative number
	_, err = inv_testing.TestClients[inv_testing.APIClient].Update(
		ctx,
		netseg.ResourceId,
		&fieldmaskpb.FieldMask{Paths: []string{networksegment.FieldVlanID}},
		createresreq2,
	)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	// checking that the Network Segment stayed unchanged
	checkNetworkSegmentIsEqual(t, netseg.ResourceId, netseg)

	// Update should fail because of wrong VLAN ID - number is out of range (5'000 is higher than the upperbound, 4'095)
	_, err = inv_testing.TestClients[inv_testing.APIClient].Update(
		ctx,
		netseg.ResourceId,
		&fieldmaskpb.FieldMask{Paths: []string{networksegment.FieldVlanID}},
		createresreq3,
	)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	// checking that the Network Segment stayed unchanged
	checkNetworkSegmentIsEqual(t, netseg.ResourceId, netseg)

	// Update should fail because of wrong VLAN ID (4'095 is a reserved value and can't be set)
	_, err = inv_testing.TestClients[inv_testing.APIClient].Update(
		ctx,
		netseg.ResourceId,
		&fieldmaskpb.FieldMask{Paths: []string{networksegment.FieldVlanID}},
		createresreq4,
	)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	// checking that the Network Segment stayed unchanged
	checkNetworkSegmentIsEqual(t, netseg.ResourceId, netseg)
}

// checkNetworkSegmentIsEqual checks (with Get) if Network Segment with resourceID is
// equal to the Network Segment passed to this function, netseg.
func checkNetworkSegmentIsEqual(tb testing.TB, resourceID string, netseg *network_v1.NetworkSegment) {
	tb.Helper()

	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// checking that the Network Segment stayed unchanged
	getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, resourceID)
	require.NoError(tb, err, "GetResource() failed")
	if eq, diff := inv_testing.ProtoEqualOrDiff(netseg, getresp.GetResource().GetNetworkSegment()); !eq {
		tb.Errorf("Resources are different: %v", diff)
	}
}

// This test verifies that it is not possible to create a Network Segment with the same Vlan ID or a VlanID=0
// on the Site, where already exists Network Segment with non-zero Vlan ID.
func Test_CreateNetworkSegmentsInSiteWithNonZeroVlan(t *testing.T) {
	site := inv_testing.CreateSite(t, nil, nil)

	// create base Network Segment
	netseg := inv_testing.CreateNetworkSegment(t, name1, site, vlanID2)
	// Setting again the edge nilled by the helper
	netseg.Site = site

	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// create Network Segment with the same VlanID on the same site
	createresreq2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_NetworkSegment{
			NetworkSegment: &network_v1.NetworkSegment{
				Name:   name2,
				Site:   site,
				VlanId: vlanID2,
			},
		},
	}
	// Create should fail because of duplicated Vlan ID
	_, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq2)
	require.Error(t, err)
	// checking that the Network Segment stayed unchanged
	checkNetworkSegmentIsEqual(t, netseg.ResourceId, netseg)

	// create Network Segment with VlanID equal to 0
	createresreq3 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_NetworkSegment{
			NetworkSegment: &network_v1.NetworkSegment{
				Name:   name3,
				Site:   site,
				VlanId: zeroVlanID,
			},
		},
	}
	// Create should fail because the site already has non-zero Vlan ID (i.e., VlanID=0 can be only one per Site)
	_, err = inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq3)
	require.Error(t, err)
	// checking that the Network Segment stayed unchanged
	checkNetworkSegmentIsEqual(t, netseg.ResourceId, netseg)

	newNetSeg := network_v1.NetworkSegment{
		Name:   name4,
		Site:   site,
		VlanId: vlanID1,
	}
	// create Network Segment with different VlanID
	createresreq4 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_NetworkSegment{
			NetworkSegment: &newNetSeg,
		},
	}
	// Create should succeed
	cnetworkResp4, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq4)
	require.NoError(t, err)
	createdNetSeg := cnetworkResp4.GetNetworkSegment()
	networksec4ResID := createdNetSeg.GetResourceId()
	t.Cleanup(func() { inv_testing.DeleteResource(t, networksec4ResID) })
	// checking that the Network Segment is changed
	checkNetworkSegmentIsEqual(t, networksec4ResID, createdNetSeg)
}

// This test verifies that it is not possible to create any further Network Segments on the Site, which already
// contains Network Segment with VlanID=0.
func Test_CreateNetworkSegmentsInSiteWithZeroVlan(t *testing.T) {
	site := inv_testing.CreateSite(t, nil, nil)

	// create base Network Segment
	netseg := inv_testing.CreateNetworkSegment(t, name1, site, zeroVlanID)
	// Setting again the edge nilled by the helper
	netseg.Site = site

	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// create Network Segment with different VlanID on the same Site
	createresreq2 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_NetworkSegment{
			NetworkSegment: &network_v1.NetworkSegment{
				Name:   name2,
				Site:   site,
				VlanId: vlanID2,
			},
		},
	}
	// Create should fail because of Site already containing NetworkSegment with VlanID=0 (i.e., no other
	// VLANs can be present at the Site)
	_, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq2)
	require.Error(t, err)
	checkNetworkSegmentIsEqual(t, netseg.ResourceId, netseg)

	// create Network Segment with VlanID equal to 0
	createresreq3 := &inv_v1.Resource{
		Resource: &inv_v1.Resource_NetworkSegment{
			NetworkSegment: &network_v1.NetworkSegment{
				Name:   name3,
				Site:   site,
				VlanId: zeroVlanID,
			},
		},
	}
	// Create should fail because the site already has VlanID=0 (i.e., VlanID=0 can be only one per Site)
	_, err = inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq3)
	require.Error(t, err)
	checkNetworkSegmentIsEqual(t, netseg.ResourceId, netseg)
}

// This test verifies that it is not possible to move Network Segment from one Site to the other Site, which
// contains Network Segment with VlanID=0. It also verifies that it is not possible to move Network
// Segment with VlanID=0 to the other Site containing Network Segment with VlanID=0.
func Test_UpdateNetworkSegmentSiteWithPATCH(t *testing.T) {
	site := inv_testing.CreateSite(t, nil, nil)
	site2 := inv_testing.CreateSite(t, nil, nil)

	netseg1 := inv_testing.CreateNetworkSegment(t, name2, site, vlanID1)
	// Setting again the edge nilled by the helper
	netseg1.Site = site
	inv_testing.CreateNetworkSegment(t, name1, site2, zeroVlanID)

	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	updNetSeg := &network_v1.NetworkSegment{
		Site:   site,
		VlanId: vlanID2,
	}
	// updating name and VlanID of the first Network Segment - should succeed
	upResp, err := inv_testing.TestClients[inv_testing.APIClient].Update(
		ctx,
		netseg1.ResourceId,
		&fieldmaskpb.FieldMask{
			Paths: []string{networksegment.FieldVlanID, networksegment.EdgeSite},
		},
		&inv_v1.Resource{
			Resource: &inv_v1.Resource_NetworkSegment{
				NetworkSegment: updNetSeg,
			},
		},
	)
	require.NoError(t, err)
	// checking that the Network Segment has been updated properly
	// (i.e., expected Network Segment is stored in updNetSeg2 variable)
	updatedNetSeg := upResp.GetNetworkSegment()
	updNetSeg.Name = name2
	updNetSeg.ResourceId = netseg1.ResourceId
	updNetSeg.TenantId = netseg1.TenantId
	updNetSeg.CreatedAt = updatedNetSeg.CreatedAt
	updNetSeg.UpdatedAt = updatedNetSeg.UpdatedAt
	checkNetworkSegmentIsEqual(t, netseg1.ResourceId, updatedNetSeg)

	// updating Site and setting VlanID to 0 for the first Network Segment - should fail
	_, err = inv_testing.TestClients[inv_testing.APIClient].Update(
		ctx,
		netseg1.ResourceId,
		&fieldmaskpb.FieldMask{
			Paths: []string{networksegment.FieldVlanID, networksegment.EdgeSite, networksegment.FieldName},
		},
		&inv_v1.Resource{
			Resource: &inv_v1.Resource_NetworkSegment{
				NetworkSegment: &network_v1.NetworkSegment{
					Name:   name4,
					Site:   site2,
					VlanId: zeroVlanID,
				},
			},
		},
	)
	require.Error(t, err)
	// checking that the Network Segment stayed unchanged
	checkNetworkSegmentIsEqual(t, netseg1.ResourceId, updNetSeg)

	// updating Site and setting a non-zero VlanID (than already present in the Site) - should fail,
	// because Site already contains Network Segment with VlanID=0
	_, err = inv_testing.TestClients[inv_testing.APIClient].Update(
		ctx,
		netseg1.ResourceId,
		&fieldmaskpb.FieldMask{
			Paths: []string{networksegment.FieldVlanID, networksegment.EdgeSite},
		},
		&inv_v1.Resource{
			Resource: &inv_v1.Resource_NetworkSegment{
				NetworkSegment: &network_v1.NetworkSegment{
					Site:   site2,
					VlanId: vlanID2,
				},
			},
		},
	)
	require.Error(t, err)
	// checking that the Network Segment stayed unchanged
	checkNetworkSegmentIsEqual(t, netseg1.ResourceId, updNetSeg)
}

func Test_NestedFilterNetworkSegments(t *testing.T) {
	region := inv_testing.CreateRegion(t, nil)
	site1 := inv_testing.CreateSite(t, region, nil)
	site2 := inv_testing.CreateSite(t, nil, nil)

	// Setting again the edge nilled by the helper
	expNetwork1 := inv_testing.CreateNetworkSegment(t, name3, site1, zeroVlanID)
	expNetwork1.Site = site1
	expNetwork2 := inv_testing.CreateNetworkSegment(t, name4, site2, zeroVlanID)
	expNetwork2.Site = site2

	testcases := map[string]struct {
		in                *inv_v1.ResourceFilter
		resources         []*network_v1.NetworkSegment
		valid             bool
		expectedCodeError codes.Code
	}{
		"FilterByRegionID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s = %q`, networksegment.EdgeSite, siteresource.EdgeRegion,
					regionresource.FieldResourceID, region.GetResourceId()),
			},
			resources: []*network_v1.NetworkSegment{expNetwork1},
			valid:     true,
		},
		"FilterByEmptyRegion": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s.%s)`, networksegment.EdgeSite, siteresource.EdgeRegion),
			},
			resources: []*network_v1.NetworkSegment{expNetwork2},
			valid:     true,
		},
		"FailTooDeep": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s.%s.%s = %q`, networksegment.EdgeSite, siteresource.EdgeRegion,
					regionresource.EdgeParentRegion, regionresource.EdgeParentRegion, regionresource.FieldResourceID,
					regionresource.FieldResourceID, region.GetResourceId()),
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_NetworkSegment{}} // Set the resource kind

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
						"FilterNetworkSegment() failed - want: %s, got: %s",
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

				resources := make([]*network_v1.NetworkSegment, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetNetworkSegment())
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

func TestNetworkSegmentMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				site := dao.CreateSite(t, tenantID)
				segment := dao.CreateNetworkSegment(t, tenantID, "test-segment", site, 666, true)
				res, err := util.WrapResource(segment)
				require.NoError(t, err)
				return segment.GetResourceId(), res
			},
		},
	})
}

func TestDeleteResources_NetworkSegments(t *testing.T) {
	suite.Run(t, &struct{ hardDeleteAllResourcesSuite }{
		hardDeleteAllResourcesSuite: hardDeleteAllResourcesSuite{
			createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
				tenantID := uuid.NewString()
				site := dao.CreateSite(t, tenantID)
				return tenantID, len(
					[]any{
						dao.CreateNetworkSegment(t, tenantID, "ns1", site, 3456, false),
						dao.CreateNetworkSegment(t, tenantID, "ns1", site, 3457, false),
					},
				)
			},
			resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_NETWORKSEGMENT,
		},
	})
}
