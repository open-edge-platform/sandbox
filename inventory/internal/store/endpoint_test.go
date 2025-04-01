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
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/providerresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/regionresource"
	"github.com/open-edge-platform/infra-core/inventory/v2/internal/ent/siteresource"
	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	inv_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/inventory/v1"
	network_v1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/network/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	inv_testing "github.com/open-edge-platform/infra-core/inventory/v2/pkg/testing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util"
)

func Test_Create_Get_Delete_Endpoint(t *testing.T) {
	host1 := inv_testing.CreateHost(t, nil, nil)
	host2 := inv_testing.CreateHost(t, nil, nil)
	host3 := &computev1.HostResource{ResourceId: "host-aabbccdd"}

	testcases := map[string]struct {
		in    *network_v1.EndpointResource
		valid bool
	}{
		"CreateGoodEndpoint": {
			in: &network_v1.EndpointResource{
				Name: "Test Endpoint 1",
				Host: host1,
			},
			valid: true,
		},
		"CreateBadEndpointWithInvalidResourceIdSet": {
			// This tests case verifies that create requests with a invalid resource ID
			// already set are rejected.
			in: &network_v1.EndpointResource{
				ResourceId: "endpo-12345678",
				Name:       "Test Endpoint 2",
				Host:       host2,
			},
			valid: false,
		},
		"CreateBadEndpointWithInvalidHost": {
			// This tests case verifies that create requests without Host
			// already set are rejected.
			in: &network_v1.EndpointResource{
				Name: "Test Endpoint 2",
				Host: host3,
			},
			valid: false,
		},
		"CreateBadEndpointWithResourceIdSet": {
			// This tests case verifies that create requests with a resource ID
			// already set are rejected.
			in: &network_v1.EndpointResource{
				ResourceId: "endpoint-12345678",
				Name:       "Test Endpoint 2",
				Host:       host2,
			},
			valid: false,
		},
	}

	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			createresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Endpoint{Endpoint: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			// create

			createdRes, err := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
			endpointResID := createdRes.GetEndpoint().GetResourceId()

			if err != nil {
				if tc.valid {
					t.Errorf("CreateEndpoint() failed: %s", err)
				}
			} else {
				tc.in.ResourceId = createdRes.GetEndpoint().ResourceId // Update with created resource ID.
				tc.in.CreatedAt = createdRes.GetEndpoint().GetCreatedAt()
				tc.in.UpdatedAt = createdRes.GetEndpoint().GetUpdatedAt()
				assertSameResource(t, createresreq, createdRes, nil)
				if !tc.valid {
					t.Errorf("CreateEndpoint() succeeded but should have failed")
				}
			}

			// only get/delete if valid test and hasn't failed otherwise may segfault
			if !t.Failed() && tc.valid {
				// get non-existent first
				_, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, "endpoint-12345678")
				require.Error(t, err)

				// get
				getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, endpointResID)
				require.NoError(t, err, "GetEndpoint() failed")

				// verify data
				if eq, diff := inv_testing.ProtoEqualOrDiff(tc.in, getresp.GetResource().GetEndpoint()); !eq {
					t.Errorf("GetEndpoint() data not equal: %v", diff)
				}

				// delete
				_, err = inv_testing.TestClients[inv_testing.APIClient].Delete(ctx, endpointResID)
				if err != nil {
					t.Errorf("DeleteEndpoint() failed %s", err)
				}
			}
		})
	}
}

func Test_FilterEndpoints(t *testing.T) {
	host1 := inv_testing.CreateHost(t, nil, nil)
	host2 := inv_testing.CreateHost(t, nil, nil)

	cendpResp1 := inv_testing.CreateEndpoint(t, host1)
	cendpResp1.Host = host1
	cendpResp2 := inv_testing.CreateEndpoint(t, host2)
	cendpResp2.Host = host2
	cendpResp3 := inv_testing.CreateEndpoint(t, nil)

	testcases := map[string]struct {
		in        *inv_v1.ResourceFilter
		resources []*network_v1.EndpointResource
		valid     bool
	}{
		"NoFilter": {
			in:        &inv_v1.ResourceFilter{},
			resources: []*network_v1.EndpointResource{cendpResp1, cendpResp2, cendpResp3},
			valid:     true,
		},
		"NoFilterOrderByResourceID": {
			in: &inv_v1.ResourceFilter{
				OrderBy: endpointresource.FieldResourceID,
			},
			resources: []*network_v1.EndpointResource{cendpResp1, cendpResp2, cendpResp3},
			valid:     true,
		},
		"FilterByInvalidResID": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, endpointresource.FieldResourceID, "endpoint-aabbccdd"),
			},
			resources: []*network_v1.EndpointResource{},
			valid:     true,
		},
		"FilterByResourceIdEq": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`%s = %q`, endpointresource.FieldResourceID, cendpResp1.ResourceId),
			},
			resources: []*network_v1.EndpointResource{cendpResp1},
			valid:     true,
		},
		"FilterByHasHost": {
			in: &inv_v1.ResourceFilter{
				Filter: fmt.Sprintf(`has(%s)`, endpointresource.EdgeHost),
			},
			resources: []*network_v1.EndpointResource{cendpResp1, cendpResp2},
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
			valid:     true,
			resources: []*network_v1.EndpointResource{cendpResp1, cendpResp2, cendpResp3},
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
			tc.in.Resource = &inv_v1.Resource{Resource: &inv_v1.Resource_Endpoint{}} // Set the resource kind
			findresreq := tc.in
			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			findres, err := inv_testing.TestClients[inv_testing.APIClient].Find(ctx, findresreq)

			if err != nil {
				if tc.valid {
					t.Errorf("FilterEndpoints() failed: %s", errors.ErrorToStringWithDetails(err))
				}
			} else {
				if !tc.valid {
					t.Errorf("FilterEndpoints() succeeded but should have failed")
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
						"FilterEndpoints() failed - want: %s, got: %s",
						resIDs,
						findres.Resources,
					)
				}
			}
		})
	}
}

func Test_UpdateEndpoint(t *testing.T) {
	host := inv_testing.CreateHost(t, nil, nil)
	host2 := inv_testing.CreateHost(t, nil, nil)

	// create Endpoint to update
	createresreq := &inv_v1.Resource{
		Resource: &inv_v1.Resource_Endpoint{
			Endpoint: &network_v1.EndpointResource{
				Name: "Test Endpoint 4",
				Host: host,
				Kind: "kind",
			},
		},
	}
	// build a context for gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cendpResp, _ := inv_testing.TestClients[inv_testing.APIClient].Create(ctx, createresreq)
	endpointResID := cendpResp.GetEndpoint().GetResourceId()
	t.Cleanup(func() { inv_testing.DeleteResource(t, endpointResID) })

	testcases := map[string]struct {
		in           *network_v1.EndpointResource
		fieldMask    *fieldmaskpb.FieldMask
		valid        bool
		expErrorCode codes.Code
	}{
		"Update1": {
			in: &network_v1.EndpointResource{
				ResourceId: endpointResID,
				Name:       "Updated Name",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{endpointresource.FieldName},
			},
			valid: true,
		},
		"Update2": {
			in: &network_v1.EndpointResource{
				ResourceId: endpointResID,
				Name:       "Updated Name",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{endpointresource.FieldName, "host"},
			},
			valid: true,
		},
		"Update3": {
			in: &network_v1.EndpointResource{
				ResourceId: endpointResID,
				Name:       "Updated Name",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{endpointresource.FieldKind},
			},
			valid: true,
		},
		"UpdateSetNewHost": {
			in: &network_v1.EndpointResource{
				ResourceId: endpointResID,
				Name:       "Updated Name",
				Host:       host2,
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{endpointresource.FieldName, "host"},
			},
			valid: true,
		},
		"UpdateNoFieldMask": {
			in: &network_v1.EndpointResource{
				ResourceId: endpointResID,
				Name:       "Updated Name",
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateInvalidFieldMask1": {
			in: &network_v1.EndpointResource{
				ResourceId: endpointResID,
				Name:       "Updated Name",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{"INVALID_FIELD"},
			},
			valid:        false,
			expErrorCode: codes.InvalidArgument,
		},
		"UpdateResourceIDNotFound": {
			in: &network_v1.EndpointResource{
				ResourceId: "endpoint-12345678",
				Name:       "Updated Name",
			},
			fieldMask: &fieldmaskpb.FieldMask{
				Paths: []string{endpointresource.FieldName},
			},
			valid:        false,
			expErrorCode: codes.NotFound,
		},
	}
	for tcname, tc := range testcases {
		t.Run(tcname, func(t *testing.T) {
			updateresreq := &inv_v1.Resource{
				Resource: &inv_v1.Resource_Endpoint{Endpoint: tc.in},
			}

			// build a context for gRPC
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			upRes, err := inv_testing.TestClients[inv_testing.APIClient].Update(
				ctx,
				tc.in.GetResourceId(),
				tc.fieldMask,
				updateresreq,
			)

			if !tc.valid {
				require.Error(t, err, "UpdateResource() succeeded but should have failed")
				assert.Equal(t, tc.expErrorCode, status.Code(err))
				assert.Nil(t, upRes)
				return
			}
			require.NoErrorf(t, err, "UpdateResource() failed: %s", err)

			// Validate returned resource
			assertSameResource(t, updateresreq, upRes, tc.fieldMask)

			// validate update via a get
			getresp, err := inv_testing.TestClients[inv_testing.APIClient].Get(ctx, tc.in.GetResourceId())
			require.NoError(t, err, "GetResource() failed")

			assertSameResource(t, updateresreq, getresp.GetResource(), tc.fieldMask)
		})
	}
}

func Test_NestedFilterEndpoint(t *testing.T) {
	provider := inv_testing.CreateProvider(t, "Test Provider1")
	host1 := inv_testing.CreateHost(t, nil, provider)
	host2 := inv_testing.CreateHost(t, nil, nil)
	endpoint1 := inv_testing.CreateEndpoint(t, host1)
	endpoint1.Host = host1
	endpoint2 := inv_testing.CreateEndpoint(t, host2)
	endpoint2.Host = host2

	testcases := map[string]struct {
		in                *inv_v1.ResourceFilter
		resources         []*network_v1.EndpointResource
		valid             bool
		expectedCodeError codes.Code
	}{
		"FilterByHostIDMoreFieldsSet": {
			in: &inv_v1.ResourceFilter{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Endpoint{},
				},
				Filter: fmt.Sprintf("%s.%s = %q", endpointresource.EdgeHost, hostresource.FieldResourceID, host2.GetResourceId()),
			},
			resources: []*network_v1.EndpointResource{endpoint2},
			valid:     true,
		},
		"FilterByHostUuid": {
			in: &inv_v1.ResourceFilter{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Endpoint{},
				},
				Filter: fmt.Sprintf("%s.%s = %q", endpointresource.EdgeHost, hostresource.FieldUUID, host1.GetUuid()),
			},
			resources: []*network_v1.EndpointResource{endpoint1},
			valid:     true,
		},
		"FilterByEmptySite": {
			in: &inv_v1.ResourceFilter{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Endpoint{},
				},
				Filter: fmt.Sprintf("NOT has(%s.%s)", endpointresource.EdgeHost, hostresource.EdgeSite),
			},
			resources: []*network_v1.EndpointResource{endpoint1, endpoint2},
			valid:     true,
		},
		"FilterByProviderID": {
			in: &inv_v1.ResourceFilter{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Endpoint{},
				},
				Filter: fmt.Sprintf("%s.%s.%s = %q", endpointresource.EdgeHost, hostresource.EdgeProvider,
					providerresource.FieldResourceID, provider.GetResourceId()),
			},
			resources: []*network_v1.EndpointResource{endpoint1},
			valid:     true,
		},
		"FailTooDeep": {
			in: &inv_v1.ResourceFilter{
				Resource: &inv_v1.Resource{
					Resource: &inv_v1.Resource_Endpoint{},
				},
				Filter: fmt.Sprintf("has(%s.%s.%s.%s.%s.%s)", endpointresource.EdgeHost, hostresource.EdgeSite,
					siteresource.EdgeRegion, regionresource.EdgeParentRegion,
					regionresource.EdgeParentRegion, regionresource.EdgeParentRegion),
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
				require.Len(t, listres.Resources, len(tc.resources))

				resources := make([]*network_v1.EndpointResource, 0, len(listres.Resources))
				for _, r := range listres.Resources {
					resources = append(resources, r.GetResource().GetEndpoint())
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

func TestEndpointMTSanity(t *testing.T) {
	dao := inv_testing.NewInvResourceDAOOrFail(t)

	suite.Run(t, &struct{ mt }{
		mt: mt{
			createResource: func(tenantID string) (string, *inv_v1.Resource) {
				ep := dao.CreateEndpoint(t, tenantID, nil)
				res, err := util.WrapResource(ep)
				require.NoError(t, err)
				return ep.GetResourceId(), res
			},
		},
	})
}

func TestDeleteResources_Endpoints(t *testing.T) {
	suite.Run(t, &struct{ hardDeleteAllResourcesSuite }{
		hardDeleteAllResourcesSuite: hardDeleteAllResourcesSuite{
			createModel: func(dao *inv_testing.InvResourceDAO) (string, int) {
				tenantID := uuid.NewString()
				host := dao.CreateHost(t, tenantID)
				return tenantID, len(
					[]any{
						dao.CreateEndpointNoCleanup(t, tenantID, host),
						dao.CreateEndpointNoCleanup(t, tenantID, host),
						dao.CreateEndpointNoCleanup(t, tenantID, host),
					},
				)
			},
			resourceKind: inv_v1.ResourceKind_RESOURCE_KIND_ENDPOINT,
		},
	})
}
