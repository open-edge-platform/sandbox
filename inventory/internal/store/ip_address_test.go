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
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/ipaddressresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/regionresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/store"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	network_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/network/v1"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

func Test_Create_Get_Delete_IPAddress(t *testing.T) {
	// We explicitly set the edges again for the nested eager loading
	host := inv_testing.CreateHost(t, nil, nil)
	hostNic := inv_testing.CreateHostNic(t, host)
	hostNic.Host = host
	// Eager loading is two levels deep
	site := inv_testing.CreateSite(t, nil, nil)
	hostWithSite := inv_testing.CreateHost(t, site, nil)
	hostWithSite.Site = site
	hostNicWithSite := inv_testing.CreateHostNic(t, hostWithSite)
	hostNicWithSite.Host = hostWithSite
	// For error scenarios
	ghostNic := &computev1.HostnicResource{
		ResourceId: "hostnic-12345678",
	}

	testcases := map[string]struct {
		in    *network_v1.IPAddressResource
		valid bool
	}{
		"CreateGoodIPAddress1": {
			in: &network_v1.IPAddressResource{
				Address:      "10.0.0.1/24",
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          hostNic,
			},
			valid: true,
		},
		"CreateGoodIPAddress2": {
			in: &network_v1.IPAddressResource{
				Address:      "fe80:0000:0000:0000:0204:61ff:fe9d:f156/2",
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          hostNic,
			},
			valid: true,
		},
		"CreateGoodIPAddress3": {
			in: &network_v1.IPAddressResource{
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          hostNic,
			},
			valid: true,
		},
		"CreateGoodIPAddress4": {
			in: &network_v1.IPAddressResource{
				Address:      "10.0.0.1/24",
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          hostNicWithSite,
			},
			valid: true,
		},
		// No hostnic
		"CreateBadIPAddress1": {
			in: &network_v1.IPAddressResource{
				Address:      "10.0.0.1/24",
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
			},
			valid: false,
		},
		// Undefined hostnic
		"CreateBadIPAddress2": {
			in: &network_v1.IPAddressResource{
				Address:      "10.0.0.1/24",
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          ghostNic,
			},
			valid: false,
		},
		// Not a prefix
		"CreateBadIPAddress3": {
			in: &network_v1.IPAddressResource{
				Address:      "10.0.0.1",
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          hostNic,
			},
			valid: false,
		},
		// Invalid prefix
		"CreateBadIPAddress4": {
			in: &network_v1.IPAddressResource{
				Address:      "10.0.0.1/",
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          hostNic,
			},
			valid: false,
		},
		// not ipv4 prefix
		"CreateBadIPAddress5": {
			in: &network_v1.IPAddressResource{
				Address:      "10.0.0.1/33",
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          hostNic,
			},
			valid: false,
		},
		// Invalid prefix
		"CreateBadIPAddress6": {
			in: &network_v1.IPAddressResource{
				Address:      "10.0.0.1/aasd",
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          hostNic,
			},
			valid: false,
		},
		// 01 invalid bits but unaccepted by the library as 1
		"CreateBadIPAddress7": {
			in: &network_v1.IPAddressResource{
				Address:      "10.0.0.1/01",
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          hostNic,
			},
			valid: false,
		},
		// 00 invalid bits but not accepted by the library as 0
		"CreateBadIPAddress8": {
			in: &network_v1.IPAddressResource{
				Address:      "10.0.0.1/00",
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          hostNic,
			},
			valid: false,
		},
		// Resource id set on create
		"CreateBadIPAddress9": {
			in: &network_v1.IPAddressResource{
				ResourceId:   "ipaddr-12345678",
				Address:      "10.0.0.1/00",
				DesiredState: network_v1.IPAddressState_IP_ADDRESS_STATE_UNSPECIFIED,
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          hostNic,
			},
			valid: false,
		},
		// invalid resource id
		"CreateBadIPAddress10": {
			in: &network_v1.IPAddressResource{
				ResourceId:   "ip-addr-12345678",
				Address:      "10.0.0.1/00",
				DesiredState: network_v1.IPAddressState_IP_ADDRESS_STATE_UNSPECIFIED,
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          hostNic,
			},
			valid: false,
		},
		// Not a prefix
		"CreateBadIPAddress12": {
			in: &network_v1.IPAddressResource{
				Address:      "fe80:0000:0000:0000:0204:61ff:fe9d:f156",
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          hostNic,
			},
			valid: false,
		},
		// Invalid prefix
		"CreateBadIPAddress13": {
			in: &network_v1.IPAddressResource{
				Address:      "fe80:0000:0000:0000:0204:61ff:fe9d:f156/",
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          hostNic,
			},
			valid: false,
		},
		// not ipv6 prefix
		"CreateBadIPAddress14": {
			in: &network_v1.IPAddressResource{
				Address:      "fe80:0000:0000:0000:0204:61ff:fe9d:f156/129",
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
				Nic:          hostNic,
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Ipaddress{Ipaddress: tc.in},
			}
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// created
			ipResp, err := inv_testing.TestClients[inv_testing.RMClient].Create(ctx, createresreq)
			resourceID := ipResp.GetIpaddress().GetResourceId()

			if err != nil {
				if tc.valid {
					t.Errorf("CreateIPAddress() failed: %s", err)
					t.FailNow()
				}
			} else {
				tc.in.ResourceId = resourceID // Update with created resource ID.
				tc.in.CreatedAt = ipResp.GetIpaddress().GetCreatedAt()
				tc.in.UpdatedAt = ipResp.GetIpaddress().GetUpdatedAt()
				if !tc.valid {
					t.Errorf("CreateIPAddress() succeeded but should have failed")
					t.FailNow()
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(
					ctx, resourceID)
				if err != nil {
					require.NoError(t, err, "GetIPAddress() failed")
				}

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetIpaddress()); !eq {
					t.Errorf("GetIPAddress() data not equal: %v", diff)
					t.FailNow()
				}

				// Delete is unsupported
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(
					ctx,
					resourceID,
				)
				if err == nil {
					require.NoError(t, err, "DeleteIPAddress() should have failed")
				}

				// get after complete Delete of ipaddress, should fail as ipaddress is 2-phase deleted
				_, err = inv_testing.TestClients[inv_testing.APIClient].Get(ctx, resourceID)
				if err != nil {
					require.NoError(t, err, "GetIPAddress() IPAddress was not deleted")
				}

				// hard delete now
				inv_testing.HardDeleteIPAddress(t, resourceID)

				// get should fail now
				_, err = inv_testing.TestClients[inv_testing.APIClient].Get(ctx, resourceID)
				if err == nil {
					t.Errorf("GetIPAddress() IPAddress was deleted")
				}
			}
		})
	}
}

type UpdateTCIPAddr struct {
	in           *network_v1.IPAddressResource
	fieldMask    *fieldmaskpb.FieldMask
	valid        bool
	expErrorCode codes.Code
}

func runUpdateTCForIPAddr(t *testing.T, testcases map[string]UpdateTCIPAddr, resourceID string) {
	t.Helper()

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Ipaddress{Ipaddress: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			upRes, err := inv_testing.TestClients[inv_testing.RMClient].Update(
				ctx,
				resourceID,
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
			assert.Equal(t, resourceID, upRes.GetIpaddress().GetResourceId())

			// validate update via a get
			getresp, err := inv_testing.TestClients[inv_testing.RMClient].Get(
				ctx, resourceID)
			require.NoError(t, err, "GetResource() failed")

			assertSameResource(t, updateresreq, getresp.GetResource(), tc.fieldMask)
		})
	}
}

func Test_UpdateIPAddress(t *testing.T) {
	// We explicitly set the edges again for the nested eager loading
	host := inv_testing.CreateHost(t, nil, nil)
	hostNic := inv_testing.CreateHostNic(t, host)
	hostNic.Host = host
	hostNicNew := inv_testing.CreateHostNic(t, host)
	hostNicNew.Host = host
	ghostNic := &computev1.HostnicResource{
		ResourceId: "hostnic-12345678",
	}

	ipaddress1 := inv_testing.CreateIPAddress(t, hostNic, true)

	testcases := map[string]UpdateTCIPAddr{
		// Set error state.
		"Update1": {
			in: &network_v1.IPAddressResource{
				CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_ERROR,
			},
			valid: true,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ipaddressresource.FieldCurrentState},
			},
		},
		// Update status.
		"Update2": {
			in: &network_v1.IPAddressResource{
				Status: network_v1.IPAddressStatus_IP_ADDRESS_STATUS_CONFIGURED,
			},
			valid: true,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ipaddressresource.FieldStatus},
			},
		},
		// Change config mode.
		"Update3": {
			in: &network_v1.IPAddressResource{
				ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_STATIC,
			},
			valid: true,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ipaddressresource.FieldConfigMethod},
			},
		},
		// Change nic.
		"Update4": {
			in: &network_v1.IPAddressResource{
				Nic: hostNicNew,
			},
			valid: true,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ipaddressresource.EdgeNic},
			},
		},
		// Not defined nic.
		"BadUpdate1": {
			in: &network_v1.IPAddressResource{
				Nic: ghostNic,
			},
			valid:        false,
			expErrorCode: codes.NotFound,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ipaddressresource.EdgeNic},
			},
		},
		// Wrong address.
		"BadUpdate2": {
			in: &network_v1.IPAddressResource{
				Address: "aaa",
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{ipaddressresource.FieldAddress},
			},
		},
		// Wrong fieldmask.
		"BadUpdate3": {
			in: &network_v1.IPAddressResource{
				Address: "10.0.0.1/10",
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"aaa"},
			},
		},
		// Missing fieldmask.
		"BadUpdate4": {
			in: &network_v1.IPAddressResource{
				Address: "10.0.0.1/10",
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
	}

	runUpdateTCForIPAddr(t, testcases, ipaddress1.ResourceId)
}

func Test_FilterIPAddress(t *testing.T) {
	// We explicitly set the edges again for the nested eager loading
	region := inv_testing.CreateRegion(t, nil)
	site1 := inv_testing.CreateSite(t, region, nil)
	site2 := inv_testing.CreateSite(t, region, nil)
	host1 := inv_testing.CreateHost(t, site1, nil)
	host1.Site = site1
	host2 := inv_testing.CreateHost(t, site2, nil)
	host2.Site = site2
	hostNic1 := inv_testing.CreateHostNic(t, host1)
	hostNic1.Host = host1
	hostNic2 := inv_testing.CreateHostNic(t, host2)
	hostNic2.Host = host2
	// Setting again the edge nilled by the helper
	ipaddress1 := inv_testing.CreateIPAddress(t, hostNic1, true)
	ipaddress1.Nic = hostNic1
	ipaddress2 := inv_testing.CreateIPAddress(t, hostNic2, true)
	ipaddress2.Nic = hostNic2

	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*network_v1.IPAddressResource
		valid     bool
	}{
		"NoFilter": {
			in:        &inv_v1.ResourceFilter{},
			resources: []*network_v1.IPAddressResource{ipaddress1, ipaddress2},
			valid:     true,
		},
		"NoFilterOrderByResourceID": {
			in: &inv_v1.ResourceFilter{
				OrderBy: ipaddressresource.FieldResourceID,
			},
			resources: []*network_v1.IPAddressResource{ipaddress1, ipaddress2},
			valid:     true,
		},
		"FilterByEmptyResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = ""`, ipaddressresource.FieldResourceID),
			},
			valid: true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, ipaddressresource.FieldResourceID, ipaddress1.ResourceId),
			},
			resources: []*network_v1.IPAddressResource{ipaddress1},
			valid:     true,
		},
		"FilterNic": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s = %q`, ipaddressresource.EdgeNic,
					ipaddressresource.FieldResourceID, hostNic1.GetResourceId()),
			},
			resources: []*network_v1.IPAddressResource{ipaddress1},
			valid:     true,
		},
		// Not found is returned since Nic can never be empty (mandatory field)
		"FilterNoNic": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s)`, ipaddressresource.EdgeNic),
			},
			valid: true,
		},
		"FilterByHasNic": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, ipaddressresource.EdgeNic),
			},
			resources: []*network_v1.IPAddressResource{ipaddress1, ipaddress2},
			valid:     true,
		},
		"FilterByHasNicHasHost": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s.%s)`, ipaddressresource.EdgeNic, hostnicresource.EdgeHost),
			},
			resources: []*network_v1.IPAddressResource{ipaddress1, ipaddress2},
			valid:     true,
		},
		"FilterLimit": {
			in: &inv_v1.ResourceFilter{
				Offset: 0,
				Limit:  2,
			},
			resources: []*network_v1.IPAddressResource{ipaddress1, ipaddress2},
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Ipaddress{}} // Set the resource kind
			findres, err := inv_testing.TestClients[inv_testing.RMClient].Find(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterIPAddress() failed: %s", err)
					t.FailNow()
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterIPAddress() succeeded but should have failed")
					t.FailNow()
				}
			}

			// only compare if valid test with non-zero returned response and hasn't failed, otherwise may segfault
			if !t.Failed() && tc.valid {
				if len(findres.Resources) != len(tc.resources) {
					t.Errorf("Expected to obtain %d Resource IDs, but obtained back %d Resource IDs",
						len(tc.resources), len(findres.Resources))
				}

				resIDs := inv_testing.GetSortedResourceIDSlice(tc.resources)
				inv_testing.SortHasResourceIDAndTenantID(findres.Resources)

				if !reflect.DeepEqual(resIDs, findres.Resources) {
					t.Errorf(
						"FilterIPAddress() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}

			listres, err := inv_testing.TestClients[inv_testing.RMClient].List(ctx, tc.in)

			if err != nil {
				if tc.valid {
					t.Errorf("ListIPAddress() failed: %s", err)
					t.FailNow()
				}
			} else {
				if !tc.valid {
					t.Errorf("ListIPAddress() succeeded but should have failed")
					t.FailNow()
				}
			}

			// only compare if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				resources := make([]*network_v1.IPAddressResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetIpaddress())
				}
				inv_testing.OrderByResourceID(resources)
				inv_testing.OrderByResourceID(tc.resources)
				for i, expected := range tc.resources {
					if eq, diff := inv_testing.ProtoEqualOrDiff(expected, resources[i]); !eq {
						t.Errorf("ListIPAddress() data not equal: %v", diff)
					}
				}
			}
		})
	}
}

func Test_Events_DeleteIPAddress(t *testing.T) {
	host := inv_testing.CreateHost(t, nil, nil)
	hostNic := inv_testing.CreateHostNic(t, host)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ipAddressDyn := network_v1.IPAddressResource{
		Address:      "10.0.0.1/24",
		CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
		ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_DYNAMIC,
		Nic:          hostNic,
	}
	respDyn, err := inv_testing.TestClients[inv_testing.RMClient].Create(ctx,
		&inv_v1.Resource{
			Resource: &inv_v1.Resource_Ipaddress{Ipaddress: &ipAddressDyn},
		})
	require.NoError(t, err)
	resIDDyn := respDyn.GetIpaddress().GetResourceId()

	ipAddressSta := network_v1.IPAddressResource{
		Address:      "10.1.0.1/24",
		CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
		ConfigMethod: network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_STATIC,
		Nic:          hostNic,
	}
	respSta, err := inv_testing.TestClients[inv_testing.RMClient].Create(ctx,
		&inv_v1.Resource{
			Resource: &inv_v1.Resource_Ipaddress{Ipaddress: &ipAddressSta},
		})
	require.NoError(t, err)
	resIDSta := respSta.GetIpaddress().GetResourceId()

	// Empty both channels
	time.Sleep(1 * time.Second)
	for len(inv_testing.TestClientsEvents[inv_testing.RMClient]) > 0 {
		<-inv_testing.TestClientsEvents[inv_testing.RMClient]
	}
	for len(inv_testing.TestClientsEvents[inv_testing.APIClient]) > 0 {
		<-inv_testing.TestClientsEvents[inv_testing.APIClient]
	}

	_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, resIDDyn)
	require.Error(t, err)

	upRes, err := inv_testing.TestClients[inv_testing.RMClient].Update(ctx,
		resIDDyn,
		&fieldmaskpb.FieldMask{Paths: []string{ipaddressresource.FieldCurrentState}},
		&inv_v1.Resource{
			Resource: &inv_v1.Resource_Ipaddress{
				Ipaddress: &network_v1.IPAddressResource{
					CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_DELETED,
				},
			},
		},
	)
	require.NoError(t, err, "UpdateIPAddress() failed")
	assert.Equal(t, network_v1.IPAddressState_IP_ADDRESS_STATE_DELETED, upRes.GetIpaddress().GetCurrentState())
	assertReceiveEvent(
		t,
		inv_testing.TestClientsEvents[inv_testing.APIClient],
		inv_v1.SubscribeEventsResponse_EVENT_KIND_DELETED,
		inv_v1.ResourceKind_RESOURCE_KIND_IPADDRESS,
		resIDDyn,
	)

	upRes, err = inv_testing.TestClients[inv_testing.RMClient].Update(ctx,
		resIDSta,
		&fieldmaskpb.FieldMask{
			Paths: []string{"current_state"},
		},
		&inv_v1.Resource{
			Resource: &inv_v1.Resource_Ipaddress{
				Ipaddress: &network_v1.IPAddressResource{
					CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED,
				},
			},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, network_v1.IPAddressState_IP_ADDRESS_STATE_CONFIGURED, upRes.GetIpaddress().GetCurrentState())
	assertReceiveEvent(
		t,
		inv_testing.TestClientsEvents[inv_testing.APIClient],
		inv_v1.SubscribeEventsResponse_EVENT_KIND_UPDATED,
		inv_v1.ResourceKind_RESOURCE_KIND_IPADDRESS,
		resIDSta,
	)

	upRes, err = inv_testing.TestClients[inv_testing.RMClient].Update(ctx,
		resIDSta,
		&fieldmaskpb.FieldMask{Paths: []string{ipaddressresource.FieldCurrentState}},
		&inv_v1.Resource{
			Resource: &inv_v1.Resource_Ipaddress{
				Ipaddress: &network_v1.IPAddressResource{
					CurrentState: network_v1.IPAddressState_IP_ADDRESS_STATE_DELETED,
				},
			},
		},
	)
	require.NoError(t, err, "UpdateIPAddress() failed")
	assert.Equal(t, network_v1.IPAddressState_IP_ADDRESS_STATE_DELETED, upRes.GetIpaddress().GetCurrentState())
	assertReceiveEvent(
		t,
		inv_testing.TestClientsEvents[inv_testing.APIClient],
		inv_v1.SubscribeEventsResponse_EVENT_KIND_DELETED,
		inv_v1.ResourceKind_RESOURCE_KIND_IPADDRESS,
		resIDSta,
	)
}

func Test_NestedFilterIpAddress(t *testing.T) {
	// We explicitly set the edges again for the nested eager loading
	region := inv_testing.CreateRegion(t, nil)
	site1 := inv_testing.CreateSite(t, region, nil)
	host1 := inv_testing.CreateHost(t, site1, nil)
	host1.Site = site1
	host2 := inv_testing.CreateHost(t, site1, nil)
	host2.Site = site1
	host3 := inv_testing.CreateHost(t, nil, nil)
	hostNic1 := inv_testing.CreateHostNic(t, host1)
	hostNic1.Host = host1
	hostNic2 := inv_testing.CreateHostNic(t, host2)
	hostNic2.Host = host2
	hostNic3 := inv_testing.CreateHostNic(t, host3)
	hostNic3.Host = host3
	ipaddress1 := inv_testing.CreateIPAddress(t, hostNic1, true)
	ipaddress1.Nic = hostNic1
	ipaddress2 := inv_testing.CreateIPAddress(t, hostNic2, true)
	ipaddress2.Nic = hostNic2
	ipaddress3 := inv_testing.CreateIPAddress(t, hostNic3, true)
	ipaddress3.Nic = hostNic3

	testcases := map[string]struct {
		in                *inv_v1.ResourceFilter
		resources         []*network_v1.IPAddressResource
		valid             bool
		expectedCodeError codes.Code
	}{
		"FilterByHostID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s = %q`, ipaddressresource.EdgeNic, hostnicresource.EdgeHost,
					hostresource.FieldResourceID, host1.GetResourceId()),
			},
			resources: []*network_v1.IPAddressResource{ipaddress1},
			valid:     true,
		},
		"FilterByIPInSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s = %q`, ipaddressresource.EdgeNic,
					hostnicresource.EdgeHost, hostresource.EdgeSite,
					siteresource.FieldResourceID, site1.GetResourceId()),
			},
			resources: []*network_v1.IPAddressResource{ipaddress1, ipaddress2},
			valid:     true,
		},
		"FilterByIPInNoSite": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`NOT has(%s.%s.%s)`, ipaddressresource.EdgeNic,
					hostnicresource.EdgeHost, hostresource.EdgeSite),
			},
			resources: []*network_v1.IPAddressResource{ipaddress3},
			valid:     true,
		},
		"FailTooDeep": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s.%s.%s.%s.%s.%s = %q`, ipaddressresource.EdgeNic,
					hostnicresource.EdgeHost, hostresource.EdgeSite, siteresource.EdgeRegion,
					regionresource.EdgeParentRegion, regionresource.FieldResourceID, region.GetResourceId()),
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

			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Ipaddress{}} // Set the resource kind

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
						"FilterIPAddress() failed - want: %s, got: %s",
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

				resources := make([]*network_v1.IPAddressResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetIpaddress())
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

func Test_IPAddressEnumsMap(t *testing.T) {
	v, err := store.IPAddressEnumsMap(ipaddressresource.FieldDesiredState,
		int32(network_v1.IPAddressState_IP_ADDRESS_STATE_ASSIGNED))
	val0, ok0 := v.(ipaddressresource.DesiredState)
	assert.True(t, ok0)
	assert.Nil(t, err)
	assert.Equal(t, "IP_ADDRESS_STATE_ASSIGNED", val0.String())

	v, err = store.IPAddressEnumsMap("invalid_input",
		int32(network_v1.IPAddressConfigMethod_IP_ADDRESS_CONFIG_METHOD_STATIC))
	assert.Error(t, err)
	assert.Nil(t, v)
}

func TestIPAddressMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)
	suite.Run(t, &struct{ mt }{
		mt: mt{
			updateReqClientType: inv_testing.RMClient,
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				host := dao.CreateHost(t, tenantID)
				nic := dao.CreateHostNic(t, tenantID, host)
				ip := dao.CreateIPAddress(t, tenantID, nic, true)
				res, err := util.WrapResource(ip)
				require.NoError(t, err)
				return ip.GetResourceId(), res
			},
		},
	})
}

func TestDeleteResources_IPAddresses(t *testing.T) {
	suite.Run(t, &hardDeleteAllResourcesSuite{
		createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
			tenantID := uuid.NewString()
			host := dao.CreateHost(t, tenantID)
			hostNic := dao.CreateHostNic(t, tenantID, host)
			return tenantID, len(
				[]any{
					dao.CreateIPAddress(t, tenantID, hostNic, false),
					dao.CreateIPAddress(t, tenantID, hostNic, false),
					dao.CreateIPAddress(t, tenantID, hostNic, false),
				},
			)
		},
		resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_IPADDRESS,
	})
}
